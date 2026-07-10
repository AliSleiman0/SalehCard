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
// release stale shared amount slots, scan the chain for new payments.
// Exported-for-test via watcher_test.go; all per-intent errors are logged and
// never abort the pass.
func (w *Watcher) tick(ctx context.Context) {
	tickCtx, cancel := context.WithTimeout(ctx, w.tickBudget())
	defer cancel()

	w.sweepExpired(tickCtx)
	w.retrySettlements(tickCtx)
	w.releaseSharedSlots(tickCtx)
	w.scanChain(tickCtx)
}

// releaseSharedSlots frees the amount slots of shared intents whose
// late-payment grace ended — the amount becomes reusable exactly when the
// scan stops matching it.
func (w *Watcher) releaseSharedSlots(ctx context.Context) {
	if err := w.svc.store.ReleaseSharedSlots(ctx, time.Now().UTC(), w.svc.cfg.LateGrace); err != nil {
		slog.Error("payment: release shared slots failed", "error", err)
	}
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

// scanChain polls the chain for every watchable intent and claims + settles
// any observed payment. Intents are partitioned by their stamped address mode
// — derived ones poll their unique address, shared ones amount-match against
// one transfer listing per shared address — so a mid-flight mode flip is safe.
func (w *Watcher) scanChain(ctx context.Context) {
	now := time.Now().UTC()
	watchable, err := w.svc.store.ListWatchable(ctx, now, w.svc.cfg.LateGrace, perTickLimit)
	if err != nil {
		slog.Error("payment: list watchable failed", "error", err)
		return
	}
	var derived, shared []*Intent
	for _, in := range watchable {
		if in.IsShared() {
			shared = append(shared, in)
		} else {
			derived = append(derived, in)
		}
	}
	w.scanDerived(ctx, derived)
	w.scanShared(ctx, shared)
}

// scanDerived is the per-address poll: one FindPayment call per intent.
func (w *Watcher) scanDerived(ctx context.Context, intents []*Intent) {
	for i, in := range intents {
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
		w.claimAndSettle(ctx, in, p)
	}
}

// scanShared amount-matches shared-address intents: one ListTransfers call
// per (network, address) group (in practice one per enabled network) covers
// every open intent on it. Each network resolves its own transfer lister, so
// TRC20 and BEP20 scan independently within the same tick.
func (w *Watcher) scanShared(ctx context.Context, intents []*Intent) {
	if len(intents) == 0 {
		return
	}
	groups := map[string][]*Intent{}
	for _, in := range intents {
		groups[in.Network+"|"+in.Address] = append(groups[in.Network+"|"+in.Address], in)
	}
	first := true
	for _, group := range groups {
		if ctx.Err() != nil {
			return
		}
		if !first {
			select {
			case <-ctx.Done():
				return
			case <-time.After(chainCallGap):
			}
		}
		first = false
		network, addr := group[0].Network, group[0].Address
		lister := w.svc.listerFor(network)
		if lister == nil {
			// A network flipped off (or a reader that can't list) with open
			// intents still in their window — they drain via expiry/grace.
			slog.Error("payment: no transfer lister for network — its shared-address intents cannot settle",
				"network", network)
			continue
		}
		w.scanSharedGroup(ctx, lister, network, addr, group)
	}
}

// scanSharedGroup matches one network+address's transfers to its open intents.
// Matching is strict: exact salted amount AND the transfer is not older than
// the intent (an already-consumed transfer must never match a NEW intent that
// later reused the amount slot). Everything else lands in the reconciliation
// queue as an unmatched deposit.
func (w *Watcher) scanSharedGroup(ctx context.Context, lister tron.TransferLister, network, addr string, group []*Intent) {
	since := group[0].CreatedAt
	hints := make([]tron.AmountHint, 0, len(group))
	byAmount := make(map[int64]*Intent, len(group))
	for _, in := range group {
		if in.CreatedAt.Before(since) {
			since = in.CreatedAt
		}
		hints = append(hints, tron.AmountHint{AmountMicros: in.AmountExpectedMicros, CreatedAt: in.CreatedAt})
		byAmount[in.AmountExpectedMicros] = in
	}

	transfers, err := lister.ListTransfers(ctx, addr, since, hints)
	if err != nil {
		slog.Warn("payment: shared-address transfer list failed", "address", addr, "error", err)
		return
	}
	if len(transfers) == 0 {
		return
	}

	hashes := make([]string, len(transfers))
	for i := range transfers {
		hashes[i] = transfers[i].TxHash
	}
	known, err := w.svc.store.FilterKnownTxHashes(ctx, network, hashes)
	if err != nil {
		slog.Error("payment: filter known tx hashes failed", "address", addr, "error", err)
		return
	}

	for i := range transfers {
		if ctx.Err() != nil {
			return
		}
		p := &transfers[i]
		if known[p.TxHash] || p.AmountMicros <= 0 {
			continue
		}
		if in, ok := byAmount[p.AmountMicros]; ok && !p.BlockTime.Before(in.CreatedAt) {
			// Remove the intent from the map first: a second equal transfer
			// in this same tick must go to the unmatched queue, not re-claim.
			delete(byAmount, p.AmountMicros)
			w.claimAndSettle(ctx, in, p)
			continue
		}
		if err := w.svc.store.RecordUnmatchedDeposit(ctx, &Deposit{
			Network:      network,
			TxHash:       p.TxHash,
			FromAddress:  p.From,
			ToAddress:    addr,
			AmountMicros: p.AmountMicros,
			BlockTime:    p.BlockTime,
		}); err != nil {
			slog.Error("payment: record unmatched deposit failed", "tx", p.TxHash, "error", err)
		} else {
			slog.Info("payment: unmatched shared-address deposit recorded",
				"network", network, "tx", p.TxHash, "from", p.From, "amountMicros", p.AmountMicros)
		}
	}
}

// claimAndSettle is the shared tail of both scan paths: atomically claim the
// observed transfer onto the intent, then settle.
func (w *Watcher) claimAndSettle(ctx context.Context, in *Intent, p *tron.Payment) {
	claimed, err := w.svc.store.ClaimPaymentSeen(ctx, in.ID, p.TxHash, p.From, p.AmountMicros)
	if err != nil {
		if !errors.Is(err, apperrors.ErrConflict) { // conflict = another claim won / tx already recorded
			slog.Error("payment: claim failed", "intent", in.ID.Hex(), "tx", p.TxHash, "error", err)
		}
		return
	}
	slog.Info("payment: on-chain payment claimed",
		"intent", claimed.ID.Hex(), "purpose", claimed.Purpose, "tx", p.TxHash, "receivedMicros", p.AmountMicros)
	if err := w.svc.settle(ctx, claimed); err != nil {
		return // stays confirming; retried next tick
	}
}
