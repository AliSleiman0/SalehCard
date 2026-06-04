package reseller

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Repository defines persistence operations for ResellerTier entities.
type Repository interface {
	FindByUserID(ctx context.Context, userID bson.ObjectID) (*ResellerTier, error)
	Create(ctx context.Context, tier *ResellerTier) error
	Update(ctx context.Context, tier *ResellerTier) error
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository using the given collection.
func NewMongoRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: col}
}

func (r *MongoRepository) FindByUserID(ctx context.Context, userID bson.ObjectID) (*ResellerTier, error) {
	return nil, errors.New("TODO: not implemented")
}

func (r *MongoRepository) Create(ctx context.Context, tier *ResellerTier) error {
	return errors.New("TODO: not implemented")
}

func (r *MongoRepository) Update(ctx context.Context, tier *ResellerTier) error {
	return errors.New("TODO: not implemented")
}
