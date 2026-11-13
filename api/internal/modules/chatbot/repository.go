package chatbot

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var errNotFound = errors.New("not found")

// Repository persists chatbot entries (collection chatbot_entries).
type Repository interface {
	List(ctx context.Context, enabledOnly bool) ([]Entry, error)
	Get(ctx context.Context, id bson.ObjectID) (*Entry, error)
	Count(ctx context.Context) (int64, error)
	Create(ctx context.Context, e *Entry) error
	Replace(ctx context.Context, e *Entry) error
	// DeleteWithChildren removes the entry and its follow-ups, returning the
	// deleted entry and how many follow-ups went with it.
	DeleteWithChildren(ctx context.Context, id bson.ObjectID) (*Entry, int64, error)
	HasChildren(ctx context.Context, id bson.ObjectID) (bool, error)
}

// MongoRepository is the MongoDB-backed Repository.
type MongoRepository struct{ col *mongo.Collection }

// NewMongoRepository builds the repository over chatbot_entries.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{col: db.Collection("chatbot_entries")}
}

// EnsureIndexes creates the list-order index.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("chatbot_entries").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "parentId", Value: 1}, {Key: "sortOrder", Value: 1}}},
	})
	return err
}

// List returns entries ordered by sortOrder, then creation time.
func (r *MongoRepository) List(ctx context.Context, enabledOnly bool) ([]Entry, error) {
	filter := bson.D{}
	if enabledOnly {
		filter = append(filter, bson.E{Key: "enabled", Value: true})
	}
	cur, err := r.col.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: 1}}).
		SetLimit(maxEntries))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []Entry{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get loads one entry or errNotFound.
func (r *MongoRepository) Get(ctx context.Context, id bson.ObjectID) (*Entry, error) {
	var e Entry
	if err := r.col.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&e); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errNotFound
		}
		return nil, err
	}
	return &e, nil
}

// Count returns the number of entries (a sanity cap on growth).
func (r *MongoRepository) Count(ctx context.Context) (int64, error) {
	return r.col.CountDocuments(ctx, bson.D{})
}

// Create inserts a new entry.
func (r *MongoRepository) Create(ctx context.Context, e *Entry) error {
	_, err := r.col.InsertOne(ctx, e)
	return err
}

// Replace overwrites an existing entry (errNotFound when missing).
func (r *MongoRepository) Replace(ctx context.Context, e *Entry) error {
	res, err := r.col.ReplaceOne(ctx, bson.D{{Key: "_id", Value: e.ID}}, e)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errNotFound
	}
	return nil
}

// DeleteWithChildren implements Repository.
func (r *MongoRepository) DeleteWithChildren(ctx context.Context, id bson.ObjectID) (*Entry, int64, error) {
	var gone Entry
	if err := r.col.FindOneAndDelete(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&gone); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, 0, errNotFound
		}
		return nil, 0, err
	}
	res, err := r.col.DeleteMany(ctx, bson.D{{Key: "parentId", Value: id}})
	if err != nil {
		return &gone, 0, err
	}
	return &gone, res.DeletedCount, nil
}

// HasChildren reports whether any entry follows up on id.
func (r *MongoRepository) HasChildren(ctx context.Context, id bson.ObjectID) (bool, error) {
	n, err := r.col.CountDocuments(ctx, bson.D{{Key: "parentId", Value: id}}, options.Count().SetLimit(1))
	return n > 0, err
}
