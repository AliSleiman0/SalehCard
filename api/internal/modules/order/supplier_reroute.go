package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// RerouteInput is the admin "send this stuck order to another supplier"
// request (client msgs #5–#8).
type RerouteInput struct {
	ProviderID        int    `json:"providerId"`
	UpstreamProductID string `json:"upstreamProductId"`
	// AcknowledgeAmbiguous must be true to reroute away from an attempt whose
	// outcome is unknown (a umanage "search:" ref): the old supplier may still
	// deliver, so the admin has to accept the double-delivery risk explicitly.
	AcknowledgeAmbiguous bool `json:"acknowledgeAmbiguous"`
}

// Reroute error codes (typed so the console can explain each refusal).
func rerouteErr(code, msg string, base error) error {
	return &apperrors.AppError{Code: code, Message: msg, Err: base}
}

// rerouteEligible reports whether an order may be sent to another supplier and,
// if not, why. Only a processing api-mode order that is NOT actively being
// handled upstream qualifies: flagged stuck, or parked with no upstream ref
// (environmental retry park or an unwired-supplier stub park). An order the
// supplier accepted and is still working on must wait for the stuck flag.
func rerouteEligible(o *Order) (itemIndex int, err error) {
	if o.Status != OrderStatusProcessing {
		return -1, rerouteErr("REROUTE_NOT_PROCESSING", "only orders that are still processing can be sent to another supplier", apperrors.ErrConflict)
	}
	itemIndex = -1
	for i, it := range o.Items {
		if it.FulfillmentMode == string(product.FulfillmentModeAPI) {
			itemIndex = i
			break
		}
	}
	if itemIndex < 0 {
		return -1, rerouteErr("REROUTE_NOT_SUPPLIER_ORDER", "this order is not fulfilled by a supplier", apperrors.ErrBadRequest)
	}
	if o.Fulfillment.SupplierStuckAt == nil && o.Fulfillment.ProviderRef != "" {
		return -1, rerouteErr("REROUTE_NOT_STUCK", "the supplier accepted this order and is still working on it — wait until it is flagged stuck", apperrors.ErrConflict)
	}
	return itemIndex, nil
}

// Reroute moves a stuck api-mode order to another supplier and dispatches it
// there immediately. Safety:
//   - one guarded claim (ClaimSupplierReroute) pins status + attempt + ref, so
//     it cannot interleave with a settler tick, an admin refund or a double click;
//   - the new attempt gets a fresh upstream idempotency key, and the old ref is
//     archived (no longer polled) — retries within the new attempt stay deduped;
//   - outcomes settle only through the guarded settler paths. A definitive
//     rejection by the new supplier does NOT auto-refund: the order is flagged
//     stuck again so the admin can try another supplier or refund.
//
// The settler's give-up window restarts at the reroute (SupplierAttemptAt).
func (s *SupplierSettler) Reroute(ctx context.Context, id bson.ObjectID, in RerouteInput, actor string) (*Order, error) {
	upstream := strings.TrimSpace(in.UpstreamProductID)
	if upstream == "" {
		return nil, rerouteErr("REROUTE_UPSTREAM_REQUIRED", "choose the supplier product to order", apperrors.ErrBadRequest)
	}
	if !s.svc.providers.Has(in.ProviderID) {
		return nil, rerouteErr("REROUTE_UNKNOWN_SUPPLIER", "that supplier is not configured", apperrors.ErrBadRequest)
	}

	o, err := s.svc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	idx, err := rerouteEligible(o)
	if err != nil {
		return nil, err
	}
	ref := o.Fulfillment.ProviderRef
	if strings.HasPrefix(ref, "search:") && !in.AcknowledgeAmbiguous {
		return nil, rerouteErr("REROUTE_AMBIGUOUS", "the previous supplier's outcome is unknown and it may still deliver — confirm the double-delivery risk to continue", apperrors.ErrConflict)
	}

	old := o.Items[idx]
	now := time.Now().UTC()
	claimed, err := s.svc.repo.ClaimSupplierReroute(ctx, RerouteClaim{
		OrderID:         o.ID,
		ItemIndex:       idx,
		ExpectedAttempt: o.Fulfillment.SupplierAttempt,
		ExpectedRef:     ref,
		NewProvider:     in.ProviderID,
		NewUpstreamID:   upstream,
		NewOrderUUID:    bson.NewObjectID().Hex(),
		Archived: SupplierAttempt{
			Provider:          old.FulfillmentProvider,
			UpstreamProductID: old.UpstreamProductID,
			ProviderRef:       ref,
			OrderUUID:         supplierOrderUUID(o),
			StuckAt:           o.Fulfillment.SupplierStuckAt,
			ReroutedAt:        now,
			ReroutedBy:        actor,
		},
		Event: TimelineEvent{
			Status: "processing",
			Note:   fmt.Sprintf("sent to another supplier (#%d) by admin", in.ProviderID),
			At:     now,
		},
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			return nil, rerouteErr("REROUTE_CONFLICT", "the order changed while you were rerouting it — reload and try again", apperrors.ErrConflict)
		}
		return nil, err
	}

	if s.rec != nil {
		// Actor fields left empty: Record fills them from the admin's JWT.
		s.rec.Record(ctx, audit.Entry{
			Action:     audit.ActionOrderSupplierReroute,
			TargetType: "order",
			TargetID:   o.ID.Hex(),
			Summary: map[string]any{
				"fromProvider": providerLabel(old.FulfillmentProvider), "fromUpstream": old.UpstreamProductID,
				"fromRef": ref, "toProvider": in.ProviderID, "toUpstream": upstream,
				"attempt": claimed.Fulfillment.SupplierAttempt,
			},
		})
	}

	if err := s.dispatchRerouted(ctx, claimed); err != nil {
		return nil, err
	}
	return s.svc.repo.FindByID(ctx, o.ID)
}

// dispatchRerouted sends the freshly claimed attempt to its new supplier and
// settles the outcome through the guarded paths.
func (s *SupplierSettler) dispatchRerouted(ctx context.Context, o *Order) error {
	provID, in, ok := supplierFulfillInput(o)
	if !ok {
		return s.flagStuck(ctx, o, "rerouted order has no api-mode line")
	}
	res, err := s.svc.providers.Resolve(provID).Fulfill(ctx, in)
	switch {
	case err == nil:
		return s.settleAccept(ctx, o, res.Reference, res.Codes, "completed by the new supplier")
	case errors.Is(err, provider.ErrPending):
		return s.svc.repo.SetSupplierRef(ctx, o.ID, res.Reference,
			TimelineEvent{Status: "processing", Note: "awaiting the new supplier", At: time.Now().UTC()})
	case errors.Is(err, provider.ErrUnavailable), errors.Is(err, provider.ErrNotImplemented):
		// Environmental: the settler retries on the normal backoff (same new key).
		return s.svc.repo.BumpSupplierRetry(ctx, o.ID, 1, time.Now().UTC().Add(backoffDelay(1)))
	default:
		// A definitive rejection by the NEW supplier: no automatic refund — the
		// admin just chose this route, so flag it stuck again and let them try
		// another supplier or refund. The upstream error goes to the audit log
		// only (the timeline is customer-visible).
		s.record(ctx, audit.ActionOrderSupplierSettle, o.ID, map[string]any{
			"outcome": "rejected_after_reroute", "error": err.Error(), "attempt": o.Fulfillment.SupplierAttempt,
		})
		return s.flagStuck(ctx, o, "rejected by the new supplier")
	}
}

func providerLabel(id *int) any {
	if id == nil {
		return nil
	}
	return *id
}
