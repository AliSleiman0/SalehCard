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
	FindByID(ctx context.Context, id bson.ObjectID) (*Category, error)
	Insert(ctx context.Context, c *Category) error
	UpdateFields(ctx context.Context, id bson.ObjectID, set bson.D) (*Category, error)
	Delete(ctx context.Context, id bson.ObjectID) error
	HasChildren(ctx context.Context, id bson.ObjectID) (bool, error)
	// FindSubtree returns the node id plus every descendant (matched via the
	// ancestors chain), used both to expand a product filter and to rewrite a
	// moved subtree.
	FindSubtree(ctx context.Context, id bson.ObjectID) ([]Category, error)
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

// EnsureIndexes creates the sparse-unique legacyId index plus slug / parent /
// tree lookups. legacyId is sparse so admin-created nodes (no legacyId) don't
// collide on a single null; parentId + ancestors back the tree drill-down and
// descendant expansion.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	col := db.Collection("categories")
	if err := migrateLegacyIDIndex(ctx, col); err != nil {
		return err
	}
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "legacyId", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		{Keys: bson.D{{Key: "slug", Value: 1}}},
		{Keys: bson.D{{Key: "parentLegacyId", Value: 1}}},
		{Keys: bson.D{{Key: "parentId", Value: 1}}},
		{Keys: bson.D{{Key: "ancestors", Value: 1}}},
		{Keys: bson.D{{Key: "parentId", Value: 1}, {Key: "slug", Value: 1}}},
		{Keys: bson.D{{Key: "rootDomain", Value: 1}}},
		{Keys: bson.D{{Key: "depth", Value: 1}}},
	})
	return err
}

// legacyIDIndexName is the driver-generated name of the legacyId index.
const legacyIDIndexName = "legacyId_1"

// migrateLegacyIDIndex drops a pre-tree, NON-sparse unique legacyId index.
//
// Databases created before the catalog tree have legacyId_1 as plain unique.
// Mongo refuses to change an existing index's options, so the CreateMany in
// EnsureIndexes failed with IndexOptionsConflict, silently skipping the newer
// parentId / ancestors indexes too. Worse, the non-sparse index indexes every
// admin-created node (no legacyId) as null, so the second admin-created node
// collided (E11000) and the console showed "an unexpected error occurred".
//
// Only a non-sparse legacyId_1 is dropped, so this is a no-op once migrated
// (no drop/recreate churn on every boot); CreateMany then rebuilds it sparse.
func migrateLegacyIDIndex(ctx context.Context, col *mongo.Collection) error {
	cur, err := col.Indexes().List(ctx)
	if err != nil {
		return err
	}
	var specs []struct {
		Name   string `bson:"name"`
		Sparse bool   `bson:"sparse"`
	}
	if err := cur.All(ctx, &specs); err != nil {
		return err
	}
	for _, s := range specs {
		if s.Name == legacyIDIndexName && !s.Sparse {
			return col.Indexes().DropOne(ctx, legacyIDIndexName)
		}
	}
	return nil
}

// FindAll returns categories matching f, sorted by sortOrder then legacyId.
func (r *MongoRepository) FindAll(ctx context.Context, f CategoryFilter) ([]Category, error) {
	filter := bson.D{}
	// VisibleOnly (public) filters to visible; IncludeHidden (admin) shows all.
	if f.VisibleOnly && !f.IncludeHidden {
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
	if f.ParentID != nil {
		filter = append(filter, bson.E{Key: "parentId", Value: *f.ParentID})
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

// FindByID retrieves a category by its ObjectID, or ErrNotFound.
func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*Category, error) {
	var c Category
	err := r.col.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// Insert persists a new admin-created category document.
func (r *MongoRepository) Insert(ctx context.Context, c *Category) error {
	_, err := r.col.InsertOne(ctx, c)
	return err
}

// UpdateFields applies the given $set document (plus updatedAt) atomically and
// returns the refreshed node, or ErrNotFound.
func (r *MongoRepository) UpdateFields(ctx context.Context, id bson.ObjectID, set bson.D) (*Category, error) {
	set = append(set, bson.E{Key: "updatedAt", Value: time.Now().UTC()})
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated Category
	err := r.col.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
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

// Delete removes a category by id, or ErrNotFound.
func (r *MongoRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// HasChildren reports whether any category has id as its direct parent.
func (r *MongoRepository) HasChildren(ctx context.Context, id bson.ObjectID) (bool, error) {
	n, err := r.col.CountDocuments(ctx, bson.D{{Key: "parentId", Value: id}}, options.Count().SetLimit(1))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// FindSubtree returns the node with the given id plus every descendant (matched
// via the ancestors chain). Sorted by depth so a caller rewriting a moved
// subtree processes parents before children.
func (r *MongoRepository) FindSubtree(ctx context.Context, id bson.ObjectID) ([]Category, error) {
	filter := bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "ancestors", Value: id}},
	}}}
	opts := options.Find().SetSort(bson.D{{Key: "depth", Value: 1}})
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

// DescendantIDs returns the node id plus every descendant id. It satisfies the
// product module's CategoryResolver port so a product list filtered by a
// category returns products in that node AND all its subcategories. An unknown
// or malformed id yields an empty slice (the caller then matches no products).
func (r *MongoRepository) DescendantIDs(ctx context.Context, id string) ([]bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil
	}
	cur, err := r.col.Find(ctx,
		bson.D{{Key: "$or", Value: bson.A{
			bson.D{{Key: "_id", Value: oid}},
			bson.D{{Key: "ancestors", Value: oid}},
		}}},
		options.Find().SetProjection(bson.D{{Key: "_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make([]bson.ObjectID, len(rows))
	for i, row := range rows {
		out[i] = row.ID
	}
	return out, nil
}

// Resolve returns a category's slug + rootDomain, used by the product module to
// denormalize the flat category/rootDomain fields when a product is assigned to
// a taxonomy node (keeping the legacy flat filters working). ErrNotFound on a
// missing/malformed id.
func (r *MongoRepository) Resolve(ctx context.Context, id string) (slug, rootDomain string, err error) {
	oid, oerr := bson.ObjectIDFromHex(id)
	if oerr != nil {
		return "", "", apperrors.ErrNotFound
	}
	c, ferr := r.FindByID(ctx, oid)
	if ferr != nil {
		return "", "", ferr
	}
	return c.Slug, c.RootDomain, nil
}
