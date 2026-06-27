// Package seed inserts development promo codes so the admin list/filters and a
// checkout discount are demoable. Codes cover every type and lifecycle state
// (active / expired / depleted) so the status filters have signal.
package seed

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/modules/promo"
)

// promoSeed is one promo definition to ensure.
type promoSeed struct {
	Code      string
	Type      promo.PromoType
	Value     float64
	MinOrder  float64
	MaxUses   int
	Uses      int
	StartsAt  *time.Time
	ExpiresAt *time.Time
	Active    bool
}

// Seed ensures a handful of promo codes exist. Idempotent: each code is inserted
// only when its code is absent (admin edits + use counts are preserved). Safe to
// run repeatedly; must run after EnsureIndexes (called here).
func Seed(ctx context.Context, db *mongo.Database) error {
	if err := promo.EnsureIndexes(ctx, db); err != nil {
		return err
	}

	now := time.Now().UTC()
	future := now.AddDate(0, 1, 0)
	past := now.AddDate(0, 0, -3)

	promos := []promoSeed{
		{Code: "WELCOME10", Type: promo.PromoTypePercent, Value: 10, MinOrder: 0, MaxUses: 5000, Uses: 1840, ExpiresAt: &future, Active: true},
		{Code: "USDT5", Type: promo.PromoTypeFixed, Value: 5, MinOrder: 20, MaxUses: 1000, Uses: 412, ExpiresAt: &future, Active: true},
		{Code: "CASH3", Type: promo.PromoTypeCashback, Value: 3, MinOrder: 0, MaxUses: 0, Uses: 96, Active: true},
		{Code: "SAVE20", Type: promo.PromoTypePercent, Value: 20, MinOrder: 50, MaxUses: 2000, Uses: 300, ExpiresAt: &past, Active: true},
		{Code: "MAXED", Type: promo.PromoTypeFixed, Value: 5, MinOrder: 0, MaxUses: 50, Uses: 50, Active: true},
		{Code: "PAUSED15", Type: promo.PromoTypePercent, Value: 15, MinOrder: 0, MaxUses: 1000, Uses: 0, Active: false},
	}

	col := db.Collection("promos")
	for _, p := range promos {
		_, err := col.UpdateOne(ctx,
			bson.D{{Key: "code", Value: p.Code}},
			bson.D{{Key: "$setOnInsert", Value: bson.D{
				{Key: "code", Value: p.Code},
				{Key: "type", Value: p.Type},
				{Key: "value", Value: p.Value},
				{Key: "minOrder", Value: p.MinOrder},
				{Key: "maxUses", Value: p.MaxUses},
				{Key: "uses", Value: p.Uses},
				{Key: "startsAt", Value: p.StartsAt},
				{Key: "expiresAt", Value: p.ExpiresAt},
				{Key: "active", Value: p.Active},
				{Key: "createdAt", Value: now},
			}}},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}
	return nil
}
