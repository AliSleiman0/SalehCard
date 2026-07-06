package bridge

import (
	"context"
	"log/slog"
	"time"
)

const defaultReaperInterval = 30 * time.Second

// Reaper is the bridge background worker: it requeues expired command leases,
// fails commands no device ever picked up, and (dev stub) auto-completes queued
// commands. Same lifecycle shape as the payment watcher — started once at boot,
// stopped via context cancellation on shutdown. Safe to run multiple instances
// (every state move is an atomic claim).
type Reaper struct {
	svc      *Service
	interval time.Duration
}

// NewReaper constructs the reaper.
func NewReaper(svc *Service, interval time.Duration) *Reaper {
	if interval <= 0 {
		interval = defaultReaperInterval
	}
	return &Reaper{svc: svc, interval: interval}
}

// Run ticks until ctx is cancelled, firing one immediate pass at start.
func (rp *Reaper) Run(ctx context.Context) {
	slog.Info("bridge: reaper started", "interval", rp.interval, "stub", rp.svc.cfg.Stub)
	ticker := time.NewTicker(rp.interval)
	defer ticker.Stop()
	for {
		rp.svc.ReapTick(ctx)
		select {
		case <-ctx.Done():
			slog.Info("bridge: reaper stopped")
			return
		case <-ticker.C:
		}
	}
}
