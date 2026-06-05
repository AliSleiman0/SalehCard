package user

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// refreshCookieName is the name of the httpOnly refresh-token cookie.
const refreshCookieName = "refresh_token"

// refreshCookiePath scopes the cookie to the auth endpoints that consume it.
const refreshCookiePath = "/api/v1/auth"

// Handler exposes user domain operations over HTTP.
type Handler struct {
	service Service
	secure  bool
}

// NewHandler constructs a Handler. secure marks the refresh cookie Secure
// (enabled outside development).
func NewHandler(service Service, secure bool) *Handler {
	return &Handler{service: service, secure: secure}
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	res, err := h.service.Register(r.Context(), in)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeAuth(w, res)
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in LoginInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	res, err := h.service.Login(r.Context(), in)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeAuth(w, res)
}

// Refresh handles POST /api/v1/auth/refresh, rotating the refresh cookie.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		response.Unauthorized(w, "missing refresh token")
		return
	}
	res, err := h.service.Refresh(r.Context(), cookie.Value)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeAuth(w, res)
}

// Logout handles POST /api/v1/auth/logout, revoking and clearing the cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		_ = h.service.Logout(r.Context(), cookie.Value)
	}
	h.clearRefreshCookie(w)
	response.OK(w, map[string]bool{"success": true})
}

// GetProfile handles GET /api/v1/users/me.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	user, err := h.service.GetProfile(r.Context(), id)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	response.OK(w, user)
}

// UpdateProfile handles PATCH /api/v1/users/me.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	var in UpdateProfileInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	user, err := h.service.UpdateProfile(r.Context(), id, in)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	response.OK(w, user)
}

// writeAuth sets the rotated refresh cookie and writes the AuthResponse body.
func (h *Handler) writeAuth(w http.ResponseWriter, res *AuthResult) {
	h.setRefreshCookie(w, res.RefreshToken)
	response.OK(w, AuthResponse{AccessToken: res.AccessToken, User: res.User})
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// writeAuthError maps domain errors to HTTP responses.
func writeAuthError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		// AppError wraps a sentinel; classify by the wrapped cause.
		switch {
		case errors.Is(appErr, apperrors.ErrBadRequest):
			response.Error(w, http.StatusBadRequest, appErr.Code, appErr.Message)
		case errors.Is(appErr, apperrors.ErrConflict):
			response.Error(w, http.StatusConflict, appErr.Code, appErr.Message)
		case errors.Is(appErr, apperrors.ErrUnauthorized):
			response.Error(w, http.StatusUnauthorized, appErr.Code, appErr.Message)
		default:
			response.InternalError(w)
		}
		return
	}

	switch {
	case errors.Is(err, apperrors.ErrConflict):
		response.Error(w, http.StatusConflict, "EMAIL_TAKEN", "an account with this email already exists")
	case errors.Is(err, apperrors.ErrUnauthorized):
		response.Unauthorized(w, "invalid credentials")
	case errors.Is(err, apperrors.ErrBadRequest):
		response.BadRequest(w, err.Error())
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	default:
		response.InternalError(w)
	}
}
