package category

import (
	"context"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository is the persistence contract for the category taxonomy.
type Repository interface {
	FindAll(ctx context.Context, f CategoryFilter) ([]Category, error)
	Upsert(ctx context.Context, in UpsertCategoryInput) (*Category, error)
	FindByLegacyID(ctx context.Context, legacyID int) (*Category, error)
}

// MongoRepository is a MongoDB-backed Repository over the "categories" collection.
type MongoRepository struct {
	col *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{col: db.Collection("categories")}
}

// EnsureIndexes creates the unique legacyId index plus slug / parent lookups.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	col := db.Collection("categories")
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "legacyId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "slug", Value: 1}}},
		{Keys: bson.D{{Key: "parentLegacyId", Value: 1}}},
		{Keys: bson.D{{Key: "rootDomain", Value: 1}}},
		{Keys: bson.D{{Key: "depth", Value: 1}}},
	})
	return err
}

// FindAll returns categories matching f, sorted by sortOrder then legacyId.
func (r *MongoRepository) FindAll(ctx context.Context, f CategoryFilter) ([]Category, error) {
	filter := bson.D{}
	if f.VisibleOnly {
		filter = append(filter, bson.E{Key: "visible", Value: true})
	}
	if f.Depth != nil {
		filter = append(filter, bson.E{Key: "depth", Value: *f.Depth})
	}
	if f.RootDomain != "" {
		filter = append(filter, bson.E{Key: "rootDomain", Value: f.RootDomain})
	}
	if f.ParentLegacyID != nil {
		filter = append(filter, bson.E{Key: "parentLegacyId", Value: *f.ParentLegacyID})
	}

	opts := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "legacyId", Value: 1}})
	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var cats []Category
	if err := cursor.All(ctx, &cats); err != nil {
		return nil, err
	}
	return cats, nil
}

// FindByLegacyID retrieves a category by its legacyId, or ErrNotFound.
func (r *MongoRepository) FindByLegacyID(ctx context.Context, legacyID int) (*Category, error) {
	var c Category
	err := r.col.FindOne(ctx, bson.D{{Key: "legacyId", Value: legacyID}}).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// Upsert inserts or updates a category keyed on legacyId. _id/createdAt/legacyId
// are insert-only; the rest is refreshed each run.
func (r *MongoRepository) Upsert(ctx context.Context, in UpsertCategoryInput) (*Category, error) {
	now := time.Now().UTC()
	set := bson.D{
		{Key: "updatedAt", Value: now},
		{Key: "parentLegacyId", Value: in.ParentLegacyID},
		{Key: "slug", Value: in.Slug},
		{Key: "name", Value: in.Name},
		{Key: "image", Value: in.Image},
		{Key: "sortOrder", Value: in.SortOrder},
		{Key: "rootDomain", Value: in.RootDomain},
		{Key: "depth", Value: in.Depth},
		{Key: "visible", Value: in.Visible},
	}
	setOnInsert := bson.D{
		{Key: "_id", Value: bson.NewObjectID()},
		{Key: "createdAt", Value: now},
		{Key: "legacyId", Value: in.LegacyID},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var updated Category
	err := r.col.FindOneAndUpdate(ctx,
		bson.D{{Key: "legacyId", Value: in.LegacyID}},
		bson.D{{Key: "$set", Value: set}, {Key: "$setOnInsert", Value: setOnInsert}},
		opts,
	).Decode(&updated)
	if err != nil {
		return nil, err
	}
	return &updated, nil
}
