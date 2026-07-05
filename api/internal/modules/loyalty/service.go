package loyalty

import (
	"context"
	"log/slog"

	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// configSource reads the loyalty knobs off the app-settings singleton (kept
// narrow so the awarder can be unit-tested without Mongo).
type configSource interface {
	Get(ctx context.Context) (*settings.Settings, error)
}

// Awarder mints loyalty points for completed orders. It is deliberately
// best-effort: every failure is logged and swallowed so awarding can never fail
// or roll back an order that has already completed — the same posture as the
// order module's completion notifications.
type Awarder struct {
	cfg    configSource
	points pointsStore
}

// NewAwarder builds an Awarder over the app-settings and users collections.
func NewAwarder(db *mongo.Database) *Awarder {
	return &Awarder{
		cfg:    settings.NewMongoRepository(db),
		points: NewMongoRepository(db),
	}
}

// Award grants floor(orderTotal / earnRate) points to the buyer when the loyalty
// program is enabled. orderTotal is the post-discount order total in dollars
// (what the customer actually paid). It never returns an error by design.
func (a *Awarder) Award(ctx context.Context, userID bson.ObjectID, orderTotal float64) {
	cfg, err := a.cfg.Get(ctx)
	if err != nil {
		slog.Warn("loyalty: could not read settings; skipping award", "err", err)
		return
	}
	if !cfg.LoyaltyEnabled || cfg.LoyaltyEarnUsdPerPoint <= 0 {
		return
	}
	pts := int(orderTotal / cfg.LoyaltyEarnUsdPerPoint)
	if pts <= 0 {
		return
	}
	total, err := a.points.AddPoints(ctx, userID, pts)
	if err != nil {
		slog.Warn("loyalty: award failed", "userId", userID.Hex(), "points", pts, "err", err)
		return
	}
	slog.Info("loyalty: awarded points", "userId", userID.Hex(), "points", pts, "total", total)
}
