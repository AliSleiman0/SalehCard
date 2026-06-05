package code

import (
	"context"
	"regexp"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrOutOfStock is returned when no available code can be claimed for a product.
var ErrOutOfStock = &apperrors.AppError{
	Code:    "OUT_OF_STOCK",
	Message: "no codes available for this product",
	Err:     apperrors.ErrConflict,
}

// Repository is the persistence contract for the code/inventory domain.
type Repository interface {
	BulkInsert(ctx context.Context, productID string, items []UploadItem, batch string) (inserted, duplicates int, err error)
	ListByProduct(ctx context.Context, productID string, status Status, p pagination.Params) ([]Code, int64, error)
	CountsByProduct(ctx context.Context, productID string) (map[Status]int, error)
	FindByCodeOrSuffix(ctx context.Context, value string) (*Code, error)
	CodeProducts(ctx context.Context) ([]ProductMeta, error)
	ProductMeta(ctx context.Context, productID string) (*ProductMeta, error)
	GetThreshold(ctx context.Context, productID string) (int, error)
	SetThreshold(ctx context.Context, productID string, threshold int) error
	SetProductStock(ctx context.Context, productID string, stock int) error
	// ClaimOne atomically marks one available code for the product as delivered,
	// stamping the order/recipient. Returns ErrOutOfStock when none is available.
	ClaimOne(ctx context.Context, productID, orderID, deliveredTo string, now time.Time) (*Code, error)
	// ReleaseByOrder reverses a claim, returning the delivered codes of an order
	// to the available pool. Used to compensate a failed order.
	ReleaseByOrder(ctx context.Context, orderID string) error
}

// ProductMeta is the minimal product info the inventory views need.
type ProductMeta struct {
	ID       string
	Title    string
	Category string
}

// MongoRepository is the MongoDB-backed Repository.
type MongoRepository struct {
	codes      *mongo.Collection
	products   *mongo.Collection
	thresholds *mongo.Collection
}

// NewMongoRepository builds a MongoRepository over the codes/products/thresholds
// collections.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		codes:      db.Collection("codes"),
		products:   db.Collection("products"),
		thresholds: db.Collection("inventory_thresholds"),
	}
}

// EnsureIndexes creates the indexes the code module relies on.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	codes := db.Collection("codes")
	_, err := codes.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "productId", Value: 1}, {Key: "code", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "code", Value: 1}}},
		{Keys: bson.D{{Key: "productId", Value: 1}, {Key: "status", Value: 1}}},
	})
	if err != nil {
		return err
	}
	_, err = db.Collection("inventory_thresholds").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "productId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// BulkInsert inserts codes that are new to the product, skipping rows that
// duplicate an existing code or repeat within the same batch.
func (r *MongoRepository) BulkInsert(ctx context.Context, productID string, items []UploadItem, batch string) (int, int, error) {
	// Load existing codes for this product into a set.
	existing := map[string]struct{}{}
	cur, err := r.codes.Find(ctx, bson.D{{Key: "productId", Value: productID}}, options.Find().SetProjection(bson.D{{Key: "code", Value: 1}}))
	if err != nil {
		return 0, 0, err
	}
	var rows []struct {
		Code string `bson:"code"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, 0, err
	}
	for _, row := range rows {
		existing[row.Code] = struct{}{}
	}

	now := time.Now().UTC()
	docs := make([]interface{}, 0, len(items))
	duplicates := 0
	for _, it := range items {
		if it.Code == "" {
			continue
		}
		if _, dup := existing[it.Code]; dup {
			duplicates++
			continue
		}
		existing[it.Code] = struct{}{} // guard in-batch repeats
		docs = append(docs, Code{
			ID:        bson.NewObjectID(),
			ProductID: productID,
			Code:      it.Code,
			Pin:       it.Pin,
			Status:    StatusAvailable,
			Batch:     batch,
			CreatedAt: now,
		})
	}

	if len(docs) > 0 {
		if _, err := r.codes.InsertMany(ctx, docs); err != nil {
			return 0, duplicates, err
		}
	}
	return len(docs), duplicates, nil
}

// ListByProduct returns a paginated slice of a product's codes, optionally
// filtered by status.
func (r *MongoRepository) ListByProduct(ctx context.Context, productID string, status Status, p pagination.Params) ([]Code, int64, error) {
	filter := bson.D{{Key: "productId", Value: productID}}
	if status != "" {
		filter = append(filter, bson.E{Key: "status", Value: status})
	}

	total, err := r.codes.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cur, err := r.codes.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var out []Code
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// CountsByProduct returns per-status counts for a single product.
func (r *MongoRepository) CountsByProduct(ctx context.Context, productID string) (map[Status]int, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "productId", Value: productID}}}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$status"}, {Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
	}
	return r.aggregateCounts(ctx, pipeline)
}

// aggregateCounts runs a status-grouping pipeline and returns a status→count map.
func (r *MongoRepository) aggregateCounts(ctx context.Context, pipeline mongo.Pipeline) (map[Status]int, error) {
	cur, err := r.codes.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		ID Status `bson:"_id"`
		N  int    `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := map[Status]int{}
	for _, row := range rows {
		out[row.ID] = row.N
	}
	return out, nil
}

// FindByCodeOrSuffix looks up a code by exact value, falling back to a
// case-insensitive suffix match (so "last 4 digits" lookups work).
func (r *MongoRepository) FindByCodeOrSuffix(ctx context.Context, value string) (*Code, error) {
	var c Code
	err := r.codes.FindOne(ctx, bson.D{{Key: "code", Value: value}}).Decode(&c)
	if err == nil {
		return &c, nil
	}
	if err != mongo.ErrNoDocuments {
		return nil, err
	}

	rx := bson.Regex{Pattern: regexp.QuoteMeta(value) + "$", Options: "i"}
	err = r.codes.FindOne(ctx, bson.D{{Key: "code", Value: rx}}).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// productDoc is the subset of a product document the inventory views read.
type productDoc struct {
	ID    bson.ObjectID `bson:"_id"`
	Title struct {
		En string `bson:"en"`
	} `bson:"title"`
	Category        string `bson:"category"`
	FulfillmentType string `bson:"fulfillmentType"`
}

// CodeProducts returns metadata for every code-type product.
func (r *MongoRepository) CodeProducts(ctx context.Context) ([]ProductMeta, error) {
	cur, err := r.products.Find(ctx, bson.D{{Key: "fulfillmentType", Value: "code"}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []productDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]ProductMeta, len(docs))
	for i, d := range docs {
		out[i] = ProductMeta{ID: d.ID.Hex(), Title: d.Title.En, Category: d.Category}
	}
	return out, nil
}

// ProductMeta returns metadata for a single product (any fulfillment type).
func (r *MongoRepository) ProductMeta(ctx context.Context, productID string) (*ProductMeta, error) {
	oid, err := bson.ObjectIDFromHex(productID)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}
	var d productDoc
	err = r.products.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&d)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &ProductMeta{ID: d.ID.Hex(), Title: d.Title.En, Category: d.Category}, nil
}

// GetThreshold returns the configured low-stock threshold, or DefaultThreshold.
func (r *MongoRepository) GetThreshold(ctx context.Context, productID string) (int, error) {
	var doc struct {
		Threshold int `bson:"threshold"`
	}
	err := r.thresholds.FindOne(ctx, bson.D{{Key: "productId", Value: productID}}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return DefaultThreshold, nil
		}
		return DefaultThreshold, err
	}
	return doc.Threshold, nil
}

// SetThreshold upserts the per-product low-stock threshold.
func (r *MongoRepository) SetThreshold(ctx context.Context, productID string, threshold int) error {
	_, err := r.thresholds.UpdateOne(ctx,
		bson.D{{Key: "productId", Value: productID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "threshold", Value: threshold}}}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// SetProductStock mirrors the available-code count onto the product's stock field.
func (r *MongoRepository) SetProductStock(ctx context.Context, productID string, stock int) error {
	oid, err := bson.ObjectIDFromHex(productID)
	if err != nil {
		return nil // non-fatal: best-effort mirror
	}
	_, err = r.products.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: oid}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "stock", Value: stock}, {Key: "updatedAt", Value: time.Now().UTC()}}}},
	)
	return err
}

// ClaimOne atomically transitions one available code to delivered. The single
// FindOneAndUpdate is the concurrency guard against double-selling — never
// select-then-update. Returns ErrOutOfStock when the product has no available code.
func (r *MongoRepository) ClaimOne(ctx context.Context, productID, orderID, deliveredTo string, now time.Time) (*Code, error) {
	var c Code
	err := r.codes.FindOneAndUpdate(ctx,
		bson.D{{Key: "productId", Value: productID}, {Key: "status", Value: StatusAvailable}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: StatusDelivered},
			{Key: "orderId", Value: orderID},
			{Key: "deliveredTo", Value: deliveredTo},
			{Key: "deliveredAt", Value: now},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrOutOfStock
		}
		return nil, err
	}
	return &c, nil
}

// ReleaseByOrder returns every delivered code claimed under orderID to the
// available pool, clearing its delivery metadata.
func (r *MongoRepository) ReleaseByOrder(ctx context.Context, orderID string) error {
	_, err := r.codes.UpdateMany(ctx,
		bson.D{{Key: "orderId", Value: orderID}, {Key: "status", Value: StatusDelivered}},
		bson.D{
			{Key: "$set", Value: bson.D{{Key: "status", Value: StatusAvailable}}},
			{Key: "$unset", Value: bson.D{{Key: "orderId", Value: ""}, {Key: "deliveredTo", Value: ""}, {Key: "deliveredAt", Value: ""}}},
		},
	)
	return err
}
