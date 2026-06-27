package offer

import (
	"context"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository defines persistence operations for Offer entities.
type Repository interface {
	FindByID(ctx context.Context, id bson.ObjectID) (*Offer, error)
	List(ctx context.Context, f OfferFilter, p pagination.Params) ([]*Offer, int64, error)
	ListLive(ctx context.Context, now time.Time) ([]*Offer, error)
	FindLiveByProduct(ctx context.Context, productID bson.ObjectID, now time.Time) (*Offer, error)
	Create(ctx context.Context, o *Offer) error
	Update(ctx context.Context, id bson.ObjectID, fields OfferUpdate) (*Offer, error)
	Delete(ctx context.Context, id bson.ObjectID) error
}

// OfferUpdate carries the editable fields of an offer (admin edit).
type OfferUpdate struct {
	ProductID     bson.ObjectID
	DiscountType  DiscountType
	DiscountValue float64
	StartsAt      *time.Time
	EndsAt        *time.Time
	Active        bool
	SortOrder     int
}

// OfferFilter narrows an admin offer listing. Zero-valued fields are ignored.
// Status is a derived bucket (active/scheduled/expired/paused) translated to
// document conditions.
type OfferFilter struct {
	Status    string
	ProductID string
}

// liveConds returns the document conditions that match an offer that is live at
// now: active, started, and not yet ended (open-ended bounds count as satisfied).
func liveConds(now time.Time) bson.A {
	return bson.A{
		bson.D{{Key: "active", Value: true}},
		bson.D{{Key: "$or", Value: bson.A{
			bson.D{{Key: "startsAt", Value: nil}},
			bson.D{{Key: "startsAt", Value: bson.D{{Key: "$lte", Value: now}}}},
		}}},
		bson.D{{Key: "$or", Value: bson.A{
			bson.D{{Key: "endsAt", Value: nil}},
			bson.D{{Key: "endsAt", Value: bson.D{{Key: "$gt", Value: now}}}},
		}}},
	}
}

// build assembles the MongoDB filter document for f.
func (f OfferFilter) build() bson.D {
	and := bson.A{}
	if f.ProductID != "" {
		if oid, err := bson.ObjectIDFromHex(f.ProductID); err == nil {
			and = append(and, bson.D{{Key: "productId", Value: oid}})
		}
	}
	now := time.Now().UTC()
	switch f.Status {
	case "active":
		and = append(and, liveConds(now)...)
	case "scheduled":
		and = append(and,
			bson.D{{Key: "active", Value: true}},
			bson.D{{Key: "startsAt", Value: bson.D{{Key: "$ne", Value: nil}, {Key: "$gt", Value: now}}}},
		)
	case "expired":
		and = append(and, bson.D{{Key: "endsAt", Value: bson.D{{Key: "$ne", Value: nil}, {Key: "$lt", Value: now}}}})
	case "paused":
		and = append(and, bson.D{{Key: "active", Value: false}})
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

// EnsureIndexes creates the offer indexes: a lookup on productId and a compound
// index supporting the live-listing sort.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("offers").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "productId", Value: 1}}},
		{Keys: bson.D{{Key: "active", Value: 1}, {Key: "sortOrder", Value: 1}}},
	})
	return err
}

// FindByID retrieves an offer by ObjectID, ErrNotFound when absent.
func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*Offer, error) {
	var o Offer
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&o)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// List returns a paginated slice of offers matching f, ordered by sortOrder then
// newest first.
func (r *MongoRepository) List(ctx context.Context, f OfferFilter, p pagination.Params) ([]*Offer, int64, error) {
	filter := f.build()
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: -1}})
	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*Offer{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// ListLive returns every offer that is live at now, ordered by sortOrder.
func (r *MongoRepository) ListLive(ctx context.Context, now time.Time) ([]*Offer, error) {
	filter := bson.D{{Key: "$and", Value: liveConds(now)}}
	opts := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: -1}})
	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*Offer{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FindLiveByProduct returns the live offer for a product (lowest sortOrder wins),
// or ErrNotFound when the product has none.
func (r *MongoRepository) FindLiveByProduct(ctx context.Context, productID bson.ObjectID, now time.Time) (*Offer, error) {
	conds := append(bson.A{bson.D{{Key: "productId", Value: productID}}}, liveConds(now)...)
	filter := bson.D{{Key: "$and", Value: conds}}
	opts := options.FindOne().SetSort(bson.D{{Key: "sortOrder", Value: 1}})
	var o Offer
	err := r.collection.FindOne(ctx, filter, opts).Decode(&o)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// Create inserts a new offer, stamping CreatedAt/UpdatedAt.
func (r *MongoRepository) Create(ctx context.Context, o *Offer) error {
	if o.ID.IsZero() {
		o.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = now
	}
	o.UpdatedAt = now
	_, err := r.collection.InsertOne(ctx, o)
	return err
}

// Update atomically sets an offer's editable fields and returns the post-update
// document. ErrNotFound when no offer matches.
func (r *MongoRepository) Update(ctx context.Context, id bson.ObjectID, fields OfferUpdate) (*Offer, error) {
	var o Offer
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "productId", Value: fields.ProductID},
			{Key: "discountType", Value: fields.DiscountType},
			{Key: "discountValue", Value: fields.DiscountValue},
			{Key: "startsAt", Value: fields.StartsAt},
			{Key: "endsAt", Value: fields.EndsAt},
			{Key: "active", Value: fields.Active},
			{Key: "sortOrder", Value: fields.SortOrder},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&o)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// Delete removes an offer by ObjectID, ErrNotFound when absent.
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
