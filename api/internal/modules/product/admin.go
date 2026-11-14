package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/reseller"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/blob"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RegisterAdminRoutes mounts the admin product CRUD onto r. The caller is
// responsible for applying the AdminOnly middleware to r (the /api/admin group).
// Destructive actions are recorded via rec. store persists uploaded product
// images (POST /products/images). providers is the env-configured fulfillment
// supplier list served to the editor's supplier dropdown.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder, store blob.Storage, categories CategoryResolver, providers []FulfillmentProviderOption) {
	repo := NewMongoRepository(db)
	_ = EnsureIndexes(context.Background(), db)
	// The reseller repository drops stale per-reseller custom prices when an
	// admin switches a product to per-unit pricing.
	svc := NewProductService(repo, WithCategoryResolver(categories), WithResellerPriceCleaner(reseller.NewMongoRepository(db)))
	h := NewHandler(svc)
	h.rec = rec
	h.store = store

	// Flat registration (not a sub-router) so the code module can mount its own
	// /products/{id}/codes routes on the same /api/admin router without a Mount
	// collision.
	r.Get("/products", h.AdminList)
	r.Get("/products/categories", categoryFacetsHandler(repo))
	r.Get("/products/fulfillment-providers", fulfillmentProvidersHandler(providers))
	r.Post("/products", h.Create)
	r.Post("/products/bulk", h.Bulk)
	r.Post("/products/images", h.UploadImage)
	// Manual product order inside a category (catalog_order.go). Static paths,
	// so chi matches them before /products/{id}.
	ord := &orderService{store: repo, categories: categories}
	r.Get("/products/order", productOrderListHandler(ord))
	r.Put("/products/order", productOrderSaveHandler(ord, h.rec))
	r.Get("/products/{id}", h.GetByID)
	r.Put("/products/{id}", h.Update)
	// DELETE soft-deletes (trash); restore undoes it. Both need products.manage
	// (the domain wrapper in server.go). Permanent erase is super-admin only and
	// only for a product already in the trash.
	r.Delete("/products/{id}", h.Delete)
	r.Post("/products/{id}/restore", h.Restore)
	r.With(auth.RequireSuperAdmin()).Delete("/products/{id}/permanent", h.Purge)
}

// FulfillmentProviderOption is one selectable upstream fulfillment supplier —
// its registry id (what Product.FulfillmentProvider stores) and display name.
type FulfillmentProviderOption struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// fulfillmentProvidersHandler serves GET /api/admin/products/fulfillment-providers
// — the env-configured suppliers the product editor's "Fulfill via supplier API"
// dropdown offers. Static per deploy (a supplier appears only when its token is
// set), so the client may cache it.
func fulfillmentProvidersHandler(providers []FulfillmentProviderOption) http.HandlerFunc {
	if providers == nil {
		providers = []FulfillmentProviderOption{} // marshal as [], never null
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		response.OK(w, providers)
	}
}

// categoryFacetsHandler serves GET /api/admin/products/categories — the distinct
// product category values + counts that populate the admin category dropdowns.
// It reads the repository directly (no Service method) to keep the Service
// interface — and its test fakes — unchanged.
func categoryFacetsHandler(repo *MongoRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		facets, err := repo.CategoryFacets(r.Context())
		if err != nil {
			response.InternalError(w)
			return
		}
		response.OK(w, facets)
	}
}

// AdminList handles GET /api/admin/products with category / fulfillmentType /
// status / search filters and pagination. Unlike the customer list it returns
// products of every availability state.
func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := ListFilter{
		Category:   q.Get("category"),
		CategoryID: q.Get("categoryId"),
		Status:     q.Get("status"),
		Search:     q.Get("search"),
	}
	if ft := q.Get("fulfillmentType"); ft != "" {
		f.FulfillmentType = FulfillmentType(ft)
	}
	if fm := q.Get("fulfillmentMode"); fm != "" {
		f.FulfillmentMode = FulfillmentMode(fm)
	}
	// Same catalog order the customer list uses, so the Categories page can
	// preview what customers see.
	if q.Get("sort") == SortByName {
		f.Sort = SortByName
	}

	p := pagination.ParseParams(r)

	products, total, err := h.svc.FindAll(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OKWithMeta(w, products, pagination.CalcMeta(p, total))
}

// Bulk handles POST /api/admin/products/bulk.
func (h *Handler) Bulk(w http.ResponseWriter, r *http.Request) {
	var in BulkInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	// Permanent erase is super-admin only, mirroring the single-product
	// /permanent route (which is gated by auth.RequireSuperAdmin).
	if in.Action == BulkPurge {
		if claims, ok := auth.ClaimsFromContext(r.Context()); !ok || !claims.IsSuperAdmin() {
			response.Forbidden(w, "super admin required")
			return
		}
	}

	modified, err := h.svc.Bulk(r.Context(), in)
	if err != nil {
		var ae *apperrors.AppError
		switch {
		case errors.As(err, &ae):
			writeProductError(w, err)
		case errors.Is(err, apperrors.ErrBadRequest):
			response.BadRequest(w, "ids (or a filter) and a valid action are required")
		default:
			response.InternalError(w)
		}
		return
	}

	h.auditBulk(r.Context(), in, modified)
	response.OK(w, map[string]int64{"modified": modified})
}

// bulkAuditActions maps the trash-related bulk actions to their audit action.
// One entry is written per bulk call (not per product).
var bulkAuditActions = map[BulkAction]string{
	BulkDelete:  audit.ActionProductSoftDelete,
	BulkRestore: audit.ActionProductRestore,
	BulkPurge:   audit.ActionProductDelete,
}

// auditBulk records a trash/restore/purge bulk call: how many products changed
// and what was targeted (the ids, or the filter for "select all matching").
func (h *Handler) auditBulk(ctx context.Context, in BulkInput, modified int64) {
	action, ok := bulkAuditActions[in.Action]
	if !ok || h.rec == nil {
		return
	}
	summary := map[string]any{"count": modified}
	if in.Filter != nil {
		summary["filter"] = in.Filter
	} else {
		summary["ids"] = in.IDs
	}
	h.rec.Record(ctx, audit.Entry{
		Action:     action,
		TargetType: "product",
		TargetID:   "bulk",
		Summary:    summary,
	})
}
