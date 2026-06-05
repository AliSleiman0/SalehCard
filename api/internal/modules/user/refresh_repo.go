package user

import (
	"context"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// RefreshRepository defines persistence operations for refresh tokens.
type RefreshRepository interface {
	Create(ctx context.Context, t *RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id bson.ObjectID) error
	RevokeAllForUser(ctx context.Context, userID bson.ObjectID) error
}

// MongoRefreshRepository is a MongoDB-backed RefreshRepository.
type MongoRefreshRepository struct {
	collection *mongo.Collection
}

// NewMongoRefreshRepository constructs a store over the "refresh_tokens" collection.
func NewMongoRefreshRepository(db *mongo.Database) *MongoRefreshRepository {
	return &MongoRefreshRepository{collection: db.Collection("refresh_tokens")}
}

// EnsureRefreshIndexes creates a unique index on tokenHash plus a TTL index on
// expiresAt so expired tokens are purged automatically by MongoDB.
func EnsureRefreshIndexes(ctx context.Context, db *mongo.Database) error {
	tokens := db.Collection("refresh_tokens")
	_, err := tokens.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "tokenHash", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
		{Keys: bson.D{{Key: "userId", Value: 1}}},
	})
	return err
}

// Create inserts a refresh-token record.
func (r *MongoRefreshRepository) Create(ctx context.Context, t *RefreshToken) error {
	if t.ID.IsZero() {
		t.ID = bson.NewObjectID()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	_, err := r.collection.InsertOne(ctx, t)
	return err
}

// FindByHash returns the token record for the given hash, or ErrNotFound.
func (r *MongoRefreshRepository) FindByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	var t RefreshToken
	err := r.collection.FindOne(ctx, bson.D{{Key: "tokenHash", Value: hash}}).Decode(&t)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// Revoke marks a single token revoked (idempotent).
func (r *MongoRefreshRepository) Revoke(ctx context.Context, id bson.ObjectID) error {
	now := time.Now().UTC()
	_, err := r.collection.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "revokedAt", Value: now}}}},
	)
	return err
}

// RevokeAllForUser marks every non-revoked token for a user revoked.
func (r *MongoRefreshRepository) RevokeAllForUser(ctx context.Context, userID bson.ObjectID) error {
	now := time.Now().UTC()
	_, err := r.collection.UpdateMany(ctx,
		bson.D{{Key: "userId", Value: userID}, {Key: "revokedAt", Value: nil}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "revokedAt", Value: now}}}},
	)
	return err
}
