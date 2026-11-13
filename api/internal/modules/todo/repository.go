package todo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// errNotFound is returned when no to-do matches an id.
var errNotFound = errors.New("to-do not found")

// Repository persists to-dos.
type Repository interface {
	List(ctx context.Context, now time.Time) ([]Todo, error)
	Create(ctx context.Context, t *Todo) error
	Update(ctx context.Context, id bson.ObjectID, update bson.D) (*Todo, error)
	Delete(ctx context.Context, id bson.ObjectID) (*Todo, error)
}

// MongoRepository is the MongoDB-backed Repository (collection admin_todos).
type MongoRepository struct{ col *mongo.Collection }

// NewMongoRepository returns a repository over db.admin_todos.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{col: db.Collection("admin_todos")}
}

// EnsureIndexes creates the list index (open/done split, newest first).
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("admin_todos").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "done", Value: 1}, {Key: "createdAt", Value: -1}},
	})
	return err
}

// List returns every open task plus tasks completed since now-doneRetention,
// sorted for display (see sortTodos).
func (r *MongoRepository) List(ctx context.Context, now time.Time) ([]Todo, error) {
	open, err := r.find(ctx, bson.D{{Key: "done", Value: false}}, openLimit)
	if err != nil {
		return nil, err
	}
	done, err := r.find(ctx, bson.D{
		{Key: "done", Value: true},
		{Key: "doneAt", Value: bson.D{{Key: "$gte", Value: now.Add(-doneRetention)}}},
	}, doneLimit)
	if err != nil {
		return nil, err
	}
	out := append(open, done...)
	sortTodos(out)
	return out, nil
}

func (r *MongoRepository) find(ctx context.Context, filter bson.D, limit int64) ([]Todo, error) {
	cur, err := r.col.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(limit))
	if err != nil {
		return nil, err
	}
	out := []Todo{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Create inserts t.
func (r *MongoRepository) Create(ctx context.Context, t *Todo) error {
	_, err := r.col.InsertOne(ctx, t)
	return err
}

// Update applies update to the to-do and returns the result.
func (r *MongoRepository) Update(ctx context.Context, id bson.ObjectID, update bson.D) (*Todo, error) {
	var out Todo
	err := r.col.FindOneAndUpdate(ctx, bson.D{{Key: "_id", Value: id}}, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes the to-do and returns what was deleted (for the audit log).
func (r *MongoRepository) Delete(ctx context.Context, id bson.ObjectID) (*Todo, error) {
	var out Todo
	err := r.col.FindOneAndDelete(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}
