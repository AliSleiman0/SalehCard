package product

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

// Repository defines the persistence contract for the product domain.
type Repository interface {
	FindAll(ctx context.Context, f ListFilter, p pagination.Params) ([]Product, int64, error)
	FindByID(ctx context.Context, id string) (*Product, error)
	FindByLegacyID(ctx context.Context, legacyID int) (*Product, error)
	Create(ctx context.Context, in CreateProductInput) (*Product, error)
	Update(ctx context.Context, id string, in UpdateProductInput) (*Product, error)
	Upsert(ctx context.Context, in UpsertProductInput) (*Product, error)
	Delete(ctx context.Context, id string) error
	BulkSetAvailable(ctx context.Context, ids []string, available bool) (int64, error)
	BulkSetCategory(ctx context.Context, ids []string, categoryID, slug, rootDomain string) (int64, error)
	BulkDelete(ctx context.Context, ids []string) (int64, error)
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
			// Top-level domain browse (storefront category tiles filter on this).
			Keys: bson.D{{Key: "rootDomain", Value: 1}},
		},
		{
			// Taxonomy-node browse (tree-aware category filter matches $in here).
			Keys: bson.D{{Key: "categoryId", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "available", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		},
		{
			// Sparse so products without a legacyId (admin-created / dev seed)
			// don't collide on a single null; unique so re-import upserts by id.
			Keys:    bson.D{{Key: "legacyId", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
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
	// Tree-aware taxonomy filter: the service expands a categoryId to the node +
	// its descendants (CategoryIDs). Fall back to an exact single-node match when
	// only the raw id is present (e.g. no resolver wired).
	if len(f.CategoryIDs) > 0 {
		filter = append(filter, bson.E{Key: "categoryId", Value: bson.D{{Key: "$in", Value: f.CategoryIDs}}})
	} else if f.CategoryID != "" {
		if oid, err := bson.ObjectIDFromHex(f.CategoryID); err == nil {
			filter = append(filter, bson.E{Key: "categoryId", Value: oid})
		}
	}
	if f.RootDomain != "" {
		filter = append(filter, bson.E{Key: "rootDomain", Value: f.RootDomain})
	}
	if f.Available != nil {
		filter = append(filter, bson.E{Key: "available", Value: *f.Available})
	}
	if f.FulfillmentType != "" {
		filter = append(filter, bson.E{Key: "fulfillmentType", Value: f.FulfillmentType})
	}
	if f.FulfillmentMode != "" {
		filter = append(filter, bson.E{Key: "fulfillmentMode", Value: f.FulfillmentMode})
	}
	switch f.Status {
	case "active":
		filter = append(filter, bson.E{Key: "available", Value: true})
	case "draft":
		filter = append(filter, bson.E{Key: "available", Value: false})
	case "out":
		filter = append(filter, bson.E{Key: "stock", Value: 0})
		// TODO(mode): re-key onto fulfillmentMode==inventory once products are
		// backfilled (code↔inventory is 1:1 under DeriveMode, so this is correct today).
		filter = append(filter, bson.E{Key: "fulfillmentType", Value: FulfillmentCode})
	}
	if f.Search != "" {
		// Escape the user-supplied query so regex metacharacters (a stray "(", "*",
		// etc.) match literally instead of erroring or opening a ReDoS vector.
		rx := bson.Regex{Pattern: regexp.QuoteMeta(f.Search), Options: "i"}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "title.en", Value: rx}},
			bson.D{{Key: "title.ar", Value: rx}},
			bson.D{{Key: "title.tr", Value: rx}},
			bson.D{{Key: "category", Value: rx}},
		}})
	}
	return filter
}

// objectIDs converts hex id strings to ObjectIDs, skipping any that are invalid.
func objectIDs(ids []string) []bson.ObjectID {
	out := make([]bson.ObjectID, 0, len(ids))
	for _, id := range ids {
		if oid, err := bson.ObjectIDFromHex(id); err == nil {
			out = append(out, oid)
		}
	}
	return out
}

// BulkSetAvailable sets the availability flag on every product in ids.
func (r *MongoRepository) BulkSetAvailable(ctx context.Context, ids []string, available bool) (int64, error) {
	oids := objectIDs(ids)
	if len(oids) == 0 {
		return 0, nil
	}
	res, err := r.col.UpdateMany(ctx,
		bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: oids}}}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "available", Value: available},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// BulkSetCategory (re)assigns every product in ids to a taxonomy node, setting
// the denormalized flat category slug + rootDomain alongside the categoryId.
// An empty categoryID clears the assignment (unassigns) by unsetting all three
// fields. The service resolves slug/rootDomain from the node before calling.
func (r *MongoRepository) BulkSetCategory(ctx context.Context, ids []string, categoryID, slug, rootDomain string) (int64, error) {
	oids := objectIDs(ids)
	if len(oids) == 0 {
		return 0, nil
	}
	filter := bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: oids}}}}
	now := time.Now().UTC()

	var update bson.D
	if categoryID == "" {
		update = bson.D{
			{Key: "$set", Value: bson.D{{Key: "updatedAt", Value: now}}},
			{Key: "$unset", Value: bson.D{
				{Key: "categoryId", Value: ""},
				{Key: "category", Value: ""},
				{Key: "rootDomain", Value: ""},
			}},
		}
	} else {
		catOID, err := bson.ObjectIDFromHex(categoryID)
		if err != nil {
			return 0, apperrors.ErrBadRequest
		}
		update = bson.D{{Key: "$set", Value: bson.D{
			{Key: "categoryId", Value: catOID},
			{Key: "category", Value: slug},
			{Key: "rootDomain", Value: rootDomain},
			{Key: "updatedAt", Value: now},
		}}}
	}

	res, err := r.col.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// BulkDelete removes every product in ids.
func (r *MongoRepository) BulkDelete(ctx context.Context, ids []string) (int64, error) {
	oids := objectIDs(ids)
	if len(oids) == 0 {
		return 0, nil
	}
	res, err := r.col.DeleteMany(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: oids}}}})
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
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
		// _id is a tiebreaker so the sort is a total order: bulk-imported products
		// share a createdAt to the millisecond, and createdAt alone is not stable
		// across separate skip/limit queries — without this, paging duplicates and
		// skips rows at page boundaries.
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}})

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var products []Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}
	for i := range products {
		products[i].normalize()
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
	p.normalize()

	return &p, nil
}

// FindByLegacyID retrieves a product by its migration legacyId, or ErrNotFound.
func (r *MongoRepository) FindByLegacyID(ctx context.Context, legacyID int) (*Product, error) {
	var p Product
	err := r.col.FindOne(ctx, bson.D{{Key: "legacyId", Value: legacyID}}).Decode(&p)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	p.normalize()
	return &p, nil
}

// CountByRootDomain returns rootDomain -> number of available products, used by
// the category service to annotate the storefront's root tiles with live counts.
// Products with an empty rootDomain (dev seed / admin-created) are excluded.
func (r *MongoRepository) CountByRootDomain(ctx context.Context) (map[string]int64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "available", Value: true},
			{Key: "rootDomain", Value: bson.D{{Key: "$ne", Value: ""}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$rootDomain"},
			{Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	cursor, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rows []struct {
		ID string `bson:"_id"`
		N  int64  `bson:"n"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.ID] = row.N
	}
	return out, nil
}

// CountByCategory returns categoryId -> number of available products directly
// assigned to that node (products without a categoryId are excluded). The
// category service rolls these up each node's subtree via the ancestors chain.
func (r *MongoRepository) CountByCategory(ctx context.Context) (map[bson.ObjectID]int64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "available", Value: true},
			{Key: "categoryId", Value: bson.D{{Key: "$ne", Value: nil}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$categoryId"},
			{Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	cursor, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rows []struct {
		ID bson.ObjectID `bson:"_id"`
		N  int64         `bson:"n"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[bson.ObjectID]int64, len(rows))
	for _, row := range rows {
		out[row.ID] = row.N
	}
	return out, nil
}

// CountAssignedTo returns how many products are directly assigned to the given
// category node (any availability). Used by the category delete-guard.
func (r *MongoRepository) CountAssignedTo(ctx context.Context, categoryID bson.ObjectID) (int64, error) {
	return r.col.CountDocuments(ctx, bson.D{{Key: "categoryId", Value: categoryID}})
}

// CategoryFacet is one distinct product category and how many products carry it.
type CategoryFacet struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// CategoryFacets returns the distinct, non-empty product `category` values with
// their product counts (alphabetical). These are the exact strings the product
// list filter matches on, so the admin category dropdowns can be populated from
// them without guessing the taxonomy's slug alignment.
func (r *MongoRepository) CategoryFacets(ctx context.Context) ([]CategoryFacet, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "category", Value: bson.D{{Key: "$nin", Value: bson.A{"", nil}}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$category"},
			{Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
	cursor, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rows []struct {
		ID string `bson:"_id"`
		N  int    `bson:"n"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make([]CategoryFacet, len(rows))
	for i, row := range rows {
		out[i] = CategoryFacet{Value: row.ID, Count: row.N}
	}
	return out, nil
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

	// Always persist a fulfillmentMode: honor an explicit value, else derive a
	// behavior-preserving default from the type so the dispatcher never sees an
	// empty mode for admin-created products.
	mode := in.FulfillmentMode
	if mode == "" {
		mode = DeriveMode(in.FulfillmentType)
	}

	// Taxonomy assignment (optional): the service has already resolved and set
	// in.Category (slug) + in.RootDomain from the node.
	var categoryID *bson.ObjectID
	if in.CategoryID != nil {
		if oid, oerr := bson.ObjectIDFromHex(*in.CategoryID); oerr == nil {
			categoryID = &oid
		}
	}

	// Admin-set unit cost (optional): seed a minimal Pricing carrying only Cost.
	// Retail/Margin/Currency/Mode stay importer-owned and absent for admin products.
	var pricing *Pricing
	if in.Cost != nil {
		pricing = &Pricing{Cost: *in.Cost}
	}

	p := Product{
		ID:                  bson.NewObjectID(),
		Title:               in.Title,
		Description:         in.Description,
		Category:            in.Category,
		CategoryID:          categoryID,
		RootDomain:          in.RootDomain,
		Images:              in.Images,
		Thumbnail:           in.Thumbnail,
		Variants:            variants,
		FulfillmentType:     in.FulfillmentType,
		FulfillmentMode:     mode,
		FulfillmentProvider: in.FulfillmentProvider,
		UpstreamProductID:   in.UpstreamProductID,
		InputFields:         in.InputFields,
		Bridge:              in.Bridge,
		MoneyTransfer:       in.MoneyTransfer,
		Pricing:             pricing,
		Stock:               in.Stock,
		Available:           in.Available,
		Ratings:             in.Ratings,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	p.normalize() // persist images as [] (not null) when created without any

	if _, err := r.col.InsertOne(ctx, p); err != nil {
		return nil, err
	}

	return &p, nil
}

// Upsert inserts or updates a product keyed on legacyId (the catalog migration
// loader). It uses an aggregation-pipeline update so re-import can refresh
// catalog-owned fields (including the synthetic variant's price, which the order
// engine charges from) while preserving operational state owned by the app:
//   - the existing variant's _id is kept (order history references it) — only a
//     fresh insert generates one; price/denomination are always refreshed.
//   - createdAt/legacyId/stock/ratings/available are insert-only ($ifNull): a
//     re-import never clobbers admin-managed stock or a hand-toggled
//     availability. available is derived from status only on first insert.
func (r *MongoRepository) Upsert(ctx context.Context, in UpsertProductInput) (*Product, error) {
	set := buildUpsertSet(in, time.Now().UTC(), bson.NewObjectID())

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var updated Product
	err := r.col.FindOneAndUpdate(ctx,
		bson.D{{Key: "legacyId", Value: in.LegacyID}},
		mongo.Pipeline{bson.D{{Key: "$set", Value: set}}},
		opts,
	).Decode(&updated)
	if err != nil {
		return nil, err
	}
	updated.normalize()
	return &updated, nil
}

// buildUpsertSet builds the aggregation-pipeline $set stage for a catalog upsert.
//
// Because a pipeline $set evaluates every value as an EXPRESSION, any plain
// string that begins with "$" (e.g. a product titled "$ 15 Roblox USA") would be
// misread as a field-path reference and resolve to missing — silently blanking
// the field. Plain data values are therefore wrapped in $literal so they store
// verbatim. Only `variants` (a real $cond) and the $ifNull insert-only fields are
// genuine expressions and must stay unwrapped.
func buildUpsertSet(in UpsertProductInput, now time.Time, newVariantID bson.ObjectID) bson.D {
	mode := in.FulfillmentMode
	if mode == "" {
		mode = DeriveMode(in.FulfillmentType)
	}

	label := "Default"
	var price float64
	if len(in.Variants) > 0 {
		label = in.Variants[0].Denomination
		price = in.Variants[0].Price
	}

	lit := func(v any) bson.D { return bson.D{{Key: "$literal", Value: v}} }

	// Single synthetic variant: on re-import keep the existing element (its _id
	// and any resellerPrice) and overlay the refreshed price/denomination; on
	// first insert build a new one. The denomination is $literal-wrapped for the
	// same "$"-prefix reason as the top-level fields.
	variantExpr := bson.D{{Key: "$cond", Value: bson.D{
		{Key: "if", Value: bson.D{{Key: "$gt", Value: bson.A{
			bson.D{{Key: "$size", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$variants", bson.A{}}}}}}, 0,
		}}}},
		{Key: "then", Value: bson.A{bson.D{{Key: "$mergeObjects", Value: bson.A{
			bson.D{{Key: "$arrayElemAt", Value: bson.A{"$variants", 0}}},
			bson.D{{Key: "price", Value: price}, {Key: "denomination", Value: lit(label)}},
		}}}}},
		{Key: "else", Value: bson.A{bson.D{
			{Key: "_id", Value: newVariantID},
			{Key: "denomination", Value: lit(label)},
			{Key: "price", Value: price},
		}}},
	}}}

	ifNull := func(field string, fallback any) bson.D {
		return bson.D{{Key: "$ifNull", Value: bson.A{"$" + field, fallback}}}
	}

	return bson.D{
		// Catalog-owned: refreshed every run. $literal-wrapped so "$"-prefixed
		// values (titles, denominations, …) are stored, not evaluated as paths.
		{Key: "updatedAt", Value: lit(now)},
		{Key: "title", Value: lit(in.Title)},
		{Key: "category", Value: lit(in.Category)},
		{Key: "categorySlug", Value: lit(in.CategorySlug)},
		{Key: "rootDomain", Value: lit(in.RootDomain)},
		{Key: "legacyCategoryId", Value: lit(in.LegacyCategoryID)},
		{Key: "images", Value: lit(in.Images)},
		{Key: "description", Value: lit(in.Description)},
		{Key: "descriptionHadMarkup", Value: lit(in.DescriptionHadMarkup)},
		{Key: "fulfillmentType", Value: lit(in.FulfillmentType)},
		{Key: "fulfillmentMode", Value: lit(mode)},
		{Key: "fulfillmentProvider", Value: lit(in.FulfillmentProvider)},
		// upstreamProductId is deliberately ABSENT here: supplier mappings are
		// admin-owned, and a legacy-catalog re-import must never wipe them
		// (omitting the key leaves the stored value untouched). moneyTransfer is
		// omitted for the same reason: the buy/sell rate board is admin-set (the
		// loader has no rate data), so a re-import leaves any stored rates intact.
		{Key: "fulfillmentConfidence", Value: lit(in.FulfillmentConfidence)},
		{Key: "fulfillmentCancellable", Value: lit(in.FulfillmentCancellable)},
		{Key: "pricing", Value: lit(in.Pricing)},
		{Key: "amountConstraints", Value: lit(in.AmountConstraints)},
		{Key: "inputFields", Value: lit(in.InputFields)},
		{Key: "verification", Value: lit(in.Verification)},
		{Key: "status", Value: lit(in.Status)},
		{Key: "sortOrder", Value: lit(in.SortOrder)},
		{Key: "flags", Value: lit(in.Flags)},
		{Key: "variants", Value: variantExpr},
		// Insert-only (admin/operational): preserved on re-import via $ifNull.
		{Key: "createdAt", Value: ifNull("createdAt", now)},
		{Key: "legacyId", Value: ifNull("legacyId", in.LegacyID)},
		// thumbnail is admin-owned (uploaded via the product editor; the loader
		// never has one). Insert-only so a catalog re-import never wipes an
		// admin-uploaded thumbnail. First insert seeds in.Thumbnail (usually "").
		{Key: "thumbnail", Value: ifNull("thumbnail", in.Thumbnail)},
		{Key: "stock", Value: ifNull("stock", 0)},
		{Key: "ratings", Value: ifNull("ratings", RatingsSummary{})},
		{Key: "available", Value: ifNull("available", in.Available)},
	}
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
	if in.Description != nil {
		set = append(set, bson.E{Key: "description", Value: *in.Description})
	}
	if in.Category != nil {
		set = append(set, bson.E{Key: "category", Value: *in.Category})
	}
	// Taxonomy re-assignment: the service resolved CategoryID and set the derived
	// RootDomain. Persist the node ref + denormalized rootDomain together.
	if in.CategoryID != nil {
		if oid, oerr := bson.ObjectIDFromHex(*in.CategoryID); oerr == nil {
			set = append(set, bson.E{Key: "categoryId", Value: oid})
		}
	}
	if in.RootDomain != nil {
		set = append(set, bson.E{Key: "rootDomain", Value: *in.RootDomain})
	}
	// Admin-set unit cost: dot-path so it creates pricing.{cost} on a product with
	// no pricing yet and preserves importer-owned retail/margin/currency on one
	// that has them (never replacing the whole pricing subdoc).
	if in.Cost != nil {
		set = append(set, bson.E{Key: "pricing.cost", Value: *in.Cost})
	}
	if in.Images != nil {
		set = append(set, bson.E{Key: "images", Value: in.Images})
	}
	if in.Thumbnail != nil {
		set = append(set, bson.E{Key: "thumbnail", Value: *in.Thumbnail})
	}
	if in.Variants != nil {
		// Generate IDs for any variant added through the admin editor (which
		// submits variants without an _id).
		variants := make([]Variant, len(in.Variants))
		for i, v := range in.Variants {
			if v.ID.IsZero() {
				v.ID = bson.NewObjectID()
			}
			variants[i] = v
		}
		set = append(set, bson.E{Key: "variants", Value: variants})
	}
	if in.FulfillmentType != nil {
		set = append(set, bson.E{Key: "fulfillmentType", Value: *in.FulfillmentType})
	}
	if in.FulfillmentMode != nil {
		set = append(set, bson.E{Key: "fulfillmentMode", Value: *in.FulfillmentMode})
	}
	if in.FulfillmentProvider != nil {
		// 0 is the clear sentinel (never a valid registry id) — mirrors the
		// verification/bridge "empty value clears" convention below.
		if *in.FulfillmentProvider == 0 {
			set = append(set, bson.E{Key: "fulfillmentProvider", Value: nil})
		} else {
			set = append(set, bson.E{Key: "fulfillmentProvider", Value: *in.FulfillmentProvider})
		}
	}
	if in.UpstreamProductID != nil {
		// "" clears the supplier mapping.
		if *in.UpstreamProductID == "" {
			set = append(set, bson.E{Key: "upstreamProductId", Value: nil})
		} else {
			set = append(set, bson.E{Key: "upstreamProductId", Value: *in.UpstreamProductID})
		}
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
	if in.InputFields != nil {
		set = append(set, bson.E{Key: "inputFields", Value: in.InputFields})
	}
	if in.Verification != nil {
		// An empty App means "disable verification" → clear the stored config.
		if in.Verification.App == "" {
			set = append(set, bson.E{Key: "verification", Value: nil})
		} else {
			set = append(set, bson.E{Key: "verification", Value: in.Verification})
		}
	}
	if in.Bridge != nil {
		// An empty Provider means "no longer bridge-fulfilled" → clear the spec.
		if in.Bridge.Provider == "" {
			set = append(set, bson.E{Key: "bridge", Value: nil})
		} else {
			set = append(set, bson.E{Key: "bridge", Value: in.Bridge})
		}
	}
	if in.MoneyTransfer != nil {
		// Both currencies empty and no rates means "remove the rate board" → clear.
		if in.MoneyTransfer.BaseCurrency == "" && in.MoneyTransfer.QuoteCurrency == "" &&
			in.MoneyTransfer.BuyRate == 0 && in.MoneyTransfer.SellRate == 0 {
			set = append(set, bson.E{Key: "moneyTransfer", Value: nil})
		} else {
			set = append(set, bson.E{Key: "moneyTransfer", Value: in.MoneyTransfer})
		}
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
	updated.normalize()

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
