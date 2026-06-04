package user

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Repository defines persistence operations for the User entity.
type Repository interface {
	FindByID(ctx context.Context, id bson.ObjectID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
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

func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*User, error) {
	return nil, errors.New("TODO: not implemented")
}

func (r *MongoRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	return nil, errors.New("TODO: not implemented")
}

func (r *MongoRepository) Create(ctx context.Context, user *User) error {
	return errors.New("TODO: not implemented")
}

func (r *MongoRepository) Update(ctx context.Context, user *User) error {
	return errors.New("TODO: not implemented")
}

func (r *MongoRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	return errors.New("TODO: not implemented")
}
