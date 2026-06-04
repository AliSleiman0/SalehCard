package reseller

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ResellerTier holds the pricing and limit configuration for a reseller account.
type ResellerTier struct {
	ID            bson.ObjectID      `bson:"_id,omitempty" json:"id"`
	Name          string             `bson:"name"          json:"name"`
	MarginPercent float64            `bson:"marginPercent" json:"marginPercent"`
	SubBalance    float64            `bson:"subBalance"    json:"subBalance"`
	Limits        map[string]float64 `bson:"limits"        json:"limits"`
	CreatedAt     time.Time          `bson:"createdAt"     json:"createdAt"`
}
