package promo

import (
	"context"
	"regexp"
	"strings"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrDepleted is returned when a redemption would exceed a promo's usage limit.
// It wraps ErrBadRequest so handlers classify it as a 400.
var ErrDepleted = &apperrors.AppError{
	Code:    "PROMO_DEPLETED",
	Message: "promo code usage limit reached",
	Err:     apperrors.ErrBadRequest,
}

// Repository defines persistence operations for PromoCode entities. Codes are
// stored upper-cased; the repository normalizes on every read/write.
type Repository interface {
	FindByCode(ctx context.Context, code string) (*PromoCode, error)
	FindByID(ctx context.Context, id bson.ObjectID) (*PromoCode, error)
	List(ctx context.Context, f PromoFilter, p pagination.Params) ([]*PromoCode, int64, error)
	Create(ctx context.Context, promo *PromoCode) error
	Update(ctx context.Context, id bson.ObjectID, fields PromoUpdate) (*PromoCode, error)
	Delete(ctx context.Context, id bson.ObjectID) error
	IncrementUses(ctx context.Context, code string) error
}

// PromoUpdate carries the editable fields of a promo (admin edit).
type PromoUpdate struct {
	Code      string
	Type      PromoType
	Value     float64
	MinOrder  float64
	MaxUses   int
	StartsAt  *time.Time
	ExpiresAt *time.Time
	Active    bool
}

// PromoFilter narrows an admin promo listing. Zero-valued fields are ignored.
// Status is a derived bucket (active/paused/expired/depleted) translated to
// document conditions.
type PromoFilter struct {
	Type   string
	Status string
	Search string
}

// notDepleted matches promos that have not hit their usage cap (unlimited when
// maxUses is 0).
var notDepleted = bson.D{{Key: "$or", Value: bson.A{
	bson.D{{Key: "$eq", Value: bson.A{"$maxUses", 0}}},
	bson.D{{Key: "$lt", Value: bson.A{"$uses", "$maxUses"}}},
}}}

// build assembles the MongoDB filter document for f.
func (f PromoFilter) build() bson.D {
	and := bson.A{}
	if f.Type != "" {
		and = append(and, bson.D{{Key: "type", Value: f.Type}})
	}
	if f.Search != "" {
		rx := bson.D{{Key: "$regex", Value: regexp.QuoteMeta(strings.ToUpper(f.Search))}, {Key: "$options", Value: "i"}}
		and = append(and, bson.D{{Key: "code", Value: rx}})
	}
	now := time.Now().UTC()
	switch f.Status {
	case "active":
		and = append(and,
			bson.D{{Key: "active", Value: true}},
			bson.D{{Key: "$or", Value: bson.A{
				bson.D{{Key: "expiresAt", Value: nil}},
				bson.D{{Key: "expiresAt", Value: bson.D{{Key: "$gt", Value: now}}}},
			}}},
			bson.D{{Key: "$expr", Value: notDepleted}},
		)
	case "paused":
		and = append(and, bson.D{{Key: "active", Value: false}})
	case "expired":
		and = append(and, bson.D{{Key: "expiresAt", Value: bson.D{{Key: "$ne", Value: nil}, {Key: "$lt", Value: now}}}})
	case "depleted":
		and = append(and, bson.D{{Key: "$expr", Value: bson.D{{Key: "$and", Value: bson.A{
			bson.D{{Key: "$gt", Value: bson.A{"$maxUses", 0}}},
			bson.D{{Key: "$gte", Value: bson.A{"$uses", "$maxUses"}}},
		}}}}})
	}
	if len(and) == 0 {
		return bson.D{}
	}
	return bson.D{{Key: "$and", Value: and}}
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository using the given collection.
func NewMongoRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: col}
}

// EnsureIndexes creates the unique index on promo code.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("promos").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "code", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// FindByCode retrieves a promo by its (upper-cased) code, ErrNotFound when absent.
func (r *MongoRepository) FindByCode(ctx context.Context, code string) (*PromoCode, error) {
	var p PromoCode
	err := r.collection.FindOne(ctx, bson.D{{Key: "code", Value: strings.ToUpper(code)}}).Decode(&p)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// FindByID retrieves a promo by ObjectID, ErrNotFound when absent.
func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*PromoCode, error) {
	var p PromoCode
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&p)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// List returns a paginated slice of promos matching f, newest first.
func (r *MongoRepository) List(ctx context.Context, f PromoFilter, p pagination.Params) ([]*PromoCode, int64, error) {
	filter := f.build()
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*PromoCode{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// Create inserts a new promo, stamping CreatedAt and upper-casing the code. A
// duplicate code (unique index violation) is surfaced as ErrConflict.
func (r *MongoRepository) Create(ctx context.Context, promo *PromoCode) error {
	if promo.ID.IsZero() {
		promo.ID = bson.NewObjectID()
	}
	if promo.CreatedAt.IsZero() {
		promo.CreatedAt = time.Now().UTC()
	}
	promo.Code = strings.ToUpper(promo.Code)
	if _, err := r.collection.InsertOne(ctx, promo); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return apperrors.ErrConflict
		}
		return err
	}
	return nil
}

// Update atomically sets a promo's editable fields and returns the post-update
// document. ErrNotFound when no promo matches; ErrConflict on a code collision.
func (r *MongoRepository) Update(ctx context.Context, id bson.ObjectID, fields PromoUpdate) (*PromoCode, error) {
	var p PromoCode
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "code", Value: strings.ToUpper(fields.Code)},
			{Key: "type", Value: fields.Type},
			{Key: "value", Value: fields.Value},
			{Key: "minOrder", Value: fields.MinOrder},
			{Key: "maxUses", Value: fields.MaxUses},
			{Key: "startsAt", Value: fields.StartsAt},
			{Key: "expiresAt", Value: fields.ExpiresAt},
			{Key: "active", Value: fields.Active},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&p)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		if mongo.IsDuplicateKeyError(err) {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &p, nil
}

// Delete removes a promo by ObjectID, ErrNotFound when absent.
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

// IncrementUses atomically bumps a code's use count, but only while it is active
// and below its cap (unlimited when maxUses is 0). A code that is missing,
// inactive, or at its cap matches nothing and returns ErrDepleted — the
// authoritative guard against concurrent over-redemption.
func (r *MongoRepository) IncrementUses(ctx context.Context, code string) error {
	filter := bson.D{
		{Key: "code", Value: strings.ToUpper(code)},
		{Key: "active", Value: true},
		{Key: "$expr", Value: notDepleted},
	}
	res, err := r.collection.UpdateOne(ctx, filter, bson.D{{Key: "$inc", Value: bson.D{{Key: "uses", Value: 1}}}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrDepleted
	}
	return nil
}
