package category

import (
	"net/http"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/bson"

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
