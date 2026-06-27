// Package seed inserts development offers (sale-price deals) over a couple of the
// seeded products so the admin list/filters and the storefront Offers tab are
// demoable. Idempotent: an offer is inserted only when one is absent for that
// product. Safe to run repeatedly; must run after the product seed.
package seed

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/modules/offer"
)

// offerSeed is one offer definition (discount + window) to ensure on a product.
type offerSeed struct {
	DiscountType  offer.DiscountType
	DiscountValue float64
	EndsInDays    int // EndsAt = now + this many days (0 = open-ended)
	SortOrder     int
}

// Seed ensures a couple of offers exist over the first available products. It is
// best-effort: with no products seeded yet it simply does nothing.
func Seed(ctx context.Context, db *mongo.Database) error {
	if err := offer.EnsureIndexes(ctx, db); err != nil {
		return err
	}

	// Pick the first two available products to attach demo offers to.
	cur, err := db.Collection("products").Find(ctx,
		bson.D{{Key: "available", Value: true}},
		options.Find().SetLimit(2).SetSort(bson.D{{Key: "createdAt", Value: 1}}),
	)
	if err != nil {
		return err
	}
	var products []struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := cur.All(ctx, &products); err != nil {
		return err
	}

	defs := []offerSeed{
		{DiscountType: offer.DiscountPercent, DiscountValue: 25, EndsInDays: 14, SortOrder: 0},
		{DiscountType: offer.DiscountFixed, DiscountValue: 3, EndsInDays: 30, SortOrder: 1},
	}

	now := time.Now().UTC()
	col := db.Collection("offers")
	for i, p := range products {
		if i >= len(defs) {
			break
		}
		d := defs[i]
		var endsAt *time.Time
		if d.EndsInDays > 0 {
			t := now.AddDate(0, 0, d.EndsInDays)
			endsAt = &t
		}
		_, err := col.UpdateOne(ctx,
			bson.D{{Key: "productId", Value: p.ID}},
			bson.D{{Key: "$setOnInsert", Value: bson.D{
				{Key: "productId", Value: p.ID},
				{Key: "discountType", Value: d.DiscountType},
				{Key: "discountValue", Value: d.DiscountValue},
				{Key: "startsAt", Value: nil},
				{Key: "endsAt", Value: endsAt},
				{Key: "active", Value: true},
				{Key: "sortOrder", Value: d.SortOrder},
				{Key: "createdAt", Value: now},
				{Key: "updatedAt", Value: now},
			}}},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}
	return nil
}
