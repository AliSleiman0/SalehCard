package review

import (
	"errors"
	"net/http"
)

// Handler exposes review domain operations over HTTP.
type Handler struct {
	service Service
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /reviews.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// ListByProduct handles GET /products/{id}/reviews.
func (h *Handler) ListByProduct(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// Delete handles DELETE /reviews/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}
