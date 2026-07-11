package offer

import (
	"net/http"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Handler exposes the customer-facing offer listing over HTTP.
type Handler struct {
	repo     Repository
	products product.Service
}

// NewHandler constructs a Handler backed by the given repository and product
// service (used to join the product summary into each offer).
func NewHandler(repo Repository, products product.Service) *Handler {
	return &Handler{repo: repo, products: products}
}

// publicOffer is one live offer as the storefront consumes it: the product
// summary plus the was/now prices to render the strikethrough deal card.
type publicOffer struct {
	ID                string         `json:"id"`
	ProductID         string         `json:"productId"`
	DiscountType      DiscountType   `json:"discountType"`
	DiscountValue     float64        `json:"discountValue"`
	EndsAt            *time.Time     `json:"endsAt,omitempty"`
	Product           productSummary `json:"product"`
	OriginalFromPrice float64        `json:"originalFromPrice"`
	OfferFromPrice    float64        `json:"offerFromPrice"`
}

// List handles GET /api/v1/offers — every live offer joined with its product.
// Offers whose product is missing or unavailable are skipped so the storefront
// never shows a deal it can't fulfil. Resellers get an empty list: sale offers
// don't stack on reseller pricing (order.PlaceOrder ignores them), so
// advertising retail deals a reseller can't buy at those prices would mislead.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok && claims.Role == "reseller" {
		response.OK(w, []publicOffer{})
		return
	}
	offers, err := h.repo.ListLive(r.Context(), time.Now().UTC())
	if err != nil {
		response.InternalError(w)
		return
	}
	out := make([]publicOffer, 0, len(offers))
	for _, o := range offers {
		p, err := h.products.FindByID(r.Context(), o.ProductID.Hex())
		if err != nil || !p.Available {
			continue
		}
		s := summarize(p)
		out = append(out, publicOffer{
			ID:                o.ID.Hex(),
			ProductID:         o.ProductID.Hex(),
			DiscountType:      o.DiscountType,
			DiscountValue:     o.DiscountValue,
			EndsAt:            o.EndsAt,
			Product:           s,
			OriginalFromPrice: s.FromPrice,
			OfferFromPrice:    OfferPriceFor(o, s.FromPrice),
		})
	}
	response.OK(w, out)
}
