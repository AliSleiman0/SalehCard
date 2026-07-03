package ratelimit

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoLimiter is a fixed-window rate limiter backed by a TTL counter collection
// (rate_counters) — no infra beyond the app database, and correct across restarts
// and instances. It fails open on any DB error.
type MongoLimiter struct {
	col *mongo.Collection
}

// NewMongoLimiter constructs a MongoLimiter over the rate_counters collection.
func NewMongoLimiter(db *mongo.Database) *MongoLimiter {
	return &MongoLimiter{col: db.Collection("rate_counters")}
}

// EnsureIndexes creates the TTL index that expires spent windows (documents
// self-delete once their window's expiresAt passes). Best-effort, called at boot.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("rate_counters").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
	return err
}

// Allow atomically increments the counter for key's current fixed window and
// reports whether the post-increment count is within limit.
func (m *MongoLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().UTC()
	start := now.Truncate(window)
	id := fmt.Sprintf("%s|%d", key, start.Unix())

	var res struct {
		Count int `bson:"count"`
	}
	err := m.col.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{
			{Key: "$inc", Value: bson.D{{Key: "count", Value: 1}}},
			{Key: "$setOnInsert", Value: bson.D{{Key: "expiresAt", Value: start.Add(window)}}},
		},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&res)
	if err != nil {
		return true, err // fail open — never block on an infra error
	}
	return res.Count <= limit, nil
}
