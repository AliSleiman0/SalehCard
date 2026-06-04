package order

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Repository defines persistence operations for the Order entity.
type Repository interface {
	FindByID(ctx context.Context, id bson.ObjectID) (*Order, error)
	FindByUserID(ctx context.Context, userID bson.ObjectID) ([]*Order, error)
	Create(ctx context.Context, order *Order) error
	UpdateStatus(ctx context.Context, id bson.ObjectID, status OrderStatus) error
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository using the given collection.
func NewMongoRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: col}
}

func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*Order, error) {
	return nil, errors.New("TODO: not implemented")
}

func (r *MongoRepository) FindByUserID(ctx context.Context, userID bson.ObjectID) ([]*Order, error) {
	return nil, errors.New("TODO: not implemented")
}

func (r *MongoRepository) Create(ctx context.Context, order *Order) error {
	return errors.New("TODO: not implemented")
}

func (r *MongoRepository) UpdateStatus(ctx context.Context, id bson.ObjectID, status OrderStatus) error {
	return errors.New("TODO: not implemented")
}
