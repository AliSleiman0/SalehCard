package order

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/internal/modules/payment"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Handler exposes order domain operations over HTTP.
type Handler struct {
	service Service
	usdt    usdtIntents // nil when on-chain USDT checkout is disabled
}

// NewHandler constructs a Handler backed by the given service. usdt (optional)
// lets a placed USDT order carry its deposit intent back in the response.
func NewHandler(service Service, usdt usdtIntents) *Handler {
	return &Handler{service: service, usdt: usdt}
}

// placeOrderResponse is the POST /orders body. It embeds the order (its fields
// promote to the top level) and, for a USDT order awaiting payment, the deposit
// intent the client shows on its waiting-for-payment screen.
type placeOrderResponse struct {
	*Order
	PaymentIntent *payment.IntentView `json:"paymentIntent,omitempty"`
}

// PlaceOrder handles POST /api/v1/orders. The price is derived server-side; the
// optional Idempotency-Key header dedupes retries.
func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}

	var in PlaceOrderInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	isReseller := claims.Role == "reseller"
	key := r.Header.Get("Idempotency-Key")

	order, err := h.service.PlaceOrder(r.Context(), userID, isReseller, key, in)
	if err != nil {
		writeOrderError(w, err)
		return
	}

	// A USDT order returns still-pending with an on-chain deposit intent; attach
	// it so the client can render its waiting-for-payment screen. This also
	// covers an Idempotency-Key replay — the live intent is re-read for the
	// order, recovering the deposit details after a lost response.
	if order.PaymentMethod == PaymentMethodUSDT && order.Status == OrderStatusPending && h.usdt != nil {
		if intent, ierr := h.usdt.GetActiveOrderIntent(r.Context(), order.ID); ierr == nil {
			view := payment.NewIntentView(intent)
			response.OK(w, placeOrderResponse{Order: order, PaymentIntent: &view})
			return
		}
	}
	response.OK(w, order)
}

// ListOrders handles GET /api/v1/orders.
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	orders, err := h.service.ListOrders(r.Context(), userID)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	response.OK(w, orders)
}

// GetOrder handles GET /api/v1/orders/{id}.
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	order, err := h.service.GetOrder(r.Context(), userID, id)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	response.OK(w, order)
}

// writeOrderError maps domain errors to HTTP responses.
func writeOrderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrKYCRequired):
		response.Error(w, http.StatusForbidden, "KYC_REQUIRED", "identity verification is required before purchasing")
	case errors.Is(err, code.ErrOutOfStock):
		response.Error(w, http.StatusConflict, "OUT_OF_STOCK", "one or more items are out of stock")
	case errors.Is(err, wallet.ErrInsufficientFunds):
		response.Error(w, http.StatusPaymentRequired, "INSUFFICIENT_FUNDS", "wallet balance is insufficient")
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, apperrors.ErrBadRequest):
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			response.Error(w, http.StatusBadRequest, appErr.Code, appErr.Message)
			return
		}
		response.BadRequest(w, err.Error())
	default:
		response.InternalError(w)
	}
}
