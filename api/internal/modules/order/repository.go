package order

import (
	"context"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository defines persistence operations for the Order entity.
type Repository interface {
	FindByID(ctx context.Context, id bson.ObjectID) (*Order, error)
	FindByUserID(ctx context.Context, userID bson.ObjectID) ([]*Order, error)
	FindByIdempotencyKey(ctx context.Context, userID bson.ObjectID, key string) (*Order, error)
	Create(ctx context.Context, order *Order) error
	UpdateStatus(ctx context.Context, id bson.ObjectID, status OrderStatus) error
	UpdateFulfillment(ctx context.Context, id bson.ObjectID, status OrderStatus, fulfillment Fulfillment) error
	ListAll(ctx context.Context, p pagination.Params) ([]*Order, int64, error)
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository using the given collection.
func NewMongoRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: col}
}

// EnsureIndexes creates the indexes the order module relies on: a lookup index
// on (userId, createdAt) for order history, and a partial-unique index on
// (userId, idempotencyKey) that guards against duplicate submissions while
// allowing the many orders that carry no key.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	orders := db.Collection("orders")
	_, err := orders.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
		{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "idempotencyKey", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "idempotencyKey", Value: bson.D{{Key: "$exists", Value: true}}}}),
		},
	})
	return err
}

// FindByID retrieves a single order by ObjectID, returning ErrNotFound when absent.
func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*Order, error) {
	var o Order
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&o)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// FindByUserID returns a user's orders, newest first.
func (r *MongoRepository) FindByUserID(ctx context.Context, userID bson.ObjectID) ([]*Order, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.collection.Find(ctx, bson.D{{Key: "userId", Value: userID}}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*Order{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FindByIdempotencyKey returns the order a user previously created under key, or
// ErrNotFound when none exists.
func (r *MongoRepository) FindByIdempotencyKey(ctx context.Context, userID bson.ObjectID, key string) (*Order, error) {
	var o Order
	err := r.collection.FindOne(ctx, bson.D{
		{Key: "userId", Value: userID},
		{Key: "idempotencyKey", Value: key},
	}).Decode(&o)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// Create inserts a new order, stamping CreatedAt/UpdatedAt. A duplicate
// (userId, idempotencyKey) violation is surfaced as ErrConflict so the caller
// can re-fetch the winning order.
func (r *MongoRepository) Create(ctx context.Context, order *Order) error {
	now := time.Now().UTC()
	if order.ID.IsZero() {
		order.ID = bson.NewObjectID()
	}
	order.CreatedAt = now
	order.UpdatedAt = now
	if order.Items == nil {
		order.Items = []OrderItem{}
	}
	if order.Fulfillment.StatusTimeline == nil {
		order.Fulfillment.StatusTimeline = []TimelineEvent{}
	}

	if _, err := r.collection.InsertOne(ctx, order); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return apperrors.ErrConflict
		}
		return err
	}
	return nil
}

// UpdateStatus sets the order's status and refreshes UpdatedAt.
func (r *MongoRepository) UpdateStatus(ctx context.Context, id bson.ObjectID, status OrderStatus) error {
	res, err := r.collection.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: status},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// UpdateFulfillment sets the order's status and fulfillment in a single update.
func (r *MongoRepository) UpdateFulfillment(ctx context.Context, id bson.ObjectID, status OrderStatus, fulfillment Fulfillment) error {
	res, err := r.collection.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: status},
			{Key: "fulfillment", Value: fulfillment},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// ListAll returns a paginated slice of every order (admin), newest first.
func (r *MongoRepository) ListAll(ctx context.Context, p pagination.Params) ([]*Order, int64, error) {
	total, err := r.collection.CountDocuments(ctx, bson.D{})
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.collection.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*Order{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
