// Package notification is the customer in-app notification inbox plus its push
// fan-out: business modules record events through the [Notifier] port (a row in
// the inbox + a best-effort push to every registered device), and customers read
// them via /api/v1/notifications.
package notification

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Notification kinds, written by the event sites in order/wallet/kyc. The mobile
// client maps these onto localized copy; unknown kinds render the raw
// title/body, so new kinds are backward-compatible.
const (
	KindOrderCompleted = "order_completed"
	KindOrderRefunded  = "order_refunded"
	KindTopUpApproved  = "topup_approved"
	KindTopUpRejected  = "topup_rejected"
	KindKYCApproved    = "kyc_approved"
	KindKYCRejected    = "kyc_rejected"
	KindCodeDelivered  = "code_delivered"
	// On-chain USDT payment lifecycle (payment module).
	KindPaymentConfirmed = "payment_confirmed"
	KindPaymentUnderpaid = "payment_underpaid"
	KindPaymentExpired   = "payment_expired"
)

// Notification is one inbox row. Title/Body are the English fallback copy (also
// what the push tray shows); Data carries the structured values (orderId,
// amount, ...) the client uses to compose localized strings.
type Notification struct {
	ID        bson.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID    bson.ObjectID     `bson:"userId" json:"-"`
	Kind      string            `bson:"kind" json:"kind"`
	Title     string            `bson:"title" json:"title"`
	Body      string            `bson:"body" json:"body"`
	Data      map[string]string `bson:"data,omitempty" json:"data,omitempty"`
	ReadAt    *time.Time        `bson:"readAt,omitempty" json:"readAt,omitempty"` // nil = unread
	CreatedAt time.Time         `bson:"createdAt" json:"createdAt"`
}

// DeviceToken is one push-capable device registration. The token is globally
// unique: re-registering an existing token reassigns it to the latest
// signed-in user (one physical device, whoever is logged in gets the pushes).
type DeviceToken struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    bson.ObjectID `bson:"userId" json:"-"`
	Token     string        `bson:"token" json:"token"`
	Platform  string        `bson:"platform" json:"platform"`
	UpdatedAt time.Time     `bson:"updatedAt" json:"updatedAt"`
}
