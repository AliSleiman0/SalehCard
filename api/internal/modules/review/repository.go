package review

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Repository defines persistence operations for Review entities.
type Repository interface {
	FindByProductID(ctx context.Context, productID bson.ObjectID) ([]*Review, error)
	Create(ctx context.Context, review *Review) error
	Delete(ctx context.Context, id bson.ObjectID) error
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository using the given collection.
func NewMongoRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: col}
}

func (r *MongoRepository) FindByProductID(ctx context.Context, productID bson.ObjectID) ([]*Review, error) {
	return nil, errors.New("TODO: not implemented")
}

func (r *MongoRepository) Create(ctx context.Context, review *Review) error {
	return errors.New("TODO: not implemented")
}

func (r *MongoRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	return errors.New("TODO: not implemented")
}
