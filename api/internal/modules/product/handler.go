package product

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Handler exposes product domain operations over HTTP.
type Handler struct {
	svc Service
	// rec records destructive admin actions; nil on the public (read-only)
	// registration, set by RegisterAdminRoutes.
	rec audit.Recorder
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/v1/products
// Query params: category (string), rootDomain (string), available (bool),
// page (int), limit (int).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var f ListFilter
	f.Category = q.Get("category")
	f.RootDomain = q.Get("rootDomain")

	if raw := q.Get("available"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(w, "available must be a boolean")
			return
		}
		f.Available = &v
	}

	p := pagination.ParseParams(r)

	products, total, err := h.svc.FindAll(r.Context(), f, p)
	if err != nil {
		response.InternalError(w)
		return
	}

	meta := pagination.CalcMeta(p, total)
	response.OKWithMeta(w, products, meta)
}

// GetByID handles GET /api/v1/products/{id}.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	product, err := h.svc.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, product)
}

// Create handles POST /api/v1/products.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	product, err := h.svc.Create(r.Context(), in)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.JSON(w, http.StatusCreated, response.Response{
		Success: true,
		Data:    product,
	})
}

// Update handles PATCH /api/v1/products/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var in UpdateProductInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	updated, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, updated)
}

// Delete handles DELETE /api/v1/products/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}

	if h.rec != nil {
		h.rec.Record(r.Context(), audit.Entry{
			Action:     audit.ActionProductDelete,
			TargetType: "product",
			TargetID:   id,
		})
	}
	w.WriteHeader(http.StatusNoContent)
}
