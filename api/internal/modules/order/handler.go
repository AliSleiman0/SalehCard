package order

import (
	"errors"
	"net/http"
)

// Handler exposes order domain operations over HTTP.
type Handler struct {
	service Service
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// PlaceOrder handles POST /orders.
func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// GetOrder handles GET /orders/{id}.
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// ListOrders handles GET /orders.
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// CancelOrder handles DELETE /orders/{id}.
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}
