package kyc

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Handler exposes the customer-facing KYC operations over HTTP.
type Handler struct {
	service Service
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Submit handles POST /api/v1/kyc — an authenticated customer submits their KYC
// background info, stored pending admin moderation.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	var in SubmitInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	profile, err := h.service.Submit(r.Context(), userID, in)
	if err != nil {
		writeKycError(w, err)
		return
	}
	response.OK(w, profile)
}

// GetMe handles GET /api/v1/kyc/me — the caller's derived KYC profile.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	profile, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		writeKycError(w, err)
		return
	}
	response.OK(w, profile)
}

// writeKycError maps domain errors to HTTP responses (404 missing, 400
// bad-request, 500 otherwise). Shared by the customer and admin handlers.
func writeKycError(w http.ResponseWriter, err error) {
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
