package offer

import (
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// DiscountType identifies how an offer reduces a product's price.
type DiscountType string

const (
	DiscountPercent DiscountType = "percent" // value% off
	DiscountFixed   DiscountType = "fixed"   // value off (absolute amount)
)

// Offer is a time-boxed sale price on a catalog product. The discount applies to
// every variant of the product, so the storefront shows the variant price struck
// through against the discounted price. Resolution when several offers target one
// product is by SortOrder (lowest first); admins control priority explicitly.
type Offer struct {
	ID            bson.ObjectID `bson:"_id,omitempty"      json:"id"`
	ProductID     bson.ObjectID `bson:"productId"          json:"productId"`
	DiscountType  DiscountType  `bson:"discountType"       json:"discountType"`
	DiscountValue float64       `bson:"discountValue"      json:"discountValue"`
	StartsAt      *time.Time    `bson:"startsAt,omitempty" json:"startsAt,omitempty"`
	EndsAt        *time.Time    `bson:"endsAt,omitempty"   json:"endsAt,omitempty"`
	Active        bool          `bson:"active"             json:"active"`
	SortOrder     int           `bson:"sortOrder"          json:"sortOrder"`
	CreatedAt     time.Time     `bson:"createdAt"          json:"createdAt"`
	UpdatedAt     time.Time     `bson:"updatedAt"          json:"updatedAt"`
}

// OfferPriceFor applies an offer's discount to a unit price, clamped to
// [0, original]. A nil offer or non-positive price returns the price unchanged.
func OfferPriceFor(o *Offer, original float64) float64 {
	if o == nil || original <= 0 {
		return original
	}
	var d float64
	switch o.DiscountType {
	case DiscountPercent:
		d = original * o.DiscountValue / 100
	case DiscountFixed:
		d = o.DiscountValue
	default:
		return original
	}
	if d < 0 {
		d = 0
	}
	if d > original {
		d = original
	}
	return original - d
}

// IsLive reports whether the offer is active and within its [StartsAt, EndsAt]
// window at now (open-ended on either side when the bound is nil).
func IsLive(o *Offer, now time.Time) bool {
	if o == nil || !o.Active {
		return false
	}
	if o.StartsAt != nil && now.Before(*o.StartsAt) {
		return false
	}
	if o.EndsAt != nil && now.After(*o.EndsAt) {
		return false
	}
	return true
}

// productSummary is the slice of a product the offer views surface (admin list +
// public storefront): enough to render a card and the was/now prices.
type productSummary struct {
	ID        string             `json:"id"`
	Title     product.I18nString `json:"title"`
	Images    []string           `json:"images"`
	Category  string             `json:"category"`
	FromPrice float64            `json:"fromPrice"`
	InStock   bool               `json:"inStock"`
}

// fromPrice returns a product's lowest list price (0 when it has no variants);
// a per-unit product quotes its minimum amount (see product.FromPrice).
func fromPrice(p *product.Product) float64 {
	return product.FromPrice(p)
}

// AppliesTo reports whether o's discount may apply to p: a fixed-amount offer
// never applies to a per-unit product (it would be taken off every block).
// Checkout (order.PlaceOrder), the catalog enrichment and the offer views
// share this rule.
func AppliesTo(o *Offer, p *product.Product) bool {
	return o != nil && p.AcceptsDiscount(string(o.DiscountType))
}

// offerFromPrice is p's "from" price with o applied: the discount is taken off
// each variant's (block) price before a per-unit product is rescaled to its
// minimum amount — the same order checkout prices a line in. An offer that
// does not apply leaves the original from price.
func offerFromPrice(o *Offer, p *product.Product) float64 {
	if !AppliesTo(o, p) {
		return fromPrice(p)
	}
	return product.FromPriceWith(p, func(v float64) float64 { return OfferPriceFor(o, v) })
}

// summarize projects a product into the offer view shape.
func summarize(p *product.Product) productSummary {
	return productSummary{
		ID:        p.ID.Hex(),
		Title:     p.Title,
		Images:    p.Images,
		Category:  p.Category,
		FromPrice: fromPrice(p),
		InStock:   p.Available,
	}
}
