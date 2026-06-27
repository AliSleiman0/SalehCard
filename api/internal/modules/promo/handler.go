package promo

import (
	"encoding/json"
	"net/http"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Handler exposes the customer-facing promo operations over HTTP.
type Handler struct {
	service Service
}

// NewHandler constructs a Handler backed by the given service.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// validateResponse is the result of POST /promos/validate — enough for the
// storefront to preview the discount before placing the order.
type validateResponse struct {
	Valid    bool      `json:"valid"`
	Code     string    `json:"code"`
	Type     PromoType `json:"type"`
	Value    float64   `json:"value"`
	Discount float64   `json:"discount"`
}

// Validate handles POST /api/v1/promos/validate. A redeemable code returns the
// computed discount; an unredeemable one returns a 400 with the reason.
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	var input ValidateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	p, err := h.service.Validate(r.Context(), input)
	if err != nil {
		writePromoError(w, err)
		return
	}
	response.OK(w, validateResponse{
		Valid:    true,
		Code:     p.Code,
		Value:    p.Value,
		Type:     p.Type,
		Discount: DiscountFor(p, input.OrderTotal),
	})
}
