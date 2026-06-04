package reseller

import (
	"errors"
	"net/http"
)

// Handler exposes reseller domain operations over HTTP.
type Handler struct {
	service Service
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetTier handles GET /reseller/tier.
func (h *Handler) GetTier(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// UpdateTier handles PATCH /reseller/tier.
func (h *Handler) UpdateTier(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}
