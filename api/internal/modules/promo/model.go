package promo

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// PromoType identifies the discount mechanism of a promo code.
type PromoType string

const (
	PromoTypePercent  PromoType = "percent"
	PromoTypeFixed    PromoType = "fixed"
	PromoTypeCashback PromoType = "cashback"
)

// PromoCode represents a discount code that can be applied at checkout.
type PromoCode struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Code      string        `bson:"code"          json:"code"`
	Type      PromoType     `bson:"type"          json:"type"`
	Value     float64       `bson:"value"         json:"value"`
	MinOrder  float64       `bson:"minOrder"      json:"minOrder"`
	MaxUses   int           `bson:"maxUses"       json:"maxUses"`
	Uses      int           `bson:"uses"          json:"uses"`
	StartsAt  *time.Time    `bson:"startsAt,omitempty"  json:"startsAt,omitempty"`
	ExpiresAt *time.Time    `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	Active    bool          `bson:"active"        json:"active"`
	CreatedAt time.Time     `bson:"createdAt"     json:"createdAt"`
}

// ValidateInput carries the data needed to check a promo code against an order.
type ValidateInput struct {
	Code       string  `json:"code"`
	OrderTotal float64 `json:"orderTotal"`
}

// DiscountFor computes the discount a valid promo applies to an order total.
// Percent codes take value% off; fixed codes take value off (never more than the
// total); cashback codes are a post-purchase wallet reward, not a checkout
// discount, so they return 0 here (crediting is a deferred follow-up). The result
// is always within [0, orderTotal].
func DiscountFor(p *PromoCode, orderTotal float64) float64 {
	if p == nil || orderTotal <= 0 {
		return 0
	}
	var d float64
	switch p.Type {
	case PromoTypePercent:
		d = orderTotal * p.Value / 100
	case PromoTypeFixed:
		d = p.Value
	default: // cashback — not a checkout discount
		return 0
	}
	if d < 0 {
		return 0
	}
	if d > orderTotal {
		return orderTotal
	}
	return d
}
