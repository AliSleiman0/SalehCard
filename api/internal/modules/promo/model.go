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
	ExpiresAt *time.Time    `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	Active    bool          `bson:"active"        json:"active"`
	CreatedAt time.Time     `bson:"createdAt"     json:"createdAt"`
}

// ValidateInput carries the data needed to check a promo code against an order.
type ValidateInput struct {
	Code       string  `json:"code"`
	OrderTotal float64 `json:"orderTotal"`
}
