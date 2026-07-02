package code

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Handler exposes inventory/code operations over HTTP.
type Handler struct {
	svc Service
}

// NewHandler constructs a Handler.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Inventory handles GET /api/admin/inventory.
func (h *Handler) Inventory(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.Inventory(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, stats)
}

// Upload handles POST /api/admin/products/{id}/codes.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")

	var in UploadInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if len(in.Codes) == 0 {
		response.BadRequest(w, "codes are required")
		return
	}

	// Stamp the uploader from the verified JWT (server-set, never trusted from the
	// body). Falls back to phone/user-id for phone-only admins; empty only in the
	// dev AdminOnly bypass → shown as "—" in the UI.
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		in.UploadedBy = auth.ActorLabel(claims)
	}

	result, err := h.svc.Upload(r.Context(), productID, in)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.JSON(w, http.StatusCreated, response.Response{Success: true, Data: result})
}

// ListCodes handles GET /api/admin/products/{id}/codes.
func (h *Handler) ListCodes(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	status := Status(r.URL.Query().Get("status"))
	p := pagination.ParseParams(r)

	codes, total, err := h.svc.ListCodes(r.Context(), productID, status, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OKWithMeta(w, codes, pagination.CalcMeta(p, total))
}

// Lookup handles GET /api/admin/codes/{code}.
func (h *Handler) Lookup(w http.ResponseWriter, r *http.Request) {
	value := chi.URLParam(r, "code")

	audit, err := h.svc.Lookup(r.Context(), value)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, audit)
}

// SetThreshold handles PUT /api/admin/products/{id}/stock-threshold.
func (h *Handler) SetThreshold(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")

	var in SetThresholdInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if in.Threshold < 0 {
		response.BadRequest(w, "threshold must be non-negative")
		return
	}

	stats, err := h.svc.SetThreshold(r.Context(), productID, in.Threshold)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, stats)
}
