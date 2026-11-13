package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// supplierBackoff is the re-dispatch schedule for ref-less unavailable parks;
// past the last entry the final delay repeats (hourly) until the give-up
// window flags the order stuck.
var supplierBackoff = []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour}

const (
	// supplierBatchLimit caps each settler pass; leftovers roll to the next tick.
	supplierBatchLimit = 100
	// supplierActor is the audit actor for every settler-driven mutation.
	supplierActor = "system:supplier-settler"

	defaultSettlerInterval = 60 * time.Second
	defaultSettlerGiveUp   = 24 * time.Hour
)

// backoffDelay returns the wait before re-dispatch attempt n (1-based),
// clamped to the schedule's last entry.
func backoffDelay(n int) time.Duration {
	if n < 1 {
		n = 1
	}
	if n > len(supplierBackoff) {
		n = len(supplierBackoff)
	}
	return supplierBackoff[n-1]
}

// supplierFulfillInput rebuilds the upstream dispatch input from a persisted
// order's api-mode line — shared by placement-time dispatch (fulfillAPI) and
// the settler's re-dispatch, so both send the identical request. ok is false
// when the order carries no api-mode line. The order id doubles as the
// upstream idempotency key (order_uuid), which is what makes re-dispatch safe:
// an original request that actually landed is deduped, not double-charged.
// Sensitive inputs never reach here — resolveOrderFields already dropped them
// from the item snapshot (a sensitive value can only ride via PlayerID).
// ok is also false when the line's quantity can't be sent (see supplierQty):
// callers park/flag such an order for an admin instead of dispatching it.
func supplierFulfillInput(o *Order) (provID *int, in provider.FulfillInput, ok bool) {
	for _, it := range o.Items {
		if it.FulfillmentMode != string(product.FulfillmentModeAPI) {
			continue
		}
		qty, qok := supplierQty(it)
		if !qok {
			return it.FulfillmentProvider, provider.FulfillInput{}, false
		}
		fields := make(map[string]string, len(it.Fields))
		for _, f := range it.Fields {
			fields[f.Key] = f.Value
		}
		return it.FulfillmentProvider, provider.FulfillInput{
			ProductID:  it.ProductID.Hex(),
			UpstreamID: it.UpstreamProductID,
			PlayerID:   it.PlayerID,
			Fields:     fields,
			Qty:        qty,
			OrderUUID:  supplierOrderUUID(o),
		}, true
	}
	return nil, provider.FulfillInput{}, false
}

// maxSupplierQty bounds the quantity sent upstream (a sanity cap far above
// any real coin amount; also keeps the int conversion safe).
const maxSupplierQty = 1_000_000_000

// supplierQty is the quantity a supplier is asked for. A per-unit line (coins,
// live-app credit) carries the customer's typed amount in Amount with Qty == 1:
// the supplier sells those units, so it gets the amount — sending the cart Qty
// (1) was the "Qty is Small" rejection seen in prod on 2026-11-13. Any other
// line sends the cart Qty. ok is false for an amount that isn't a positive
// whole number within the cap (placement already enforces that, so this only
// guards corrupt data — never send qty 0 or a truncated amount).
func supplierQty(it OrderItem) (int, bool) {
	if it.Amount == nil {
		return it.Qty, it.Qty > 0
	}
	a := *it.Amount
	if a < 1 || a > maxSupplierQty || a != math.Trunc(a) {
		return 0, false
	}
	return int(a), true
}

// supplierOrderUUID is the upstream idempotency key for the order's CURRENT
// attempt: the order id for the original dispatch, a fresh per-attempt key
// after an admin reroute (so the new attempt is new to every supplier, while
// retries within one attempt stay deduped).
func supplierOrderUUID(o *Order) string {
	if o.Fulfillment.SupplierOrderUUID != "" {
		return o.Fulfillment.SupplierOrderUUID
	}
	return o.ID.Hex()
}

// supplierClockStart is when the current upstream attempt began: the reroute
// time when the order was rerouted, else its creation. The give-up window is
// measured from here, so a rerouted old order gets a fresh window.
func supplierClockStart(o *Order) time.Time {
	if t := o.Fulfillment.SupplierAttemptAt; t != nil && t.After(o.CreatedAt) {
		return *t
	}
	return o.CreatedAt
}

// SupplierSettler is the background reconciler for parked api-mode orders
// (DESIGN-SUPPLIERS.md Phase 2). Each tick it (a) polls CheckStatus for orders
// the upstream accepted but hasn't finished (providerRef set) — completing on
// accept, refunding on reject — and (b) re-dispatches ref-less environmental
// parks on a backoff schedule. Orders that outlive the give-up window are
// flagged stuck exactly once and left for the admin (never auto-refunded).
//
// Concurrency contract: every write goes through a status-guarded atomic
// document update (TransitionStatus or the guarded bookkeeping methods) —
// never the unguarded UpdateFulfillment/compensate placement-time paths — so
// the settler can race admin refunds, its own next tick, or another replica,
// and money moves at most once.
type SupplierSettler struct {
	svc      *OrderService
	rec      audit.Recorder
	interval time.Duration
	giveUp   time.Duration
}

// NewSupplierSettler builds the settler; non-positive durations get defaults.
func NewSupplierSettler(svc *OrderService, rec audit.Recorder, interval, giveUp time.Duration) *SupplierSettler {
	if interval <= 0 {
		interval = defaultSettlerInterval
	}
	if giveUp <= 0 {
		giveUp = defaultSettlerGiveUp
	}
	return &SupplierSettler{svc: svc, rec: rec, interval: interval, giveUp: giveUp}
}

// Run ticks until ctx is cancelled, firing one immediate pass at start. The
// interval is jittered (~±10%) so multiple replicas don't poll in lockstep.
func (s *SupplierSettler) Run(ctx context.Context) {
	slog.Info("order: supplier settler started", "interval", s.interval, "giveUp", s.giveUp)
	for {
		s.Tick(ctx)
		jitter := rand.N(s.interval / 5) // uniform in [0, interval/5)
		select {
		case <-ctx.Done():
			slog.Info("order: supplier settler stopped")
			return
		case <-time.After(s.interval - s.interval/10 + jitter):
		}
	}
}

// Tick runs one settler pass. Per-order failures are logged and never abort
// the pass — an interrupted or failed pass simply re-runs next tick (every
// mutation is idempotent by construction).
func (s *SupplierSettler) Tick(ctx context.Context) {
	pollable, err := s.svc.repo.ListSupplierPollable(ctx, supplierBatchLimit)
	if err != nil {
		slog.Error("supplier settler: pollable query failed", "error", err)
	}
	for _, o := range pollable {
		if err := s.pollOne(ctx, o); err != nil {
			slog.Error("supplier settler: poll failed", "order", o.ID.Hex(), "error", err)
		}
	}

	retryable, err := s.svc.repo.ListSupplierRetryable(ctx, time.Now().UTC(), supplierBatchLimit)
	if err != nil {
		slog.Error("supplier settler: retryable query failed", "error", err)
	}
	for _, o := range retryable {
		if err := s.redispatchOne(ctx, o); err != nil {
			slog.Error("supplier settler: re-dispatch failed", "order", o.ID.Hex(), "error", err)
		}
	}
}

// pollOne reconciles one accepted-but-unfinished order by its upstream ref.
func (s *SupplierSettler) pollOne(ctx context.Context, o *Order) error {
	provID, _, _ := supplierFulfillInput(o)
	st, err := s.svc.providers.Resolve(provID).CheckStatus(ctx, o.Fulfillment.ProviderRef)
	if err != nil {
		// Unreachable/unwired upstream is NEVER a reject — leave the order
		// parked and only consider the stuck flag.
		slog.Warn("supplier settler: check failed", "order", o.ID.Hex(), "error", err)
		return s.maybeFlagStuck(ctx, o, "upstream status unavailable past give-up window")
	}
	switch st.State {
	case "accept":
		return s.settleAccept(ctx, o, o.Fulfillment.ProviderRef, st.Codes, "completed by supplier check")
	case "reject":
		if o.Fulfillment.SupplierAttempt > 0 {
			return s.rejectAdminRoute(ctx, o, "rejected by the supplier")
		}
		return s.settleReject(ctx, o, "rejected upstream — refunded")
	default: // "wait" or anything unrecognized: leave parked.
		return s.maybeFlagStuck(ctx, o, "still waiting upstream past give-up window")
	}
}

// settleAccept completes one order through the guarded transition, writing the
// upstream ref and any delivered codes atomically with the status flip. A lost
// race (the order already left processing — e.g. an admin refunded it while
// the supplier delivered) appends a reconciliation note for ops and is
// otherwise a no-op: no status change, no money movement.
func (s *SupplierSettler) settleAccept(ctx context.Context, o *Order, ref string, codes []string, note string) error {
	extra := bson.D{
		{Key: "fulfillment.transferRef", Value: ref},
		{Key: "fulfillment.providerRef", Value: ref},
	}
	if len(codes) > 0 {
		extra = append(extra, bson.E{Key: "fulfillment.deliveredCode", Value: strings.Join(codes, "\n")})
	}
	before, err := s.svc.repo.TransitionSupplierStatus(ctx, o.ID, o.Fulfillment.SupplierAttempt, OrderStatusCompleted,
		TimelineEvent{Status: "completed", Note: note, At: time.Now().UTC()},
		extra,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			_ = s.svc.repo.AppendTimelineEvent(ctx, o.ID, TimelineEvent{
				Status: "info",
				Note:   "supplier delivered after the order was rerouted or left processing — reconcile manually",
				At:     time.Now().UTC(),
			})
			return nil
		}
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil
		}
		return err
	}
	s.svc.ntf.Notify(ctx, before.UserID, orderCompletedNote(before.ID.Hex(), before.Total, before.Currency))
	if s.svc.loyalty != nil {
		s.svc.loyalty.Award(ctx, before.UserID, before.Total)
	}
	s.record(ctx, audit.ActionOrderSupplierSettle, o.ID, map[string]any{
		"outcome": "completed", "providerRef": ref, "codesDelivered": len(codes),
	})
	return nil
}

// settleReject refunds one upstream-rejected order, mirroring the admin
// doRefund ordering exactly — transition FIRST (the double-refund lock; the
// wallet ledger does not dedupe by ref), money only on the winning transition,
// revert on reversal failure so the next tick retries.
//
// Deviation from the design doc (which said processing→failed): reject lands
// on `refunded` — `failed` is reserved for placement-time compensation, and
// `refunded` puts the settler and the admin refund button under the SAME
// transition target/lock, keeping the refund-rate metric honest.
func (s *SupplierSettler) settleReject(ctx context.Context, o *Order, reason string) error {
	before, err := s.svc.repo.TransitionSupplierStatus(ctx, o.ID, o.Fulfillment.SupplierAttempt, OrderStatusRefunded,
		TimelineEvent{Status: "refunded", Note: reason, At: time.Now().UTC()},
		nil,
	)
	if err != nil {
		// Someone else already moved the order (admin refund/completion won) —
		// no wallet call, no-op.
		if errors.Is(err, apperrors.ErrConflict) || errors.Is(err, apperrors.ErrNotFound) {
			return nil
		}
		return err
	}

	if before.Total > 0 {
		var reverseErr error
		switch {
		case before.PaymentMethod == PaymentMethodWallet:
			_, reverseErr = s.svc.wallet.Refund(ctx, before.UserID, before.Total, before.ID.Hex())
		case before.PaymentRef != "" && s.svc.payments != nil:
			if prov, ok := s.svc.payments.For(string(before.PaymentMethod)); ok {
				reverseErr = prov.RefundPayment(ctx, before.PaymentRef)
			}
		}
		if reverseErr != nil {
			// Put the order back so the next tick re-polls, gets reject again,
			// and retries the refund — self-healing, still single-credit (the
			// guarded transition re-arms).
			if _, rerr := s.svc.repo.TransitionStatus(ctx, o.ID,
				[]OrderStatus{OrderStatusRefunded}, before.Status,
				TimelineEvent{Status: "refund_reverted", Note: "charge reversal failed", At: time.Now().UTC()},
				nil,
			); rerr != nil {
				slog.Error("supplier settler: charge reversal failed AND status revert failed — order refunded without credit",
					"order", o.ID.Hex(), "reverseError", reverseErr, "revertError", rerr)
			}
			return fmt.Errorf("supplier settler: refund credit failed: %w", reverseErr)
		}
	}

	s.record(ctx, audit.ActionOrderRefund, o.ID, map[string]any{
		"amount": before.Total, "paymentMethod": string(before.PaymentMethod),
		"fromStatus": string(before.Status), "reason": reason, "by": "supplier-settler",
	})
	data := map[string]string{
		"orderId":  o.ID.Hex(),
		"amount":   fmt.Sprintf("%.2f", before.Total),
		"currency": before.Currency,
		"reason":   reason,
	}
	s.svc.ntf.Notify(ctx, before.UserID, notification.Note{
		Kind:  notification.KindOrderRefunded,
		Title: "Order refunded",
		Body:  fmt.Sprintf("$%.2f was returned to your wallet.", before.Total),
		Data:  data,
	})
	return nil
}

// redispatchOne retries one ref-less environmental park by re-running the
// upstream dispatch with the SAME order_uuid (deduped upstream). Retries are
// bookkeeping-only (no timeline spam); outcomes settle through the guarded
// paths above.
func (s *SupplierSettler) redispatchOne(ctx context.Context, o *Order) error {
	if time.Since(supplierClockStart(o)) > s.giveUp {
		return s.flagStuck(ctx, o, "re-dispatch retries exhausted")
	}
	provID, in, ok := supplierFulfillInput(o)
	if !ok {
		// A retry clock on a non-api order shouldn't exist (or its quantity
		// can't be sent); flag it rather than retrying forever.
		return s.flagStuck(ctx, o, "retryable order has no sendable supplier line")
	}

	res, err := s.svc.providers.Resolve(provID).Fulfill(ctx, in)
	switch {
	case err == nil:
		// Accepted and finished. NEVER the unguarded placement-time success
		// path — an admin may have moved the order since the list query.
		return s.settleAccept(ctx, o, res.Reference, res.Codes, "completed by supplier re-dispatch")
	case errors.Is(err, provider.ErrPending):
		// Accepted, still working: promote to the pollable class.
		if res.Reference == "" {
			// Pending without a reference cannot be polled: retry instead.
			n := o.Fulfillment.SupplierRetryCount + 1
			return s.svc.repo.BumpSupplierRetry(ctx, o.ID, o.Fulfillment.SupplierAttempt, n, time.Now().UTC().Add(backoffDelay(n)))
		}
		return s.svc.repo.SetSupplierRef(ctx, o.ID, o.Fulfillment.SupplierAttempt, res.Reference,
			TimelineEvent{Status: "processing", Note: "awaiting upstream provider", At: time.Now().UTC()})
	case errors.Is(err, provider.ErrUnavailable), errors.Is(err, provider.ErrNotImplemented), isCtxErr(err):
		// Still environmental (or the supplier token was pulled → stub):
		// quiet backoff bump.
		n := o.Fulfillment.SupplierRetryCount + 1
		return s.svc.repo.BumpSupplierRetry(ctx, o.ID, o.Fulfillment.SupplierAttempt, n, time.Now().UTC().Add(backoffDelay(n)))
	default:
		// The panel adapter wraps everything ambiguous in ErrUnavailable, so a
		// plain error here is a definitive order-level rejection; upstream
		// idempotency rules out "the original landed and this is a dup error".
		if o.Fulfillment.SupplierAttempt > 0 {
			return s.rejectAdminRoute(ctx, o, "rejected by the supplier on retry: "+err.Error())
		}
		return s.settleReject(ctx, o, "rejected upstream on re-dispatch — refunded")
	}
}

// maybeFlagStuck flags o once its age exceeds the give-up window.
func (s *SupplierSettler) maybeFlagStuck(ctx context.Context, o *Order, why string) error {
	if o.Fulfillment.SupplierStuckAt != nil || time.Since(supplierClockStart(o)) <= s.giveUp {
		return nil
	}
	return s.flagStuck(ctx, o, why)
}

// rejectAdminRoute handles a definitive rejection of an attempt an admin chose
// (SupplierAttempt > 0: a reroute, or a manual order sent to a supplier). It
// never auto-refunds — the admin picked this route, so they decide next: try
// another supplier, complete by hand, or refund. The order is flagged stuck
// and the dead ref dropped (MarkSupplierRejected), so it stops being polled.
// The upstream detail goes to the audit log only (the timeline is
// customer-visible). Placement-time orders (attempt 0) keep the auto-refund.
func (s *SupplierSettler) rejectAdminRoute(ctx context.Context, o *Order, detail string) error {
	applied, err := s.svc.repo.MarkSupplierRejected(ctx, o.ID, o.Fulfillment.SupplierAttempt, TimelineEvent{
		Status: "stuck_upstream",
		Note:   "stuck upstream — needs attention: rejected by the supplier",
		At:     time.Now().UTC(),
	})
	if err != nil || !applied {
		return err
	}
	slog.Warn("supplier settler: admin-chosen supplier rejected the order", "order", o.ID.Hex())
	s.record(ctx, audit.ActionOrderSupplierSettle, o.ID, map[string]any{
		"outcome": "rejected_after_reroute", "error": detail, "attempt": o.Fulfillment.SupplierAttempt,
	})
	return nil
}

// flagStuck marks o stuck exactly once (atomic winner via MarkSupplierStuck):
// one timeline event, one audit entry, one warning — and never a refund; the
// admin decides via the existing refund action. Ref'd orders keep being
// polled afterwards (the supplier may still deliver).
func (s *SupplierSettler) flagStuck(ctx context.Context, o *Order, why string) error {
	flagged, err := s.svc.repo.MarkSupplierStuck(ctx, o.ID, o.Fulfillment.SupplierAttempt, TimelineEvent{
		Status: "stuck_upstream",
		Note:   "stuck upstream — needs attention: " + why,
		At:     time.Now().UTC(),
	})
	if err != nil || !flagged {
		return err
	}
	slog.Warn("supplier settler: order stuck upstream", "order", o.ID.Hex(), "why", why)
	s.record(ctx, audit.ActionOrderSupplierSettle, o.ID, map[string]any{
		"outcome": "stuck", "reason": why, "retryCount": o.Fulfillment.SupplierRetryCount,
	})
	return nil
}

// record writes one settler audit entry with the system actor preset (Record
// only fills the actor from JWT claims when both fields are empty — a
// background loop has no claims). Best-effort, after the winning write.
func (s *SupplierSettler) record(ctx context.Context, action string, orderID bson.ObjectID, summary map[string]any) {
	if s.rec == nil {
		return
	}
	s.rec.Record(ctx, audit.Entry{
		ActorID:    supplierActor,
		ActorEmail: supplierActor,
		Action:     action,
		TargetType: "order",
		TargetID:   orderID.Hex(),
		Summary:    summary,
	})
}

// isCtxErr reports a cancelled or timed-out call: never a definitive upstream
// rejection (the request may or may not have landed), so it is retried.
func isCtxErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
