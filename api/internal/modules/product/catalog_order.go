package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Manual product order inside a category (client request 2026-11-14).
//
// Scope: the order is per taxonomy node, over the products assigned DIRECTLY
// to that node (categoryId == node). A product belongs to exactly one node, so
// one Position field on the product is enough — it is the product's rank in
// its own node. Products in subcategories are arranged on the subcategory.
// Only a listing of exactly one node honours the ranks (directOnly, or a leaf
// node); listings spanning several nodes (a collection's subtree, a root
// domain, a catalog-wide search) stay plain A→Z (ProductService.FindAll sets
// ListFilter.manualOrder).
//
//	GET /api/admin/products/order?categoryId=…  products.view   — the node's direct products in catalog order
//	PUT /api/admin/products/order               products.manage — {categoryId, productIds} = the full new order
//
// The PUT is guarded: productIds must be exactly the node's current direct
// products (no more, no fewer, no duplicates). Any drift — a product moved in
// or out since the admin loaded the list — is a 409 STALE_ORDER instead of a
// silent partial order. The write stamps Position 1..n with one bulk write
// whose per-row filter also requires the node, so a product moved away
// concurrently is never stamped with a rank in a node it left.

// maxArrangeable bounds how many direct products a node may have and still be
// arranged by hand (one request, one bulk write, one audit row).
const maxArrangeable = 1000

// ReorderInput is the PUT /api/admin/products/order body.
type ReorderInput struct {
	CategoryID string   `json:"categoryId"`
	ProductIDs []string `json:"productIds"`
}

// errTooManyToArrange: the node has more direct products than one arrange
// request may carry (the GET would be truncated, the PUT could never match).
var errTooManyToArrange = badRequest("this category has more than 1000 products; split it into subcategories to arrange them")

// errStaleOrder: the submitted list no longer matches the node's products.
var errStaleOrder = &apperrors.AppError{
	Code:    "STALE_ORDER",
	Message: "The products in this category changed since you opened the list. Reload and arrange again.",
	Err:     apperrors.ErrConflict,
}

// positionStore is the persistence port for manual ordering (satisfied by
// MongoRepository; kept off the Repository interface so its fakes are
// untouched).
type positionStore interface {
	DirectProducts(ctx context.Context, categoryID bson.ObjectID, limit int64) ([]Product, error)
	DirectProductIDs(ctx context.Context, categoryID bson.ObjectID, limit int64) ([]bson.ObjectID, error)
	SetPositions(ctx context.Context, categoryID bson.ObjectID, ids []bson.ObjectID) (matched, modified int64, err error)
}

// orderService implements the arrange-products use case.
type orderService struct {
	store      positionStore
	categories CategoryResolver // optional; nil skips the node-exists check
}

// parseNode validates a category id and (when a resolver is wired) that the
// node exists.
func (o *orderService) parseNode(ctx context.Context, categoryID string) (bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(categoryID)
	if err != nil {
		return bson.ObjectID{}, badRequest("categoryId is required")
	}
	if o.categories != nil {
		if _, _, rerr := o.categories.Resolve(ctx, categoryID); rerr != nil {
			if errors.Is(rerr, apperrors.ErrNotFound) {
				return bson.ObjectID{}, badRequest("category not found")
			}
			return bson.ObjectID{}, rerr // a DB failure is a 500, not "not found"
		}
	}
	return oid, nil
}

// List returns the node's direct products in catalog order (the order
// customers see).
func (o *orderService) List(ctx context.Context, categoryID string) ([]Product, error) {
	oid, err := o.parseNode(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	ps, err := o.store.DirectProducts(ctx, oid, maxArrangeable+1)
	if err != nil {
		return nil, err
	}
	if len(ps) > maxArrangeable {
		return nil, errTooManyToArrange
	}
	sortProductsCatalog(ps)
	return ps, nil
}

// Reorder stamps the node's direct products with Position 1..n in the
// submitted order. Returns how many products changed rank.
func (o *orderService) Reorder(ctx context.Context, in ReorderInput) (int64, error) {
	if len(in.ProductIDs) == 0 {
		return 0, badRequest("productIds is required")
	}
	if len(in.ProductIDs) > maxArrangeable {
		return 0, errTooManyToArrange
	}
	ids := make([]bson.ObjectID, 0, len(in.ProductIDs))
	seen := make(map[bson.ObjectID]bool, len(in.ProductIDs))
	for _, raw := range in.ProductIDs {
		oid, err := bson.ObjectIDFromHex(raw)
		if err != nil {
			return 0, badRequest("productIds contains an invalid id")
		}
		if seen[oid] {
			return 0, badRequest("productIds contains a duplicate")
		}
		seen[oid] = true
		ids = append(ids, oid)
	}
	node, err := o.parseNode(ctx, in.CategoryID)
	if err != nil {
		return 0, err
	}

	// Guard: the submission must be exactly the node's current direct set.
	current, err := o.store.DirectProductIDs(ctx, node, maxArrangeable+1)
	if err != nil {
		return 0, err
	}
	if len(current) > maxArrangeable {
		return 0, errTooManyToArrange
	}
	if len(current) != len(ids) {
		return 0, errStaleOrder
	}
	for _, id := range current {
		if !seen[id] {
			return 0, errStaleOrder
		}
	}

	matched, modified, err := o.store.SetPositions(ctx, node, ids)
	if err != nil {
		return 0, err
	}
	if matched != int64(len(ids)) {
		// A product left the node between the check and the write; the rest
		// are ranked, the leaver was (correctly) not. Ask for a reload.
		return modified, errStaleOrder
	}
	return modified, nil
}

// --- persistence (MongoRepository) -----------------------------------------

// directFilter is THE definition of "the node's arrangeable products", shared
// by the arrange list and the save guard so the two can never disagree (a
// future exclusion — e.g. soft-deleted products — goes here, once).
func directFilter(categoryID bson.ObjectID) bson.D {
	return bson.D{{Key: "categoryId", Value: categoryID}}
}

// DirectProducts loads the products assigned directly to a node (any status —
// the admin arranges drafts too), capped at limit.
func (r *MongoRepository) DirectProducts(ctx context.Context, categoryID bson.ObjectID, limit int64) ([]Product, error) {
	cursor, err := r.col.Find(ctx, directFilter(categoryID),
		options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var ps []Product
	if err := cursor.All(ctx, &ps); err != nil {
		return nil, err
	}
	for i := range ps {
		ps[i].normalize()
	}
	return ps, nil
}

// DirectProductIDs returns the ids of the products assigned directly to a node.
func (r *MongoRepository) DirectProductIDs(ctx context.Context, categoryID bson.ObjectID, limit int64) ([]bson.ObjectID, error) {
	cursor, err := r.col.Find(ctx, directFilter(categoryID),
		options.Find().SetProjection(bson.D{{Key: "_id", Value: 1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rows []struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make([]bson.ObjectID, len(rows))
	for i, row := range rows {
		out[i] = row.ID
	}
	return out, nil
}

// SetPositions stamps ids[i] with Position i+1 in one unordered bulk write.
// Each row's filter also requires categoryId == node, so a product that left
// the node concurrently is skipped (counted out of matched). updatedAt is not
// bumped: arranging is not a content edit.
func (r *MongoRepository) SetPositions(ctx context.Context, categoryID bson.ObjectID, ids []bson.ObjectID) (int64, int64, error) {
	defer invalidateNameCache() // catalog-order listings must not show the old order
	if len(ids) == 0 {
		return 0, 0, nil
	}
	models := make([]mongo.WriteModel, len(ids))
	for i, id := range ids {
		models[i] = mongo.NewUpdateOneModel().
			SetFilter(bson.D{{Key: "_id", Value: id}, {Key: "categoryId", Value: categoryID}}).
			SetUpdate(bson.D{{Key: "$set", Value: bson.D{{Key: "position", Value: i + 1}}}})
	}
	res, err := r.col.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	if err != nil {
		return 0, 0, err
	}
	return res.MatchedCount, res.ModifiedCount, nil
}

// --- transport ----------------------------------------------------------------

// productOrderListHandler serves GET /api/admin/products/order?categoryId=….
func productOrderListHandler(o *orderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ps, err := o.List(r.Context(), r.URL.Query().Get("categoryId"))
		if err != nil {
			writeProductError(w, err)
			return
		}
		if ps == nil {
			ps = []Product{}
		}
		response.OK(w, ps)
	}
}

// productOrderSaveHandler serves PUT /api/admin/products/order and audits the
// new order (category.product_reorder, the full ordered id list).
func productOrderSaveHandler(o *orderService, rec audit.Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in ReorderInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			response.BadRequest(w, "invalid request body")
			return
		}
		modified, err := o.Reorder(r.Context(), in)
		if err != nil && !(errors.Is(err, apperrors.ErrConflict) && modified > 0) {
			writeProductError(w, err)
			return
		}
		// A partial write (a product left mid-save) still changed live ranks,
		// so it is audited before the 409 goes back.
		if rec != nil {
			rec.Record(r.Context(), audit.Entry{
				Action:     audit.ActionCategoryProductReorder,
				TargetType: "category",
				TargetID:   in.CategoryID,
				Summary:    map[string]any{"count": len(in.ProductIDs), "changed": modified, "productIds": in.ProductIDs},
			})
		}
		if err != nil {
			writeProductError(w, err)
			return
		}
		response.OK(w, map[string]int64{"modified": modified})
	}
}
