package promo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Repository defines persistence operations for PromoCode entities.
type Repository interface {
	FindByCode(ctx context.Context, code string) (*PromoCode, error)
	Create(ctx context.Context, promo *PromoCode) error
	IncrementUses(ctx context.Context, code string) error
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository using the given collection.
func NewMongoRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: col}
}

func (r *MongoRepository) FindByCode(ctx context.Context, code string) (*PromoCode, error) {
	return nil, errors.New("TODO: not implemented")
}

func (r *MongoRepository) Create(ctx context.Context, promo *PromoCode) error {
	return errors.New("TODO: not implemented")
}

func (r *MongoRepository) IncrementUses(ctx context.Context, code string) error {
	return errors.New("TODO: not implemented")
}
