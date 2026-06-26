package user

import (
	"context"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// OTPRepository defines persistence for one-time passcodes. There is at most one
// active record per phone number (upserted on each request).
type OTPRepository interface {
	Upsert(ctx context.Context, code *OtpCode) error
	FindByPhone(ctx context.Context, phone string) (*OtpCode, error)
	IncrementAttempts(ctx context.Context, phone string) error
	DeleteByPhone(ctx context.Context, phone string) error
}

// MongoOTPRepository is a MongoDB-backed OTPRepository over "otp_codes".
type MongoOTPRepository struct {
	collection *mongo.Collection
}

// NewMongoOTPRepository constructs a store over the "otp_codes" collection.
func NewMongoOTPRepository(db *mongo.Database) *MongoOTPRepository {
	return &MongoOTPRepository{collection: db.Collection("otp_codes")}
}

// EnsureOTPIndexes creates a unique index on phone (one active code per number)
// plus a TTL index on expiresAt so spent/expired codes are purged automatically.
func EnsureOTPIndexes(ctx context.Context, db *mongo.Database) error {
	codes := db.Collection("otp_codes")
	_, err := codes.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "phone", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	})
	return err
}

// Upsert replaces any existing code for the phone with the new one, resetting
// attempts. The phone is the conflict key.
func (r *MongoOTPRepository) Upsert(ctx context.Context, code *OtpCode) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.D{{Key: "phone", Value: code.Phone}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "codeHash", Value: code.CodeHash},
			{Key: "expiresAt", Value: code.ExpiresAt},
			{Key: "attempts", Value: 0},
			{Key: "createdAt", Value: code.CreatedAt},
			{Key: "consumedAt", Value: nil},
		}}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// FindByPhone returns the active code for a phone, or ErrNotFound.
func (r *MongoOTPRepository) FindByPhone(ctx context.Context, phone string) (*OtpCode, error) {
	var c OtpCode
	err := r.collection.FindOne(ctx, bson.D{{Key: "phone", Value: phone}}).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// IncrementAttempts bumps the wrong-guess counter for a phone's code.
func (r *MongoOTPRepository) IncrementAttempts(ctx context.Context, phone string) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.D{{Key: "phone", Value: phone}},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "attempts", Value: 1}}}},
	)
	return err
}

// DeleteByPhone removes a phone's code (called once it has been consumed).
func (r *MongoOTPRepository) DeleteByPhone(ctx context.Context, phone string) error {
	_, err := r.collection.DeleteOne(ctx, bson.D{{Key: "phone", Value: phone}})
	return err
}
