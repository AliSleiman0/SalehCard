package user

import (
	"errors"
	"net/http"
)

// Handler exposes user domain operations over HTTP.
type Handler struct {
	service Service
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register handles POST /auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// GetProfile handles GET /users/me.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// UpdateProfile handles PATCH /users/me.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}
