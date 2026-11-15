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

// RerouteInput is the admin "send this order to a supplier" request (client
// msgs #5–#8, 2026-11-15 video). The target is either one of the product's
// supplier links (LinkID — preferred: nothing is typed) or a raw supplier +
// supplier product ("other supplier product…").
type RerouteInput struct {
	LinkID            string `json:"linkId,omitempty"`
	ProviderID        int    `json:"providerId"`
	UpstreamProductID string `json:"upstreamProductId"`
	// AcknowledgeAtSupplier must be true to re-send an order a supplier may
	// still deliver (orderAtSupplier) — the same confirmation a refund needs.
	AcknowledgeAtSupplier bool `json:"acknowledgeAtSupplier"`
	// AcknowledgeAmbiguous must be true to reroute away from an attempt whose
	// outcome is unknown (a umanage "search:" ref): the old supplier may still
	// deliver, so the admin has to accept the double-delivery risk explicitly.
	AcknowledgeAmbiguous bool `json:"acknowledgeAmbiguous"`
	// AcknowledgeLowBalance sends even though the supplier's balance looks too
	// low for this order (some supplier accounts run on credit).
	AcknowledgeLowBalance bool `json:"acknowledgeLowBalance"`
}

// errServiceUnavailable marks a reroute refusal caused by the supplier's
// catalog being unreachable (HTTP 503): retrying later can succeed.
var errServiceUnavailable = errors.New("service unavailable")

// Reroute error codes (typed so the console can explain each refusal).
func rerouteErr(code, msg string, base error) error {
	return &apperrors.AppError{Code: code, Message: msg, Err: base}
}

// rerouteEligible reports whether an order may be sent to a supplier and, if
// not, why. Two shapes qualify, both still processing:
//
//   - a supplier (api-mode) order, at any time. One a supplier may still
//     deliver (orderAtSupplier) additionally needs the admin's acknowledgement
//     (rerouteCheck → REROUTE_NEEDS_ACK), exactly like a refund;
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
// re-send; a stuck flag stops those). A manual completion, refund or re-send
// then risks double delivery, so the admin must acknowledge it. A stub park
// (neither), a rejected attempt (ref dropped) and a ref-less stuck order are safe.
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

// rerouteTarget is the supplier product an order is about to be sent to.
type rerouteTarget struct {
	Provider int
	Upstream string
	LinkID   string // the product's supplier link, when chosen from the list
}

// reroutePrep is what the order-side checks learned about an order.
type reroutePrep struct {
	idx        int
	fromManual bool
	ref        string
	qty        int
}

// Reroute sends a processing order to a supplier (another one, or the same one
// again) and dispatches it there immediately. Safety:
//   - one guarded claim (ClaimSupplierReroute) pins status + attempt + ref, so
//     it cannot interleave with a settler tick, an admin refund or a double click;
//   - the new attempt gets a fresh upstream idempotency key, and the old ref is
//     archived (no longer polled) — retries within the new attempt stay deduped;
//   - an order a supplier may still deliver needs AcknowledgeAtSupplier (an
//     unknown umanage outcome AcknowledgeAmbiguous): double-delivery risk;
//   - outcomes settle only through the guarded settler paths. A definitive
//     rejection by the new supplier does NOT auto-refund: the order is flagged
//     stuck again so the admin can try another supplier or refund.
//
// The settler's give-up window restarts at the reroute (SupplierAttemptAt).
func (s *SupplierSettler) Reroute(ctx context.Context, id bson.ObjectID, in RerouteInput, actor string) (*Order, error) {
	if strings.TrimSpace(in.LinkID) == "" {
		// Cheap input checks first: no DB read for an obviously bad request.
		if _, err := s.rawTarget(in); err != nil {
			return nil, err
		}
	}
	o, err := s.svc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	prep, err := rerouteCheck(o, in)
	if err != nil {
		return nil, err
	}
	t, err := s.targetFor(ctx, o.Items[prep.idx], in)
	if err != nil {
		return nil, err
	}
	claimed, err := s.claimReroute(ctx, o, prep, t, in, actor, false)
	if err != nil {
		return nil, err
	}
	s.dispatchDetached(ctx, claimed)
	return s.svc.repo.FindByID(ctx, o.ID)
}

// dispatchDetached sends a freshly claimed attempt on a context detached from
// the admin request: closing the tab or a proxy timeout must not abort a call
// the supplier may already be processing. The claim already happened, so a
// failure to persist the outcome is logged, not returned: the claim fallback
// retry clock makes the settler re-dispatch this attempt (same key), and an
// error here would only invite a second reroute with a NEW key (a possible
// double order).
func (s *SupplierSettler) dispatchDetached(ctx context.Context, claimed *Order) {
	dctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rerouteDispatchTimeout)
	defer cancel()
	if err := s.dispatchRerouted(dctx, claimed); err != nil {
		slog.Error("order: reroute dispatch outcome not persisted; settler will retry", "order", claimed.ID.Hex(), "error", err)
	}
}

// rerouteCheck runs every order-side check of a reroute: eligibility, the
// double-delivery acknowledgements and a sendable quantity.
func rerouteCheck(o *Order, in RerouteInput) (reroutePrep, error) {
	idx, fromManual, err := rerouteEligible(o)
	if err != nil {
		return reroutePrep{}, err
	}
	ref := o.Fulfillment.ProviderRef
	ambiguous := strings.HasPrefix(ref, "search:")
	if ambiguous && !in.AcknowledgeAmbiguous {
		return reroutePrep{}, rerouteErr("REROUTE_AMBIGUOUS", "the previous supplier's outcome is unknown and it may still deliver — confirm the double-delivery risk to continue", apperrors.ErrConflict)
	}
	if !ambiguous && orderAtSupplier(o) && !in.AcknowledgeAtSupplier {
		return reroutePrep{}, rerouteErr("REROUTE_NEEDS_ACK", "the current supplier may still deliver this order — confirm the double-delivery risk to send it again", apperrors.ErrConflict)
	}
	qty, ok := supplierQty(o.Items[idx])
	if !ok {
		return reroutePrep{}, rerouteErr("REROUTE_BAD_QUANTITY", "this order's quantity can't be sent to a supplier — complete or refund it by hand", apperrors.ErrBadRequest)
	}
	return reroutePrep{idx: idx, fromManual: fromManual, ref: ref, qty: qty}, nil
}

// rawTarget validates a typed supplier + supplier product.
func (s *SupplierSettler) rawTarget(in RerouteInput) (rerouteTarget, error) {
	upstream := strings.TrimSpace(in.UpstreamProductID)
	if upstream == "" {
		return rerouteTarget{}, rerouteErr("REROUTE_UPSTREAM_REQUIRED", "choose the supplier product to order", apperrors.ErrBadRequest)
	}
	if !s.svc.providers.Has(in.ProviderID) {
		return rerouteTarget{}, rerouteErr("REROUTE_UNKNOWN_SUPPLIER", "that supplier is not configured", apperrors.ErrBadRequest)
	}
	return rerouteTarget{Provider: in.ProviderID, Upstream: upstream}, nil
}

// targetFor resolves where the order goes: one of the line's product supplier
// links (LinkID), else the typed supplier + product.
func (s *SupplierSettler) targetFor(ctx context.Context, it OrderItem, in RerouteInput) (rerouteTarget, error) {
	linkID := strings.TrimSpace(in.LinkID)
	if linkID == "" {
		return s.rawTarget(in)
	}
	links, _, err := s.productLinks(ctx, it)
	if err != nil {
		return rerouteTarget{}, err
	}
	for _, l := range links {
		if l.ID == linkID {
			return s.linkTarget(l)
		}
	}
	return rerouteTarget{}, rerouteErr("REROUTE_LINK_UNKNOWN", "that supplier is no longer linked to the product — reload the order", apperrors.ErrBadRequest)
}

// linkTarget turns a product supplier link into a target, refusing a link
// that can't be sent to.
func (s *SupplierSettler) linkTarget(l product.SupplierLink) (rerouteTarget, error) {
	switch {
	case !l.Enabled:
		return rerouteTarget{}, rerouteErr("REROUTE_LINK_DISABLED", "that supplier is switched off in the product's supplier list — switch it on there first", apperrors.ErrConflict)
	case strings.TrimSpace(l.UpstreamID) == "":
		return rerouteTarget{}, rerouteErr("REROUTE_UPSTREAM_REQUIRED", "that supplier link has no supplier product — link one in the product editor", apperrors.ErrBadRequest)
	case !s.svc.providers.Has(l.Provider):
		return rerouteTarget{}, rerouteErr("REROUTE_UNKNOWN_SUPPLIER", "that supplier is not configured", apperrors.ErrBadRequest)
	}
	return rerouteTarget{Provider: l.Provider, Upstream: l.UpstreamID, LinkID: l.ID}, nil
}

// productLinks returns the supplier links of a line's product (an unmigrated
// single mapping counts as one link) and the active link id. A deleted product
// simply has no links.
func (s *SupplierSettler) productLinks(ctx context.Context, it OrderItem) ([]product.SupplierLink, string, error) {
	if it.ProductID.IsZero() {
		return nil, "", nil
	}
	p, err := s.svc.products.FindByID(ctx, it.ProductID.Hex())
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, "", nil
		}
		return nil, "", err
	}
	links, active := product.EffectiveLinks(p)
	return links, active, nil
}

// claimReroute checks the target supplier and claims the new attempt (it does
// not dispatch). bulk only tags the audit entry.
func (s *SupplierSettler) claimReroute(ctx context.Context, o *Order, prep reroutePrep, t rerouteTarget, in RerouteInput, actor string, bulk bool) (*Order, error) {
	if err := s.checkRerouteTarget(ctx, t.Provider, t.Upstream, prep.qty, in.AcknowledgeLowBalance); err != nil {
		return nil, err
	}
	old := o.Items[prep.idx]
	now := time.Now().UTC()
	note := fmt.Sprintf("sent to another supplier (#%d) by admin", t.Provider)
	if prep.fromManual {
		note = fmt.Sprintf("sent to supplier (#%d) by admin", t.Provider)
	} else if old.FulfillmentProvider != nil && *old.FulfillmentProvider == t.Provider {
		note = fmt.Sprintf("sent again to supplier (#%d) by admin", t.Provider)
	}
	claim := RerouteClaim{
		OrderID:         o.ID,
		ItemIndex:       prep.idx,
		ExpectedAttempt: o.Fulfillment.SupplierAttempt,
		ExpectedRef:     prep.ref,
		FromManual:      prep.fromManual,
		NewProvider:     t.Provider,
		NewUpstreamID:   t.Upstream,
		NewOrderUUID:    bson.NewObjectID().Hex(),
		Event:           TimelineEvent{Status: "processing", Note: note, At: now},
	}
	if !prep.fromManual {
		// A manual line was never sent upstream: nothing to archive.
		archivedRef := prep.ref
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
		summary := map[string]any{
			"fromMode":     string(itemMode(old)),
			"fromProvider": providerLabel(old.FulfillmentProvider), "fromUpstream": old.UpstreamProductID,
			"fromRef": prep.ref, "toProvider": t.Provider, "toUpstream": t.Upstream,
			"attempt": claimed.Fulfillment.SupplierAttempt,
		}
		if t.LinkID != "" {
			summary["linkId"] = t.LinkID
		}
		if in.AcknowledgeAtSupplier && orderAtSupplier(o) {
			summary["acknowledgedAtSupplier"] = true
		}
		if bulk {
			summary["bulk"] = true
		}
		// Actor fields left empty: Record fills them from the admin's JWT.
		s.rec.Record(ctx, audit.Entry{
			Action:     audit.ActionOrderSupplierReroute,
			TargetType: "order",
			TargetID:   o.ID.Hex(),
			Summary:    summary,
		})
	}
	return claimed, nil
}

// checkRerouteTarget refuses a reroute the target supplier would certainly
// refuse, before anything is claimed: a product id that isn't in its catalog
// (the 2026-11-14 "1045" mistake), a product it marks unavailable, a quantity
// it doesn't sell, or (unless acknowledged) a balance below this order's cost.
// A supplier whose adapter can't list its catalog is not checked; a catalog
// that fails to load blocks the send (the admin retries shortly — the cache
// serves a recent copy when one exists). A failed balance probe never blocks.
func (s *SupplierSettler) checkRerouteTarget(ctx context.Context, providerID int, upstream string, qty int, ackLowBalance bool) error {
	if s.probes == nil {
		return nil
	}
	cp, found, _, err := s.probes.Lookup(ctx, providerID, upstream)
	switch {
	case errors.Is(err, provider.ErrNotImplemented):
		return nil
	case err != nil:
		slog.Warn("order: reroute catalog check failed", "provider", providerID, "error", err)
		return rerouteErr("REROUTE_CATALOG_UNAVAILABLE", "couldn't load that supplier's product list to check the product — try again in a minute", errServiceUnavailable)
	case !found:
		return rerouteErr("REROUTE_UPSTREAM_UNKNOWN", fmt.Sprintf("product %q is not in that supplier's catalog — pick the product from the list", upstream), apperrors.ErrBadRequest)
	case !cp.Available:
		return rerouteErr("REROUTE_UPSTREAM_UNAVAILABLE", "that supplier marks this product unavailable right now — choose another supplier", apperrors.ErrConflict)
	case !provider.AcceptsQty(cp, qty):
		return rerouteErr("REROUTE_QTY_REJECTED", fmt.Sprintf("this order sends quantity %d, which that supplier product doesn't accept", qty), apperrors.ErrBadRequest)
	}
	if cp.Price <= 0 || ackLowBalance {
		return nil
	}
	acc, err := s.probes.Balance(ctx, providerID)
	if err != nil {
		return nil
	}
	cost := cp.Price * float64(qty)
	// A negative balance means the account runs on credit (jentel does): we
	// can't tell its limit, so only a non-negative balance below the cost is
	// flagged.
	if acc.Balance >= 0 && acc.Balance < cost {
		return rerouteErr("REROUTE_LOW_BALANCE", fmt.Sprintf("that supplier's balance (%.2f %s) is below this order's cost (%.2f %s) — top it up, or confirm to send anyway", acc.Balance, acc.Currency, cost, cp.Currency), apperrors.ErrConflict)
	}
	return nil
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
