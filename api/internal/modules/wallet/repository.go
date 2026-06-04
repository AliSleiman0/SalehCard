package wallet

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Repository defines persistence operations for wallet transactions.
type Repository interface {
	Create(ctx context.Context, tx *WalletTransaction) error
	FindByUserID(ctx context.Context, userID bson.ObjectID) ([]*WalletTransaction, error)
	GetBalance(ctx context.Context, userID bson.ObjectID) (float64, error)
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository using the given collection.
func NewMongoRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: col}
}

func (r *MongoRepository) Create(ctx context.Context, tx *WalletTransaction) error {
	return errors.New("TODO: not implemented")
}

func (r *MongoRepository) FindByUserID(ctx context.Context, userID bson.ObjectID) ([]*WalletTransaction, error) {
	return nil, errors.New("TODO: not implemented")
}

func (r *MongoRepository) GetBalance(ctx context.Context, userID bson.ObjectID) (float64, error) {
	return 0, errors.New("TODO: not implemented")
}
