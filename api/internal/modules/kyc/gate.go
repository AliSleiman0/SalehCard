package kyc

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// Gate answers "is this user identity-verified?" for other modules (the
// checkout KYC gate) without exposing the full repository.
type Gate struct {
	repo Repository
}

// NewGate constructs a Gate over the kyc_submissions collection.
func NewGate(db *mongo.Database) *Gate {
	return &Gate{repo: NewMongoRepository(db.Collection("kyc_submissions"), db.Collection("users"))}
}

// IsApproved reports whether the user has an approved KYC submission. A user
// with no submission is simply not approved (nil error).
func (g *Gate) IsApproved(ctx context.Context, userID bson.ObjectID) (bool, error) {
	s, err := g.repo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return s.Status == StatusApproved, nil
}
