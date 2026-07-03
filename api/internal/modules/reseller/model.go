package reseller

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ResellerTier is a shared tier definition (Bronze/Silver/Gold). A reseller is a
// user.User with Role == "reseller" whose resellerTier field names the tier they
// belong to; the per-reseller sub-balance lives on the user wallet, not here.
type ResellerTier struct {
	ID            bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name          string        `bson:"name"          json:"name"`
	MarginPercent float64       `bson:"marginPercent" json:"marginPercent"`
	CreatedAt     time.Time     `bson:"createdAt"     json:"createdAt"`
}

// ResellerPrice is a per-reseller price override for one product variant — the
// most specific pricing layer (beats retail, the tier margin, and the per-variant
// global reseller price). One document per (reseller, variant).
type ResellerPrice struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    bson.ObjectID `bson:"userId"        json:"userId"`
	ProductID string        `bson:"productId,omitempty" json:"productId,omitempty"`
	VariantID string        `bson:"variantId"     json:"variantId"`
	Price     float64       `bson:"price"         json:"price"`
	CreatedAt time.Time     `bson:"createdAt"     json:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"     json:"updatedAt"`
}
