package wallet

import (
	"errors"
	"net/http"
)

// Handler exposes wallet domain operations over HTTP.
type Handler struct {
	service Service
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// TopUp handles POST /wallet/topup.
func (h *Handler) TopUp(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// GetBalance handles GET /wallet/balance.
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}

// ListTransactions handles GET /wallet/transactions.
func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("TODO: not implemented").Error(), http.StatusNotImplemented)
}
