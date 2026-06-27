package seed

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/modules/reseller"
)

// tierSeed is a tier definition to ensure.
type tierSeed struct {
	Name          string
	MarginPercent float64
	BalanceLimit  float64
}

// Seed ensures the three reseller tiers (Bronze/Silver/Gold) exist and assigns
// tiers to the seeded reseller accounts so the admin list, tier cards, and
// filters have signal. Idempotent: tiers are inserted only when missing (admin
// edits are preserved), and tier assignments are a plain $set per known email.
// Must run after the user seed (it updates existing reseller users by email).
func Seed(ctx context.Context, db *mongo.Database) error {
	if err := reseller.EnsureIndexes(ctx, db); err != nil {
		return err
	}

	now := time.Now().UTC()
	tiers := []tierSeed{
		{Name: "Bronze", MarginPercent: 5, BalanceLimit: 2000},
		{Name: "Silver", MarginPercent: 8, BalanceLimit: 5000},
		{Name: "Gold", MarginPercent: 12, BalanceLimit: 15000},
	}
	tierCol := db.Collection("reseller_tiers")
	for _, t := range tiers {
		_, err := tierCol.UpdateOne(ctx,
			bson.D{{Key: "name", Value: t.Name}},
			bson.D{{Key: "$setOnInsert", Value: bson.D{
				{Key: "marginPercent", Value: t.MarginPercent},
				{Key: "balanceLimit", Value: t.BalanceLimit},
				{Key: "createdAt", Value: now},
			}}},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}

	assignments := []struct{ Email, Tier string }{
		{"gamehub.store@salehcard.local", "Gold"},
		{"topup.pro@salehcard.local", "Silver"},
		{"blocked.reseller@salehcard.local", "Bronze"},
	}
	userCol := db.Collection("users")
	for _, a := range assignments {
		_, err := userCol.UpdateOne(ctx,
			bson.D{{Key: "email", Value: a.Email}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "resellerTier", Value: a.Tier}}}},
		)
		if err != nil {
			return err
		}
	}
	return nil
}
