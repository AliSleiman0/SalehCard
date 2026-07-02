package notification

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Handler exposes the customer notification inbox over HTTP.
type Handler struct {
	repo Repository
}

// NewHandler constructs a Handler over repo.
func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// List handles GET /api/v1/notifications — the caller's inbox, newest first.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	p := pagination.ParseParams(r)
	rows, total, err := h.repo.ListByUser(r.Context(), userID, p)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OKWithMeta(w, rows, pagination.CalcMeta(p, total))
}

// UnreadCount handles GET /api/v1/notifications/unread-count — the bell badge.
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	count, err := h.repo.CountUnread(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]int64{"count": count})
}

// ReadAll handles POST /api/v1/notifications/read-all — called when the client
// opens the inbox; marks everything read in one atomic update.
func (h *Handler) ReadAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	updated, err := h.repo.MarkAllRead(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]int64{"updated": updated})
}

// deviceInput is the register/unregister body.
type deviceInput struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// RegisterDevice handles POST /api/v1/notifications/devices — registers (or
// reassigns) a push device token to the caller.
func (h *Handler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	var in deviceInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	in.Token = strings.TrimSpace(in.Token)
	if in.Token == "" {
		response.BadRequest(w, "token is required")
		return
	}
	if in.Platform == "" {
		in.Platform = "android"
	}
	if err := h.repo.UpsertToken(r.Context(), userID, in.Token, in.Platform); err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]string{"status": "registered"})
}

// UnregisterDevice handles DELETE /api/v1/notifications/devices — removes the
// caller's device token (idempotent: deleting an absent token still succeeds).
func (h *Handler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	var in deviceInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	in.Token = strings.TrimSpace(in.Token)
	if in.Token == "" {
		response.BadRequest(w, "token is required")
		return
	}
	if err := h.repo.DeleteToken(r.Context(), userID, in.Token); err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]string{"status": "unregistered"})
}
