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
