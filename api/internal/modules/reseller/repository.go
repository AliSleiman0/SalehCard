package reseller

import (
	"context"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository defines persistence operations for reseller tier definitions, plus
// a per-tier headcount drawn from the users collection.
type Repository interface {
	ListTiers(ctx context.Context) ([]*ResellerTier, error)
	FindTierByID(ctx context.Context, id bson.ObjectID) (*ResellerTier, error)
	FindTierByName(ctx context.Context, name string) (*ResellerTier, error)
	CreateTier(ctx context.Context, tier *ResellerTier) error
	UpdateTier(ctx context.Context, id bson.ObjectID, name string, marginPercent float64) (*ResellerTier, error)
	DeleteTier(ctx context.Context, id bson.ObjectID) error
	// CountByTier returns the number of reseller users per tier name.
	CountByTier(ctx context.Context) (map[string]int64, error)
	// MarginForUser resolves a user's reseller-tier margin percent (0 when the
	// user has no tier or the tier is unknown).
	MarginForUser(ctx context.Context, userID bson.ObjectID) (float64, error)
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	tiers *mongo.Collection
	users *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository over the reseller_tiers and
// users collections.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		tiers: db.Collection("reseller_tiers"),
		users: db.Collection("users"),
	}
}

// EnsureIndexes creates the unique index on tier name.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("reseller_tiers").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// ListTiers returns all tier definitions, ordered by margin ascending.
func (r *MongoRepository) ListTiers(ctx context.Context) ([]*ResellerTier, error) {
	opts := options.Find().SetSort(bson.D{{Key: "marginPercent", Value: 1}})
	cur, err := r.tiers.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*ResellerTier{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FindTierByID retrieves a tier by ObjectID, returning ErrNotFound when absent.
func (r *MongoRepository) FindTierByID(ctx context.Context, id bson.ObjectID) (*ResellerTier, error) {
	var t ResellerTier
	err := r.tiers.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&t)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// FindTierByName retrieves a tier by name, returning ErrNotFound when absent.
func (r *MongoRepository) FindTierByName(ctx context.Context, name string) (*ResellerTier, error) {
	var t ResellerTier
	err := r.tiers.FindOne(ctx, bson.D{{Key: "name", Value: name}}).Decode(&t)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// CreateTier inserts a new tier definition, stamping CreatedAt. A duplicate name
// (unique index violation) is surfaced as ErrConflict.
func (r *MongoRepository) CreateTier(ctx context.Context, tier *ResellerTier) error {
	if tier.ID.IsZero() {
		tier.ID = bson.NewObjectID()
	}
	if tier.CreatedAt.IsZero() {
		tier.CreatedAt = time.Now().UTC()
	}
	if _, err := r.tiers.InsertOne(ctx, tier); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return apperrors.ErrConflict
		}
		return err
	}
	return nil
}

// UpdateTier atomically updates a tier's fields and returns the post-update
// document, returning ErrNotFound when no tier matches.
func (r *MongoRepository) UpdateTier(ctx context.Context, id bson.ObjectID, name string, marginPercent float64) (*ResellerTier, error) {
	var t ResellerTier
	err := r.tiers.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "name", Value: name},
			{Key: "marginPercent", Value: marginPercent},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&t)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		if mongo.IsDuplicateKeyError(err) {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &t, nil
}

// DeleteTier removes a tier definition by ObjectID.
func (r *MongoRepository) DeleteTier(ctx context.Context, id bson.ObjectID) error {
	res, err := r.tiers.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// MarginForUser resolves the reseller margin percent for a user by looking up
// their assigned tier. It returns 0 (reseller pays retail — the pre-margin
// behavior) when the user has no tier, the tier is unknown, or the user is
// missing, so a stale/blank tier never blocks or mis-prices an order.
func (r *MongoRepository) MarginForUser(ctx context.Context, userID bson.ObjectID) (float64, error) {
	var u struct {
		ResellerTier string `bson:"resellerTier"`
	}
	err := r.users.FindOne(ctx,
		bson.D{{Key: "_id", Value: userID}},
		options.FindOne().SetProjection(bson.D{{Key: "resellerTier", Value: 1}}),
	).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		return 0, err
	}
	if u.ResellerTier == "" {
		return 0, nil
	}
	var t ResellerTier
	err = r.tiers.FindOne(ctx, bson.D{{Key: "name", Value: u.ResellerTier}}).Decode(&t)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		return 0, err
	}
	return t.MarginPercent, nil
}

// CountByTier aggregates the number of reseller users grouped by their tier name.
// Resellers with no tier assigned are grouped under the empty-string key.
func (r *MongoRepository) CountByTier(ctx context.Context) (map[string]int64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "role", Value: "reseller"}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$resellerTier"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	cur, err := r.users.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		Name  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, row := range rows {
		out[row.Name] = row.Count
	}
	return out, nil
}
