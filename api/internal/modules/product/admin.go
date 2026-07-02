package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// RegisterAdminRoutes mounts the admin product CRUD onto r. The caller is
// responsible for applying the AdminOnly middleware to r (the /api/admin group).
// Destructive actions are recorded via rec.
func RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder) {
	repo := NewMongoRepository(db)
	_ = EnsureIndexes(context.Background(), db)
	svc := NewProductService(repo)
	h := NewHandler(svc)
	h.rec = rec

	// Flat registration (not a sub-router) so the code module can mount its own
	// /products/{id}/codes routes on the same /api/admin router without a Mount
	// collision.
	r.Get("/products", h.AdminList)
	r.Get("/products/categories", categoryFacetsHandler(repo))
	r.Post("/products", h.Create)
	r.Post("/products/bulk", h.Bulk)
	r.Get("/products/{id}", h.GetByID)
	r.Put("/products/{id}", h.Update)
	r.Delete("/products/{id}", h.Delete)
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
		Category: q.Get("category"),
		Status:   q.Get("status"),
		Search:   q.Get("search"),
	}
	if ft := q.Get("fulfillmentType"); ft != "" {
		f.FulfillmentType = FulfillmentType(ft)
	}
	if fm := q.Get("fulfillmentMode"); fm != "" {
		f.FulfillmentMode = FulfillmentMode(fm)
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

	modified, err := h.svc.Bulk(r.Context(), in)
	if err != nil {
		if errors.Is(err, apperrors.ErrBadRequest) {
			response.BadRequest(w, "ids and a valid action are required")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]int64{"modified": modified})
}
