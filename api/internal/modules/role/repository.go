package role

import (
	"context"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository defines persistence operations for custom admin roles.
type Repository interface {
	List(ctx context.Context) ([]*Role, error)
	FindByID(ctx context.Context, id bson.ObjectID) (*Role, error)
	Create(ctx context.Context, role *Role) error
	Update(ctx context.Context, id bson.ObjectID, name, description string, permissions []string) (*Role, error)
	Delete(ctx context.Context, id bson.ObjectID) error
	// CountAssigned returns how many users currently hold the role, blocking
	// deletion of an in-use role.
	CountAssigned(ctx context.Context, id bson.ObjectID) (int64, error)
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	collection *mongo.Collection
	users      *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository over db's roles collection
// (plus a read-only handle on users for the in-use check).
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		collection: db.Collection("roles"),
		users:      db.Collection("users"),
	}
}

// EnsureIndexes creates the roles indexes: a unique index on name so two roles
// can never share one.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("roles").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "name", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
	return err
}

// List returns all custom roles, alphabetical by name.
func (r *MongoRepository) List(ctx context.Context) ([]*Role, error) {
	cur, err := r.collection.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*Role{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FindByID retrieves a role by ObjectID, ErrNotFound when absent.
func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*Role, error) {
	var role Role
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &role, nil
}

// Create inserts a new role, stamping CreatedAt/UpdatedAt.
func (r *MongoRepository) Create(ctx context.Context, role *Role) error {
	if role.ID.IsZero() {
		role.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	role.CreatedAt = now
	role.UpdatedAt = now
	_, err := r.collection.InsertOne(ctx, role)
	return err
}

// Update atomically sets a role's editable fields and returns the post-update
// document. ErrNotFound when no role matches.
func (r *MongoRepository) Update(ctx context.Context, id bson.ObjectID, name, description string, permissions []string) (*Role, error) {
	var role Role
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "name", Value: name},
			{Key: "description", Value: description},
			{Key: "permissions", Value: permissions},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &role, nil
}

// Delete removes a role by ObjectID, ErrNotFound when absent.
func (r *MongoRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// CountAssigned returns how many admins have this role assigned. The role:admin
// predicate lets the existing role_1 index bound the scan to the handful of
// admin documents instead of a full collection scan over every customer (and it
// naturally ignores soft-deleted holders, whose adminRoleId is unset on delete).
func (r *MongoRepository) CountAssigned(ctx context.Context, id bson.ObjectID) (int64, error) {
	return r.users.CountDocuments(ctx, bson.D{
		{Key: "role", Value: "admin"},
		{Key: "adminRoleId", Value: id},
	})
}
