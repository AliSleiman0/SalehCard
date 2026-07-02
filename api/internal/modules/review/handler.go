package review

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
)

// Handler exposes customer-facing review operations over HTTP.
type Handler struct {
	service Service
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /api/v1/reviews — an authenticated customer submits a
// review, which is stored pending moderation.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}

	var in CreateReviewInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	rv, err := h.service.Create(r.Context(), userID, in)
	if err != nil {
		writeReviewError(w, err)
		return
	}
	response.OK(w, rv)
}

// ListForProduct handles GET /api/v1/products/{id}/reviews — public, returning
// the approved reviews for a product, newest-first and paginated. A malformed
// id yields 404; a product with no approved reviews yields an empty list (200).
func (h *Handler) ListForProduct(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	p := pagination.ParseParams(r)
	rows, total, err := h.service.ListApproved(r.Context(), id, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OKWithMeta(w, rows, pagination.CalcMeta(p, total))
}

// writeReviewError maps domain errors to HTTP responses (404 missing, 400
// bad-request, 500 otherwise). Shared by the customer and admin handlers.
func writeReviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, apperrors.ErrBadRequest):
		msg := err.Error()
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			msg = appErr.Message
		}
		response.BadRequest(w, msg)
	default:
		response.InternalError(w)
	}
}
