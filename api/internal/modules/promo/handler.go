package promo

import (
	"errors"
	"net/http"
)

// Handler exposes promo domain operations over HTTP.
type Handler struct {
	service Service
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Validate handles POST /promos/validate.
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// Apply handles POST /promos/apply.
func (h *Handler) Apply(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// Create handles POST /promos.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}
