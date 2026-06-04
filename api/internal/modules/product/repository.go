package product

import (
	"context"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository defines the persistence contract for the product domain.
type Repository interface {
	FindAll(ctx context.Context, f ListFilter, p pagination.Params) ([]Product, int64, error)
	FindByID(ctx context.Context, id string) (*Product, error)
	Create(ctx context.Context, in CreateProductInput) (*Product, error)
	Update(ctx context.Context, id string, in UpdateProductInput) (*Product, error)
	Delete(ctx context.Context, id string) error
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	col *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository using the "products" collection.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{col: db.Collection("products")}
}

// EnsureIndexes creates the required indexes on the products collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	col := db.Collection("products")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "category", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "available", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

// buildFilter converts a ListFilter into a bson.D suitable for MongoDB queries.
func buildFilter(f ListFilter) bson.D {
	filter := bson.D{}
	if f.Category != "" {
		filter = append(filter, bson.E{Key: "category", Value: f.Category})
	}
	if f.Available != nil {
		filter = append(filter, bson.E{Key: "available", Value: *f.Available})
	}
	return filter
}

// FindAll retrieves a paginated list of products matching the given filter.
func (r *MongoRepository) FindAll(ctx context.Context, f ListFilter, p pagination.Params) ([]Product, int64, error) {
	filter := buildFilter(f)

	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := pagination.Skip(p)
	limit := int64(p.Limit)

	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var products []Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// FindByID retrieves a single product by its hex ObjectID string.
// Returns apperrors.ErrNotFound when no document exists for the given id.
func (r *MongoRepository) FindByID(ctx context.Context, id string) (*Product, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}

	var p Product
	err = r.col.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&p)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}

	return &p, nil
}

// Create inserts a new product document built from the supplied input.
// Variant IDs are generated automatically; CreatedAt and UpdatedAt are stamped now.
func (r *MongoRepository) Create(ctx context.Context, in CreateProductInput) (*Product, error) {
	now := time.Now().UTC()

	// Generate fresh ObjectIDs for every variant that lacks one.
	variants := make([]Variant, len(in.Variants))
	for i, v := range in.Variants {
		if v.ID.IsZero() {
			v.ID = bson.NewObjectID()
		}
		variants[i] = v
	}

	p := Product{
		ID:              bson.NewObjectID(),
		Title:           in.Title,
		Category:        in.Category,
		Images:          in.Images,
		Variants:        variants,
		FulfillmentType: in.FulfillmentType,
		Stock:           in.Stock,
		Available:       in.Available,
		Ratings:         in.Ratings,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if _, err := r.col.InsertOne(ctx, p); err != nil {
		return nil, err
	}

	return &p, nil
}

// Update applies a partial update to the product identified by id.
// Only non-nil fields in UpdateProductInput are written; UpdatedAt is always refreshed.
func (r *MongoRepository) Update(ctx context.Context, id string, in UpdateProductInput) (*Product, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}

	set := bson.D{{Key: "updatedAt", Value: time.Now().UTC()}}

	if in.Title != nil {
		set = append(set, bson.E{Key: "title", Value: in.Title})
	}
	if in.Category != nil {
		set = append(set, bson.E{Key: "category", Value: *in.Category})
	}
	if in.Images != nil {
		set = append(set, bson.E{Key: "images", Value: in.Images})
	}
	if in.Variants != nil {
		set = append(set, bson.E{Key: "variants", Value: in.Variants})
	}
	if in.FulfillmentType != nil {
		set = append(set, bson.E{Key: "fulfillmentType", Value: *in.FulfillmentType})
	}
	if in.Stock != nil {
		set = append(set, bson.E{Key: "stock", Value: *in.Stock})
	}
	if in.Available != nil {
		set = append(set, bson.E{Key: "available", Value: *in.Available})
	}
	if in.Ratings != nil {
		set = append(set, bson.E{Key: "ratings", Value: *in.Ratings})
	}

	after := options.After
	opts := options.FindOneAndUpdate().SetReturnDocument(after)

	var updated Product
	err = r.col.FindOneAndUpdate(
		ctx,
		bson.D{{Key: "_id", Value: oid}},
		bson.D{{Key: "$set", Value: set}},
		opts,
	).Decode(&updated)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}

	return &updated, nil
}

// Delete removes the product identified by id.
func (r *MongoRepository) Delete(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return apperrors.ErrNotFound
	}

	res, err := r.col.DeleteOne(ctx, bson.D{{Key: "_id", Value: oid}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}
