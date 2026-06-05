package seed

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

// Seed inserts initial product documents when the products collection is empty.
// It is safe to call multiple times; it is a no-op when data already exists.
func Seed(ctx context.Context, db *mongo.Database) error {
	col := db.Collection("products")

	count, err := col.CountDocuments(ctx, bson.D{})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now().UTC()

	resellerPrice5 := 4.50
	resellerPrice10 := 9.00
	resellerPrice20 := 18.00
	resellerPrice60UC := 0.89
	resellerPrice325UC := 4.49

	products := []product.Product{
		{
			ID: bson.NewObjectID(),
			Title: product.I18nString{
				En: "Steam Wallet",
				Ar: "محفظة ستيم",
				Tr: "Steam Cüzdanı",
			},
			Category: "giftcards",
			Images: []string{
				"https://cdn.salehcard.com/images/steam-wallet.png",
			},
			Variants: []product.Variant{
				{
					ID:            bson.NewObjectID(),
					Denomination:  "$5 USD",
					Price:         5.00,
					ResellerPrice: &resellerPrice5,
				},
				{
					ID:            bson.NewObjectID(),
					Denomination:  "$10 USD",
					Price:         10.00,
					ResellerPrice: &resellerPrice10,
				},
				{
					ID:            bson.NewObjectID(),
					Denomination:  "$20 USD",
					Price:         20.00,
					ResellerPrice: &resellerPrice20,
				},
			},
			FulfillmentType: product.FulfillmentCode,
			Stock:           500,
			Available:       true,
			Ratings:         product.RatingsSummary{Average: 0, Count: 0},
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID: bson.NewObjectID(),
			Title: product.I18nString{
				En: "PUBG Mobile UC",
				Ar: "يوسي ببجي موبايل",
				Tr: "PUBG Mobile UC",
			},
			Category: "games",
			Images: []string{
				"https://cdn.salehcard.com/images/pubg-uc.png",
			},
			Variants: []product.Variant{
				{
					ID:            bson.NewObjectID(),
					Denomination:  "60 UC",
					Price:         0.99,
					ResellerPrice: &resellerPrice60UC,
				},
				{
					ID:            bson.NewObjectID(),
					Denomination:  "325 UC",
					Price:         4.99,
					ResellerPrice: &resellerPrice325UC,
				},
			},
			FulfillmentType: product.FulfillmentCredit,
			Stock:           1000,
			Available:       true,
			Ratings:         product.RatingsSummary{Average: 0, Count: 0},
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID: bson.NewObjectID(),
			Title: product.I18nString{
				En: "Bank Transfer",
				Ar: "تحويل بنكي",
				Tr: "Banka Transferi",
			},
			Category: "transfer",
			Images: []string{
				"https://cdn.salehcard.com/images/bank-transfer.png",
			},
			Variants: []product.Variant{
				{
					ID:           bson.NewObjectID(),
					Denomination: "Variable",
					Price:        0,
				},
			},
			FulfillmentType: product.FulfillmentTransfer,
			Stock:           0,
			Available:       true,
			Ratings:         product.RatingsSummary{Average: 0, Count: 0},
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}

	_, err = col.InsertMany(ctx, products)
	return err
}
