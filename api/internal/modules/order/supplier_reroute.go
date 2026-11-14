package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// rerouteDispatchTimeout bounds the immediate dispatch to the new supplier.
const rerouteDispatchTimeout = 2 * time.Minute

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

// rerouteEligible reports whether an order may be sent to a supplier and, if
// not, why. Two shapes qualify, both still processing:
//
//   - a supplier (api-mode) order that is NOT actively being handled upstream:
//     flagged stuck, or parked with no upstream ref (environmental retry park
//     or an unwired-supplier stub park). An order the supplier accepted and is
//     still working on must wait for the stuck flag;
//   - a manual order (fromManual) that was never sent to a supplier: exactly
//     one line, manual_operator account credit. Excluded on purpose: money
//     transfers (the recipient is never sent upstream), bridge recharges (the
//     device completes them unpinned), inventory/mixed carts and multi-line
//     orders (a supplier accept completes the WHOLE order).
func rerouteEligible(o *Order) (itemIndex int, fromManual bool, err error) {
	if o.Status != OrderStatusProcessing {
		return -1, false, rerouteErr("REROUTE_NOT_PROCESSING", "only orders that are still processing can be sent to a supplier", apperrors.ErrConflict)
	}
	for i, it := range o.Items {
		if it.FulfillmentMode == string(product.FulfillmentModeAPI) {
			if o.Fulfillment.SupplierStuckAt == nil && o.Fulfillment.ProviderRef != "" {
				return -1, false, rerouteErr("REROUTE_NOT_STUCK", "the supplier accepted this order and is still working on it — wait until it is flagged stuck", apperrors.ErrConflict)
			}
			return i, false, nil
		}
	}
	if manualSupplierEligible(o) {
		return 0, true, nil
	}
	return -1, false, rerouteErr("REROUTE_NOT_SUPPLIER_ORDER", "only supplier orders and single-item account-credit orders can be sent to a supplier", apperrors.ErrBadRequest)
}

// orderAtSupplier reports whether a supplier may still deliver o: a processing
// order with an api-mode line that either carries an upstream ref (accepted,
// still polled — EVEN when flagged stuck for slowness, since the supplier can
// still deliver late) or a live retry clock (a dispatch the settler will
// re-send; a stuck flag stops those). A manual completion or refund then risks
// double delivery, so the admin must acknowledge it. A stub park (neither), a
// rejected attempt (ref dropped) and a ref-less stuck order are safe.
//
// notAtSupplierFilter is the SAME rule as a Mongo filter (its negation); keep
// the two in step — TestAtSupplierRule_Mongo checks them against shared fixtures.
func orderAtSupplier(o *Order) bool {
	if o.Status != OrderStatusProcessing {
		return false
	}
	f := o.Fulfillment
	live := f.ProviderRef != "" || (f.SupplierNextRetryAt != nil && f.SupplierStuckAt == nil)
	if !live {
		return false
	}
	for _, it := range o.Items {
		if it.FulfillmentMode == string(product.FulfillmentModeAPI) {
			return true
		}
	}
	return false
}

// notAtSupplierFilter matches exactly the orders orderAtSupplier rejects.
func notAtSupplierFilter() bson.E {
	return bson.E{Key: "$or", Value: bson.A{
		bson.D{{Key: "status", Value: bson.D{{Key: "$ne", Value: OrderStatusProcessing}}}},
		bson.D{{Key: "items.fulfillmentMode", Value: bson.D{{Key: "$ne", Value: "api"}}}},
		bson.D{
			{Key: "fulfillment.providerRef", Value: bson.D{{Key: "$in", Value: bson.A{"", nil}}}},
			{Key: "$or", Value: bson.A{
				bson.D{{Key: "fulfillment.supplierNextRetryAt", Value: nil}},
				bson.D{{Key: "fulfillment.supplierStuckAt", Value: bson.D{{Key: "$ne", Value: nil}}}},
			}},
		},
	}}
}

// manualSupplierEligible is the manual half of rerouteEligible: a single
// manual_operator account-credit line with no supplier history at all.
func manualSupplierEligible(o *Order) bool {
	if len(o.Items) != 1 {
		return false
	}
	it := o.Items[0]
	if itemMode(it) != product.FulfillmentModeManualOperator ||
		product.FulfillmentType(it.FulfillmentType) != product.FulfillmentCredit {
		return false
	}
	f := o.Fulfillment
	return f.SupplierAttempt == 0 && f.ProviderRef == "" && len(f.SupplierAttempts) == 0
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
	idx, fromManual, err := rerouteEligible(o)
	if err != nil {
		return nil, err
	}
	ref := o.Fulfillment.ProviderRef
	if strings.HasPrefix(ref, "search:") && !in.AcknowledgeAmbiguous {
		return nil, rerouteErr("REROUTE_AMBIGUOUS", "the previous supplier's outcome is unknown and it may still deliver — confirm the double-delivery risk to continue", apperrors.ErrConflict)
	}
	if _, ok := supplierQty(o.Items[idx]); !ok {
		return nil, rerouteErr("REROUTE_BAD_QUANTITY", "this order's quantity can't be sent to a supplier — complete or refund it by hand", apperrors.ErrBadRequest)
	}

	old := o.Items[idx]
	now := time.Now().UTC()
	claim := RerouteClaim{
		OrderID:         o.ID,
		ItemIndex:       idx,
		ExpectedAttempt: o.Fulfillment.SupplierAttempt,
		ExpectedRef:     ref,
		FromManual:      fromManual,
		NewProvider:     in.ProviderID,
		NewUpstreamID:   upstream,
		NewOrderUUID:    bson.NewObjectID().Hex(),
		Event: TimelineEvent{
			Status: "processing",
			Note:   fmt.Sprintf("sent to another supplier (#%d) by admin", in.ProviderID),
			At:     now,
		},
	}
	if fromManual {
		// Never sent upstream before: no attempt to archive.
		claim.Event.Note = fmt.Sprintf("sent to supplier (#%d) by admin", in.ProviderID)
	} else {
		archivedRef := ref
		if archivedRef == "" {
			// A rejected attempt's ref was moved aside so it isn't polled.
			archivedRef = o.Fulfillment.RejectedProviderRef
		}
		claim.Archived = &SupplierAttempt{
			Provider:          old.FulfillmentProvider,
			UpstreamProductID: old.UpstreamProductID,
			ProviderRef:       archivedRef,
			OrderUUID:         supplierOrderUUID(o),
			StuckAt:           o.Fulfillment.SupplierStuckAt,
			ReroutedAt:        now,
			ReroutedBy:        actor,
		}
	}
	claimed, err := s.svc.repo.ClaimSupplierReroute(ctx, claim)
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
				"fromMode":     string(itemMode(old)),
				"fromProvider": providerLabel(old.FulfillmentProvider), "fromUpstream": old.UpstreamProductID,
				"fromRef": ref, "toProvider": in.ProviderID, "toUpstream": upstream,
				"attempt": claimed.Fulfillment.SupplierAttempt,
			},
		})
	}

	// Dispatch on a context detached from the admin request: closing the tab
	// or a proxy timeout must not abort a call the supplier may already be
	// processing. The claim already happened, so a failure to persist the
	// outcome is logged, not returned: the claim fallback retry clock makes
	// the settler re-dispatch this attempt (same key), and a 500 here would
	// only invite a second reroute with a NEW key (a possible double order).
	dctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rerouteDispatchTimeout)
	defer cancel()
	if err := s.dispatchRerouted(dctx, claimed); err != nil {
		slog.Error("order: reroute dispatch outcome not persisted; settler will retry", "order", o.ID.Hex(), "error", err)
	}
	return s.svc.repo.FindByID(ctx, o.ID)
}

// dispatchRerouted sends the freshly claimed attempt to its new supplier and
// settles the outcome through the guarded paths.
func (s *SupplierSettler) dispatchRerouted(ctx context.Context, o *Order) error {
	provID, in, ok := supplierFulfillInput(o)
	if !ok {
		return s.flagStuck(ctx, o, "rerouted order has no sendable supplier line")
	}
	res, err := s.svc.providers.Resolve(provID).Fulfill(ctx, in)
	switch {
	case err == nil:
		return s.settleAccept(ctx, o, res.Reference, res.Codes, "completed by the new supplier")
	case errors.Is(err, provider.ErrPending) && res.Reference != "":
		return s.svc.repo.SetSupplierRef(ctx, o.ID, o.Fulfillment.SupplierAttempt, res.Reference,
			TimelineEvent{Status: "processing", Note: "awaiting the new supplier", At: time.Now().UTC()})
	case errors.Is(err, provider.ErrPending), errors.Is(err, provider.ErrUnavailable),
		errors.Is(err, provider.ErrNotImplemented), isCtxErr(err):
		// Environmental: the settler retries on the normal backoff (same new key).
		// (Pending with no reference cannot be polled, and a cancelled or timed-out
		// call is never a definitive rejection: both retry.)
		return s.svc.repo.BumpSupplierRetry(ctx, o.ID, o.Fulfillment.SupplierAttempt, 1, time.Now().UTC().Add(backoffDelay(1)))
	default:
		// A definitive rejection by the NEW supplier: no automatic refund — the
		// admin just chose this route (rejectAdminRoute, shared with the
		// settler's later rejections).
		return s.rejectAdminRoute(ctx, o, "", "rejected by the new supplier", err.Error())
	}
}

func providerLabel(id *int) any {
	if id == nil {
		return nil
	}
	return *id
}
