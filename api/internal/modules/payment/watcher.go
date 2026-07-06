package payment

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

const (
	defaultWatchInterval = 25 * time.Second
	// perTickLimit caps each scan set so one tick stays bounded; anything
	// beyond the cap is picked up next tick (oldest first).
	perTickLimit = 200
	// chainCallGap throttles chain-provider calls well under TronGrid's
	// keyed rate limit.
	chainCallGap = 100 * time.Millisecond
)

// Watcher is the background loop that turns on-chain transfers into settled
// payments. It is the codebase's first long-running worker: started once at
// boot, stopped via context cancellation on shutdown. Running two instances
// (scale-out) is safe — every state move is an atomic claim and the
// (network, txHash) unique index blocks double-recording — it only wastes
// chain-provider quota.
type Watcher struct {
	svc      *Service
	interval time.Duration
}

// NewWatcher constructs the watcher.
func NewWatcher(svc *Service, interval time.Duration) *Watcher {
	if interval <= 0 {
		interval = defaultWatchInterval
	}
	return &Watcher{svc: svc, interval: interval}
}

// Run ticks until ctx is cancelled. It fires one immediate tick at start so a
// restart doesn't add a full interval of settlement latency.
func (w *Watcher) Run(ctx context.Context) {
	slog.Info("payment: watcher started", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		w.tick(ctx)
		select {
		case <-ctx.Done():
			slog.Info("payment: watcher stopped")
			return
		case <-ticker.C:
		}
	}
}

// tick runs one full pass: expire overdue intents, retry stuck settlements,
// scan the chain for new payments. Exported-for-test via watcher_test.go; all
// per-intent errors are logged and never abort the pass.
func (w *Watcher) tick(ctx context.Context) {
	tickCtx, cancel := context.WithTimeout(ctx, w.tickBudget())
	defer cancel()

	w.sweepExpired(tickCtx)
	w.retrySettlements(tickCtx)
	w.scanChain(tickCtx)
}

// tickBudget bounds one pass so a slow chain provider can't overlap ticks.
func (w *Watcher) tickBudget() time.Duration {
	if w.interval > 4*time.Second {
		return w.interval - 2*time.Second
	}
	return w.interval
}

// sweepExpired flips overdue pending intents to expired, failing the order
// behind an order intent. Expiry runs FIRST in the tick so an order intent
// can't be claimed and expired in the same pass; a payment that lands in the
// same tick is picked up by the grace re-scan next tick and settles as a
// wallet credit via the order's ErrOrderNotPayable fallback.
func (w *Watcher) sweepExpired(ctx context.Context) {
	now := time.Now().UTC()
	candidates, err := w.svc.store.ListExpiryCandidates(ctx, now, perTickLimit)
	if err != nil {
		slog.Error("payment: list expiry candidates failed", "error", err)
		return
	}
	for _, in := range candidates {
		expired, err := w.svc.store.ExpireOne(ctx, in.ID, now)
		if err != nil {
			if !errors.Is(err, apperrors.ErrConflict) { // conflict = moved on concurrently, fine
				slog.Error("payment: expire intent failed", "intent", in.ID.Hex(), "error", err)
			}
			continue
		}
		if expired.Purpose == PurposeOrder && expired.OrderID != nil && w.svc.orders != nil {
			if err := w.svc.orders.FailUnpaidOrder(ctx, *expired.OrderID, "payment expired"); err != nil {
				// Best-effort: the order-side sweep retries implicitly because
				// FulfillPaidOrder on a late payment checks order state anyway;
				// an unpaid order stuck pending is visible to admins.
				slog.Error("payment: fail expired order failed", "intent", expired.ID.Hex(), "order", expired.OrderID.Hex(), "error", err)
			}
		}
		w.svc.notify(ctx, expired, notification.KindPaymentExpired, "Payment window expired",
			"Your USDT payment window expired before a transfer was received.")
	}
}

// retrySettlements re-runs settlement for confirming intents — payments whose
// transfer was recorded but whose money-move didn't complete (crash or a
// dependency error on a previous attempt).
func (w *Watcher) retrySettlements(ctx context.Context) {
	stuck, err := w.svc.store.ListSettleRetries(ctx, perTickLimit)
	if err != nil {
		slog.Error("payment: list settle retries failed", "error", err)
		return
	}
	for _, in := range stuck {
		if err := w.svc.settle(ctx, in); err != nil {
			continue // settle already logged + bumped attempts
		}
	}
}

// scanChain polls the chain reader for every watchable intent and claims +
// settles any observed payment.
func (w *Watcher) scanChain(ctx context.Context) {
	now := time.Now().UTC()
	watchable, err := w.svc.store.ListWatchable(ctx, now, w.svc.cfg.LateGrace, perTickLimit)
	if err != nil {
		slog.Error("payment: list watchable failed", "error", err)
		return
	}
	for i, in := range watchable {
		if ctx.Err() != nil {
			return
		}
		if i > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(chainCallGap):
			}
		}
		p, err := w.svc.reader.FindPayment(ctx, tron.Watch{
			Address:        in.Address,
			CreatedAt:      in.CreatedAt,
			ExpectedMicros: in.AmountExpectedMicros,
		})
		if err != nil {
			slog.Warn("payment: chain lookup failed", "intent", in.ID.Hex(), "error", err)
			continue
		}
		if p == nil || p.AmountMicros <= 0 {
			continue
		}
		claimed, err := w.svc.store.ClaimPaymentSeen(ctx, in.ID, p.TxHash, p.From, p.AmountMicros)
		if err != nil {
			if !errors.Is(err, apperrors.ErrConflict) { // conflict = another claim won / tx already recorded
				slog.Error("payment: claim failed", "intent", in.ID.Hex(), "tx", p.TxHash, "error", err)
			}
			continue
		}
		slog.Info("payment: on-chain payment claimed",
			"intent", claimed.ID.Hex(), "purpose", claimed.Purpose, "tx", p.TxHash, "receivedMicros", p.AmountMicros)
		if err := w.svc.settle(ctx, claimed); err != nil {
			continue // stays confirming; retried next tick
		}
	}
}
