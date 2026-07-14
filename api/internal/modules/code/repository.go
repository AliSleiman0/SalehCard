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

// ErrDuplicateCode is returned when adding or editing a code would collide with
// an existing code for the same product (the (productId, code) unique index).
var ErrDuplicateCode = &apperrors.AppError{
	Code:    "CODE_DUPLICATE",
	Message: "a code with this value already exists for this product",
	Err:     apperrors.ErrConflict,
}

// Repository is the persistence contract for the code/inventory domain.
type Repository interface {
	BulkInsert(ctx context.Context, productID string, items []UploadItem, batch string) (inserted, duplicates int, err error)
	// InsertOne inserts a single available code for the product, returning the
	// created document. A collision with an existing code (unique index) returns
	// ErrDuplicateCode.
	InsertOne(ctx context.Context, productID string, item UploadItem, batch string) (*Code, error)
	// FindByID returns one code by its ObjectID (scoped to the product), or
	// ErrNotFound for a bad hex / missing document. Used to disambiguate a failed
	// available-only mutation (not-found vs not-available).
	FindByID(ctx context.Context, productID, codeID string) (*Code, error)
	// UpdateAvailableCode edits the value/pin of an available code (matched by
	// id + product), returning the updated document. A value collision returns
	// ErrDuplicateCode; no matching available code returns ErrNotFound.
	UpdateAvailableCode(ctx context.Context, productID, codeID, newCode, newPin string) (*Code, error)
	// DeleteAvailableCode removes a code that is not delivered (available or
	// expired), returning the deleted document, or ErrNotFound if none matches.
	DeleteAvailableCode(ctx context.Context, productID, codeID string) (*Code, error)
	ListByProduct(ctx context.Context, productID string, status Status, p pagination.Params) ([]Code, int64, error)
	CountsByProduct(ctx context.Context, productID string) (map[Status]int, error)
	// CountsByAllProducts returns per-product status→count maps for every product
	// in a single aggregation over the codes collection (productId → status →
	// count). Backs the bulk inventory rollup.
	CountsByAllProducts(ctx context.Context) (map[string]map[Status]int, error)
	// AllThresholds returns every configured low-stock threshold keyed by product
	// id, in a single query. Products without a row are simply absent (callers
	// default them).
	AllThresholds(ctx context.Context) (map[string]int, error)
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
	// MarkExpired atomically transitions one available code (matched by product +
	// value) to expired. Returns ErrNotFound when no matching available code exists.
	MarkExpired(ctx context.Context, productID, code string) (*Code, error)
	// FindByOrder returns the delivered codes claimed under an order (for resend).
	FindByOrder(ctx context.Context, orderID string) ([]Code, error)
	// RecordBatch persists an upload-batch record (best-effort history).
	RecordBatch(ctx context.Context, b UploadBatch) error
	// ListUploadHistory returns the most recent upload batches (newest first),
	// each enriched with its product title.
	ListUploadHistory(ctx context.Context, limit int) ([]UploadBatchView, error)
}

// ProductMeta is the minimal product info the inventory views need.
type ProductMeta struct {
	ID       string
	Title    string
	Category string
	// Cost is the admin-set unit cost (Product.Pricing.Cost). Nil = no cost set —
	// the inventory view renders "—" (distinct from a genuine 0).
	Cost *float64
}

// MongoRepository is the MongoDB-backed Repository.
type MongoRepository struct {
	codes      *mongo.Collection
	products   *mongo.Collection
	thresholds *mongo.Collection
	uploads    *mongo.Collection
}

// NewMongoRepository builds a MongoRepository over the codes/products/thresholds
// and upload_batches collections.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		codes:      db.Collection("codes"),
		products:   db.Collection("products"),
		thresholds: db.Collection("inventory_thresholds"),
		uploads:    db.Collection("upload_batches"),
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
		{Keys: bson.D{{Key: "orderId", Value: 1}}}, // resend: find an order's delivered codes
	})
	if err != nil {
		return err
	}
	_, err = db.Collection("inventory_thresholds").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "productId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}
	_, err = db.Collection("upload_batches").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "createdAt", Value: -1}},
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

// InsertOne inserts a single available code for the product. The (productId,
// code) unique index is the dedup guard: a collision surfaces as a Mongo
// duplicate-key error, mapped to ErrDuplicateCode.
func (r *MongoRepository) InsertOne(ctx context.Context, productID string, item UploadItem, batch string) (*Code, error) {
	c := Code{
		ID:        bson.NewObjectID(),
		ProductID: productID,
		Code:      item.Code,
		Pin:       item.Pin,
		Status:    StatusAvailable,
		Batch:     batch,
		CreatedAt: time.Now().UTC(),
	}
	if _, err := r.codes.InsertOne(ctx, c); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return &c, nil
}

// FindByID returns one code by ObjectID scoped to the product.
func (r *MongoRepository) FindByID(ctx context.Context, productID, codeID string) (*Code, error) {
	oid, err := bson.ObjectIDFromHex(codeID)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}
	var c Code
	err = r.codes.FindOne(ctx, bson.D{{Key: "_id", Value: oid}, {Key: "productId", Value: productID}}).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// UpdateAvailableCode edits the value/pin of an available code (matched by id +
// product). The single FindOneAndUpdate guards against editing a code that was
// just claimed. A value collision surfaces as a duplicate-key error mapped to
// ErrDuplicateCode; no matching available code returns ErrNotFound.
func (r *MongoRepository) UpdateAvailableCode(ctx context.Context, productID, codeID, newCode, newPin string) (*Code, error) {
	oid, err := bson.ObjectIDFromHex(codeID)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}
	set := bson.D{{Key: "code", Value: newCode}}
	update := bson.D{{Key: "$set", Value: set}}
	if newPin == "" {
		update = append(update, bson.E{Key: "$unset", Value: bson.D{{Key: "pin", Value: ""}}})
	} else {
		set = append(set, bson.E{Key: "pin", Value: newPin})
		update = bson.D{{Key: "$set", Value: set}}
	}
	var c Code
	err = r.codes.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: oid}, {Key: "productId", Value: productID}, {Key: "status", Value: StatusAvailable}},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&c)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicateCode
		}
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// DeleteAvailableCode removes a code that is not delivered (available or
// expired) — a delivered code is tied to an order's fulfillment and is never
// deleted. Returns the deleted document, or ErrNotFound when none matches.
func (r *MongoRepository) DeleteAvailableCode(ctx context.Context, productID, codeID string) (*Code, error) {
	oid, err := bson.ObjectIDFromHex(codeID)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}
	var c Code
	err = r.codes.FindOneAndDelete(ctx,
		bson.D{{Key: "_id", Value: oid}, {Key: "productId", Value: productID}, {Key: "status", Value: bson.D{{Key: "$ne", Value: StatusDelivered}}}},
	).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
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

// CountsByAllProducts groups the entire codes collection by (productId, status)
// in one aggregation, returning a nested productId → status → count map. This
// replaces per-product count queries in the bulk inventory rollup; the
// (productId, status) index covers the group.
func (r *MongoRepository) CountsByAllProducts(ctx context.Context) (map[string]map[Status]int, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "productId", Value: "$productId"},
				{Key: "status", Value: "$status"},
			}},
			{Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	cur, err := r.codes.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		ID struct {
			ProductID string `bson:"productId"`
			Status    Status `bson:"status"`
		} `bson:"_id"`
		N int `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]map[Status]int, len(rows))
	for _, row := range rows {
		m := out[row.ID.ProductID]
		if m == nil {
			m = map[Status]int{}
			out[row.ID.ProductID] = m
		}
		m[row.ID.Status] = row.N
	}
	return out, nil
}

// AllThresholds loads every per-product threshold in one Find, keyed by product
// id. Products without a stored threshold are absent (callers default them).
func (r *MongoRepository) AllThresholds(ctx context.Context) (map[string]int, error) {
	cur, err := r.thresholds.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		ProductID string `bson:"productId"`
		Threshold int    `bson:"threshold"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.ProductID] = row.Threshold
	}
	return out, nil
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
	// Pricing carries only the admin-set unit cost for the inventory value columns.
	// A product with no pricing subdoc decodes to a zero-value struct (Cost == nil).
	Pricing struct {
		Cost *float64 `bson:"cost"`
	} `bson:"pricing"`
}

// CodeProducts returns metadata for every inventory-mode product. The query is
// backfill-tolerant: it matches products explicitly keyed to inventory mode, and
// (for products that predate fulfillmentMode) falls back to the legacy
// fulfillmentType=="code" — so the admin inventory view is never blank before
// `go run ./cmd/seed` backfills the mode.
func (r *MongoRepository) CodeProducts(ctx context.Context) ([]ProductMeta, error) {
	cur, err := r.products.Find(ctx, bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "fulfillmentMode", Value: "inventory"}},
		bson.D{
			{Key: "fulfillmentMode", Value: bson.D{{Key: "$exists", Value: false}}},
			{Key: "fulfillmentType", Value: "code"},
		},
		// Bridge recharge_line products consume one scratch-card PIN per order from
		// this same code pool, so they need a code inventory too. bridge.method is
		// unique to bridge products (Bridge is nil otherwise; disabling it clears the
		// method), so this branch never pulls in transfer_credit or non-bridge products.
		bson.D{{Key: "bridge.method", Value: "recharge_line"}},
	}}})
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
		out[i] = ProductMeta{ID: d.ID.Hex(), Title: d.Title.En, Category: d.Category, Cost: d.Pricing.Cost}
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
	return &ProductMeta{ID: d.ID.Hex(), Title: d.Title.En, Category: d.Category, Cost: d.Pricing.Cost}, nil
}

// RecordBatch persists an upload-batch history record, stamping id/createdAt.
func (r *MongoRepository) RecordBatch(ctx context.Context, b UploadBatch) error {
	if b.ID.IsZero() {
		b.ID = bson.NewObjectID()
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now().UTC()
	}
	_, err := r.uploads.InsertOne(ctx, b)
	return err
}

// ListUploadHistory returns the most recent upload batches (newest first), each
// enriched with its product title (resolved in a single products query).
func (r *MongoRepository) ListUploadHistory(ctx context.Context, limit int) ([]UploadBatchView, error) {
	if limit <= 0 {
		limit = 20
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(int64(limit))
	cur, err := r.uploads.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var batches []UploadBatch
	if err := cur.All(ctx, &batches); err != nil {
		return nil, err
	}

	// Resolve product titles for the batches in one query.
	oids := make([]bson.ObjectID, 0, len(batches))
	for _, b := range batches {
		if oid, err := bson.ObjectIDFromHex(b.ProductID); err == nil {
			oids = append(oids, oid)
		}
	}
	titles := make(map[string]string, len(oids))
	if len(oids) > 0 {
		pc, err := r.products.Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: oids}}}})
		if err == nil {
			var docs []productDoc
			if err := pc.All(ctx, &docs); err == nil {
				for _, d := range docs {
					titles[d.ID.Hex()] = d.Title.En
				}
			}
		}
	}

	out := make([]UploadBatchView, len(batches))
	for i, b := range batches {
		out[i] = UploadBatchView{UploadBatch: b, ProductTitle: titles[b.ProductID]}
	}
	return out, nil
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

// MarkExpired atomically transitions one available code (matched by product +
// value) to expired. The single FindOneAndUpdate guards against expiring a code
// that was just claimed. Returns ErrNotFound when no matching available code exists.
func (r *MongoRepository) MarkExpired(ctx context.Context, productID, code string) (*Code, error) {
	var c Code
	err := r.codes.FindOneAndUpdate(ctx,
		bson.D{{Key: "productId", Value: productID}, {Key: "code", Value: code}, {Key: "status", Value: StatusAvailable}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: StatusExpired}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// FindByOrder returns the delivered codes claimed under orderID (for resend).
func (r *MongoRepository) FindByOrder(ctx context.Context, orderID string) ([]Code, error) {
	cur, err := r.codes.Find(ctx, bson.D{{Key: "orderId", Value: orderID}, {Key: "status", Value: StatusDelivered}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []Code
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
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
