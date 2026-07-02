package wallet

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Handler exposes wallet domain operations over HTTP. It holds the concrete
// WalletService because the top-up request queue lives outside the shared
// Service interface.
type Handler struct {
	service *WalletService
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(service *WalletService) *Handler {
	return &Handler{service: service}
}

// walletView is the combined balance + history payload returned by GET /wallet.
type walletView struct {
	Balance      float64              `json:"balance"`
	Transactions []*WalletTransaction `json:"transactions"`
}

// GetWallet handles GET /api/v1/wallet, returning the balance and ledger.
func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	balance, err := h.service.GetBalance(r.Context(), id)
	if err != nil {
		writeWalletError(w, err)
		return
	}
	txs, err := h.service.ListTransactions(r.Context(), id)
	if err != nil {
		writeWalletError(w, err)
		return
	}
	response.OK(w, walletView{Balance: balance, Transactions: txs})
}

// TopUp handles POST /api/v1/wallet/topups — files a PENDING top-up request
// (the wallet is credited only when an admin approves; the previous instant
// mock credit is gone).
func (h *Handler) TopUp(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	var in TopUpInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	req, err := h.service.CreateTopUpRequest(r.Context(), id, in)
	if err != nil {
		writeWalletError(w, err)
		return
	}
	response.OK(w, req)
}

// ListTopUps handles GET /api/v1/wallet/topups — the user's request history.
func (h *Handler) ListTopUps(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	reqs, err := h.service.ListTopUpRequests(r.Context(), id)
	if err != nil {
		writeWalletError(w, err)
		return
	}
	response.OK(w, reqs)
}

// writeWalletError maps domain errors to HTTP responses.
func writeWalletError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		switch {
		case errors.Is(appErr, apperrors.ErrBadRequest):
			response.Error(w, http.StatusBadRequest, appErr.Code, appErr.Message)
		case errors.Is(appErr, apperrors.ErrNotFound):
			response.Error(w, http.StatusNotFound, appErr.Code, appErr.Message)
		default:
			response.InternalError(w)
		}
		return
	}
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, apperrors.ErrBadRequest):
		response.BadRequest(w, err.Error())
	default:
		response.InternalError(w)
	}
}
