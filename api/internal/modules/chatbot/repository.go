package chatbot

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var errNotFound = errors.New("not found")

// maxLegacyEntries bounds the legacy quick-question read.
const maxLegacyEntries = 500

// Repository reads the legacy quick questions (collection chatbot_entries).
type Repository interface {
	List(ctx context.Context, enabledOnly bool) ([]Entry, error)
}

// MongoRepository is the MongoDB-backed Repository.
type MongoRepository struct{ col *mongo.Collection }

// NewMongoRepository builds the repository over chatbot_entries.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{col: db.Collection("chatbot_entries")}
}

// List returns entries ordered by sortOrder, then creation time.
func (r *MongoRepository) List(ctx context.Context, enabledOnly bool) ([]Entry, error) {
	filter := bson.D{}
	if enabledOnly {
		filter = append(filter, bson.E{Key: "enabled", Value: true})
	}
	cur, err := r.col.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: 1}}).
		SetLimit(maxLegacyEntries))
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
