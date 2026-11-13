package category

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Handler exposes the read-only category taxonomy over HTTP.
type Handler struct {
	svc Service
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/v1/categories.
// Query params: depth (int, e.g. 0 for root domains), rootDomain (string),
// parentLegacyId (int), withCounts (bool). Only visible categories are returned.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := CategoryFilter{VisibleOnly: true}

	if raw := q.Get("depth"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			response.BadRequest(w, "depth must be an integer")
			return
		}
		f.Depth = &v
	}
	f.RootDomain = q.Get("rootDomain")
	if raw := q.Get("parentLegacyId"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			response.BadRequest(w, "parentLegacyId must be an integer")
			return
		}
		f.ParentLegacyID = &v
	}
	if raw := q.Get("parentId"); raw != "" {
		oid, err := bson.ObjectIDFromHex(raw)
		if err != nil {
			response.BadRequest(w, "parentId must be a valid id")
			return
		}
		f.ParentID = &oid
	}

	withCounts := false
	if raw := q.Get("withCounts"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(w, "withCounts must be a boolean")
			return
		}
		withCounts = v
	}

	items, err := h.svc.List(r.Context(), f, withCounts)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, items)
}

// Get handles GET /api/v1/categories/{id}: one visible node (hasChildren +
// rolled-up productCount) plus its root→parent breadcrumb, for the storefront's
// subcategory drill-down pages. Hidden nodes (or hidden ancestors) → 404.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	view, err := h.svc.Node(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, view)
}
