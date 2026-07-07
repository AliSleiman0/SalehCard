package payment

import (
	"context"
	"log/slog"
	"time"
)

const defaultWhishSweepInterval = 60 * time.Second

// WhishSweeper is the background loop that reconciles Whish redirect intents:
// each tick it re-polls the gateway for pending intents past their expiry and
// settles a late success, records a failure, or expires an abandoned intent
// (Service.SweepWhishExpired). It exists separately from the USDT Watcher
// because Whish is enabled independently — a deployment can run Whish with USDT
// off. Running two instances is safe: every state move is an atomic claim and
// settlement dedupes on the wallet (method, ref) index.
type WhishSweeper struct {
	svc      *Service
	interval time.Duration
}

// NewWhishSweeper constructs the sweeper.
func NewWhishSweeper(svc *Service, interval time.Duration) *WhishSweeper {
	if interval <= 0 {
		interval = defaultWhishSweepInterval
	}
	return &WhishSweeper{svc: svc, interval: interval}
}

// Run ticks until ctx is cancelled, firing one immediate tick at start.
func (w *WhishSweeper) Run(ctx context.Context) {
	slog.Info("payment: whish sweeper started", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		w.svc.SweepWhishExpired(ctx)
		select {
		case <-ctx.Done():
			slog.Info("payment: whish sweeper stopped")
			return
		case <-ticker.C:
		}
	}
}
