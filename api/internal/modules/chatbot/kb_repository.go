package chatbot

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var errDuplicateSlug = errors.New("duplicate slug")

// KBRepository persists knowledge-base docs (collection kb_docs).
type KBRepository interface {
	List(ctx context.Context, enabledOnly bool) ([]KBDoc, error)
	Get(ctx context.Context, id bson.ObjectID) (*KBDoc, error)
	Create(ctx context.Context, d *KBDoc) error
	Replace(ctx context.Context, d *KBDoc) error
	Delete(ctx context.Context, id bson.ObjectID) (*KBDoc, error)
	// InsertIfSlugMissing inserts d unless a doc with d.Slug already exists,
	// reporting whether it inserted (the idempotent seed path).
	InsertIfSlugMissing(ctx context.Context, d *KBDoc) (bool, error)
}

// MongoKBRepository is the MongoDB-backed KBRepository.
type MongoKBRepository struct{ col *mongo.Collection }

// NewMongoKBRepository builds the repository over kb_docs.
func NewMongoKBRepository(db *mongo.Database) *MongoKBRepository {
	return &MongoKBRepository{col: db.Collection("kb_docs")}
}

// EnsureKBIndexes creates the list-order index and the sparse-unique slug.
func EnsureKBIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("kb_docs").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: 1}}},
		{Keys: bson.D{{Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
	})
	return err
}

// List returns docs ordered by sortOrder, then creation time (a stable order,
// which keeps the model prompt — and so its provider-side cache — stable).
func (r *MongoKBRepository) List(ctx context.Context, enabledOnly bool) ([]KBDoc, error) {
	filter := bson.D{}
	if enabledOnly {
		filter = append(filter, bson.E{Key: "enabled", Value: true})
	}
	cur, err := r.col.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: 1}, {Key: "_id", Value: 1}}).
		SetLimit(maxKBDocs*2))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []KBDoc{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get loads one doc or errNotFound.
func (r *MongoKBRepository) Get(ctx context.Context, id bson.ObjectID) (*KBDoc, error) {
	var d KBDoc
	if err := r.col.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&d); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errNotFound
		}
		return nil, err
	}
	return &d, nil
}

// Create inserts a new doc (errDuplicateSlug on a slug clash).
func (r *MongoKBRepository) Create(ctx context.Context, d *KBDoc) error {
	_, err := r.col.InsertOne(ctx, d)
	if mongo.IsDuplicateKeyError(err) {
		return errDuplicateSlug
	}
	return err
}

// Replace overwrites an existing doc (errNotFound when missing,
// errDuplicateSlug on a slug clash).
func (r *MongoKBRepository) Replace(ctx context.Context, d *KBDoc) error {
	res, err := r.col.ReplaceOne(ctx, bson.D{{Key: "_id", Value: d.ID}}, d)
	if mongo.IsDuplicateKeyError(err) {
		return errDuplicateSlug
	}
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errNotFound
	}
	return nil
}

// Delete removes a doc, returning it (errNotFound when missing).
func (r *MongoKBRepository) Delete(ctx context.Context, id bson.ObjectID) (*KBDoc, error) {
	var gone KBDoc
	if err := r.col.FindOneAndDelete(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&gone); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errNotFound
		}
		return nil, err
	}
	return &gone, nil
}

// InsertIfSlugMissing implements KBRepository with a document-atomic upsert
// ($setOnInsert only), so an existing — possibly admin-edited — doc is never
// touched.
func (r *MongoKBRepository) InsertIfSlugMissing(ctx context.Context, d *KBDoc) (bool, error) {
	if d.Slug == "" {
		return false, errors.New("slug required")
	}
	res, err := r.col.UpdateOne(ctx,
		bson.D{{Key: "slug", Value: d.Slug}},
		bson.D{{Key: "$setOnInsert", Value: d}},
		options.UpdateOne().SetUpsert(true))
	if err != nil {
		return false, err
	}
	return res.UpsertedCount > 0, nil
}
