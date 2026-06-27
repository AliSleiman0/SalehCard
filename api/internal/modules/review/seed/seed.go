// Package seed inserts development product reviews so the admin moderation
// queue, its status filters, and the spam flag have signal. Reviews reference
// real seeded products + users (resolved by title / email) so the product
// rating recompute and the admin product/user join resolve correctly.
package seed

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/review"
)

// reviewSeed is one demo review, keyed to a product (by English title) and a
// user (by email).
type reviewSeed struct {
	ProductTitle string
	UserEmail    string
	Rating       int
	Body         string
	Status       string
	Verified     bool
	DaysAgo      int
}

// seeds spread reviews across products, statuses, and ratings: four pending
// (incl. a 1-star to exercise the spam flag), two approved, one rejected.
var seeds = []reviewSeed{
	{"PUBG Mobile UC", "customer@salehcard.local", 5, "Instant delivery, UC landed in seconds. Will buy again.", review.StatusPending, true, 0},
	{"Steam Wallet", "omar.haddad@salehcard.local", 4, "Smooth top-up, code worked first try.", review.StatusPending, false, 0},
	{"PUBG Mobile UC", "lina.khoury@salehcard.local", 1, "SCAM!!! buy from t.me/cheapuc instead cheaper!!!", review.StatusPending, false, 1},
	{"Steam Wallet", "lina.khoury@salehcard.local", 2, "Took a while to arrive, support was slow to respond.", review.StatusPending, false, 1},
	{"PUBG Mobile UC", "sara.nasser@salehcard.local", 5, "Best prices for UC, been using SalehCard for months.", review.StatusApproved, true, 4},
	{"Steam Wallet", "customer@salehcard.local", 4, "Reliable gift cards, good rates vs the local shops.", review.StatusApproved, true, 6},
	{"PUBG Mobile UC", "omar.haddad@salehcard.local", 1, "Wrong region code, had to contact support.", review.StatusRejected, false, 8},
}

// Seed ensures the demo reviews exist and refreshes the affected products'
// denormalized ratings. Idempotent: each review is inserted only when an entry
// with the same (productId, userId, body) is absent. Must run after the product
// and user seeds (it resolves both by their natural keys); calls EnsureIndexes.
func Seed(ctx context.Context, db *mongo.Database) error {
	if err := review.EnsureIndexes(ctx, db); err != nil {
		return err
	}

	products := db.Collection("products")
	users := db.Collection("users")
	reviews := db.Collection("reviews")
	now := time.Now().UTC()

	touched := map[bson.ObjectID]struct{}{}

	for _, s := range seeds {
		var prod struct {
			ID bson.ObjectID `bson:"_id"`
		}
		err := products.FindOne(ctx, bson.D{{Key: "title.en", Value: s.ProductTitle}}).Decode(&prod)
		if err == mongo.ErrNoDocuments {
			continue // product not seeded on this box; skip its reviews
		}
		if err != nil {
			return err
		}

		var usr struct {
			ID bson.ObjectID `bson:"_id"`
		}
		err = users.FindOne(ctx, bson.D{{Key: "email", Value: s.UserEmail}}).Decode(&usr)
		if err == mongo.ErrNoDocuments {
			continue
		}
		if err != nil {
			return err
		}

		exists, err := reviews.CountDocuments(ctx, bson.D{
			{Key: "productId", Value: prod.ID},
			{Key: "userId", Value: usr.ID},
			{Key: "body", Value: s.Body},
		})
		if err != nil {
			return err
		}
		if exists == 0 {
			_, err = reviews.InsertOne(ctx, &review.Review{
				ID:               bson.NewObjectID(),
				ProductID:        prod.ID,
				UserID:           usr.ID,
				Rating:           s.Rating,
				Body:             s.Body,
				VerifiedPurchase: s.Verified,
				Status:           s.Status,
				CreatedAt:        now.AddDate(0, 0, -s.DaysAgo),
			})
			if err != nil {
				return err
			}
		}
		touched[prod.ID] = struct{}{}
	}

	// Refresh the denormalized rating for every product that got reviews, so the
	// storefront rating reflects the seeded approved reviews.
	repo := review.NewMongoRepository(reviews, products)
	for pid := range touched {
		if err := repo.RecomputeProductRating(ctx, pid); err != nil {
			return err
		}
	}
	return nil
}
