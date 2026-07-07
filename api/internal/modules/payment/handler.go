package payment

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Handler exposes the customer payment-intent endpoints.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// IntentView is the client-facing intent shape: amounts as USD floats, micros
// and internal bookkeeping never leak. The provider decides which fields the
// client uses — USDT reads network/address; Whish reads redirectUrl.
type IntentView struct {
	ID          string  `json:"id"`
	Provider    string  `json:"provider"`
	Purpose     Purpose `json:"purpose"`
	OrderID     string  `json:"orderId,omitempty"`
	Network     string  `json:"network,omitempty"`
	Address     string  `json:"address,omitempty"`
	RedirectURL string  `json:"redirectUrl,omitempty"`
	AmountUSD   float64 `json:"amountUsd"`
	ReceivedUSD float64 `json:"receivedUsd"`
	Status      string  `json:"status"`
	TxHash      string  `json:"txHash,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	ExpiresAt   string  `json:"expiresAt"`
	ConfirmedAt string  `json:"confirmedAt,omitempty"`
	Settlement  string  `json:"settlement,omitempty"`
}

// NewIntentView adapts an Intent for clients.
func NewIntentView(in *Intent) IntentView {
	v := IntentView{
		ID:          in.ID.Hex(),
		Provider:    ProviderOf(in),
		Purpose:     in.Purpose,
		Network:     in.Network,
		Address:     in.Address,
		RedirectURL: in.RedirectURL,
		AmountUSD:   MicrosToUSD(in.AmountExpectedMicros),
		ReceivedUSD: MicrosToUSD(in.AmountReceivedMicros),
		Status:      string(in.Status),
		TxHash:      in.TxHash,
		CreatedAt:   in.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		ExpiresAt:   in.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Settlement:  in.Settlement,
	}
	if in.OrderID != nil {
		v.OrderID = in.OrderID.Hex()
	}
	if in.ConfirmedAt != nil {
		v.ConfirmedAt = in.ConfirmedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return v
}

// topUpIntentInput is the body of POST /payments/usdt/topup-intents.
type topUpIntentInput struct {
	Amount float64 `json:"amount"`
}

// CreateTopUpIntent handles POST /api/v1/payments/usdt/topup-intents. The
// optional Idempotency-Key header dedupes retries (same contract as orders).
func (h *Handler) CreateTopUpIntent(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	var in topUpIntentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	intent, err := h.svc.CreateTopUpIntent(r.Context(), userID, in.Amount, r.Header.Get("Idempotency-Key"))
	if err != nil {
		writePaymentError(w, err)
		return
	}
	response.OK(w, NewIntentView(intent))
}

// CreateWhishTopUpIntent handles POST /api/v1/payments/whish/topup-intents.
// The optional Idempotency-Key header dedupes retries.
func (h *Handler) CreateWhishTopUpIntent(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	var in topUpIntentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	intent, err := h.svc.CreateWhishTopUpIntent(r.Context(), userID, in.Amount, r.Header.Get("Idempotency-Key"))
	if err != nil {
		writePaymentError(w, err)
		return
	}
	response.OK(w, NewIntentView(intent))
}

// WhishCallback handles the public GET /api/v1/payments/webhooks/whish/{success,
// failure} that Whish fires server-to-server. It carries no JWT — the HMAC token
// in the query string is the authentication. The outcome (success/failure) is
// ignored in favour of re-polling the gateway (see Service.HandleCallback);
// invalid/unknown callbacks → 400 (no retry), transient errors → 500 (retry).
func (h *Handler) WhishCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	externalID := q.Get("externalId")
	token := q.Get("token")
	if externalID == "" || token == "" {
		response.BadRequest(w, "missing externalId or token")
		return
	}
	if err := h.svc.HandleCallback(r.Context(), externalID, token); err != nil {
		if errors.Is(err, ErrInvalidCallback) {
			response.BadRequest(w, "invalid callback")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]string{"status": "ok"})
}

// GetIntent handles GET /api/v1/payments/intents/{id} — the polling endpoint.
func (h *Handler) GetIntent(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	intent, err := h.svc.GetIntent(r.Context(), userID, id)
	if err != nil {
		writePaymentError(w, err)
		return
	}
	response.OK(w, NewIntentView(intent))
}

// configView is the payload of GET /api/v1/payments/config.
type configView struct {
	USDTEnabled   bool   `json:"usdtEnabled"`
	WhishEnabled  bool   `json:"whishEnabled"`
	Network       string `json:"network"`
	ExpiryMinutes int    `json:"expiryMinutes"`
}

// GetConfig handles GET /api/v1/payments/config — the client feature gate.
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	response.OK(w, configView{
		USDTEnabled:   h.svc.Enabled(),
		WhishEnabled:  h.svc.WhishEnabled(),
		Network:       NetworkTRC20,
		ExpiryMinutes: h.svc.ExpiryMinutes(),
	})
}

func (h *Handler) userID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return bson.ObjectID{}, false
	}
	return id, true
}

// writePaymentError maps domain errors to HTTP responses.
func writePaymentError(w http.ResponseWriter, err error) {
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
