// Package loyalty owns the earning of customer loyalty points. The authoritative
// balance lives on the user document (users.loyaltyPoints); this module mutates
// it, mirroring how the wallet module owns users.walletBalance. Awarding is
// driven by the admin-configurable knobs on the settings singleton.
package loyalty

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// pointsStore is the persistence slice the awarder needs: an atomic increment of
// a user's loyaltyPoints balance (kept narrow for testability).
type pointsStore interface {
	AddPoints(ctx context.Context, userID bson.ObjectID, delta int) (int, error)
}

// MongoRepository increments loyaltyPoints on the users collection.
type MongoRepository struct {
	users *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository over the users collection.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{users: db.Collection("users")}
}

// AddPoints atomically increments the user's loyaltyPoints and returns the new
// total. It mirrors wallet.adjust (single-document $inc). A missing user
// surfaces mongo.ErrNoDocuments unchanged.
func (r *MongoRepository) AddPoints(ctx context.Context, userID bson.ObjectID, delta int) (int, error) {
	var doc struct {
		LoyaltyPoints int `bson:"loyaltyPoints"`
	}
	err := r.users.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: userID}},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "loyaltyPoints", Value: delta}}},
			{Key: "$set", Value: bson.D{{Key: "updatedAt", Value: time.Now().UTC()}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		return 0, err
	}
	return doc.LoyaltyPoints, nil
}
