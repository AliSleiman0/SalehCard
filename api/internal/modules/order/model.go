package order

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// PaymentMethod identifies how an order was paid.
type PaymentMethod string

const (
	PaymentMethodWallet PaymentMethod = "wallet"
	PaymentMethodCard   PaymentMethod = "card"
	PaymentMethodUSDT   PaymentMethod = "usdt"
)

// OrderStatus tracks the lifecycle of an order.
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusFailed     OrderStatus = "failed"
	OrderStatusRefunded   OrderStatus = "refunded"
)

// OrderItem represents a single line within an order.
type OrderItem struct {
	ProductID       bson.ObjectID `bson:"productId"       json:"productId"`
	VariantID       bson.ObjectID `bson:"variantId"       json:"variantId"`
	Qty             int           `bson:"qty"             json:"qty"`
	Price           float64       `bson:"price"           json:"price"`
	FulfillmentType string        `bson:"fulfillmentType" json:"fulfillmentType"`
}

// TimelineEvent records a status transition on a fulfillment.
type TimelineEvent struct {
	Status string    `bson:"status" json:"status"`
	Note   string    `bson:"note"   json:"note"`
	At     time.Time `bson:"at"     json:"at"`
}

// Fulfillment carries delivery details for an order.
type Fulfillment struct {
	DeliveredCode  string          `bson:"deliveredCode,omitempty"   json:"deliveredCode,omitempty"`
	CreditedToID   string          `bson:"creditedToId,omitempty"    json:"creditedToId,omitempty"`
	TransferRef    string          `bson:"transferRef,omitempty"     json:"transferRef,omitempty"`
	StatusTimeline []TimelineEvent `bson:"statusTimeline"            json:"statusTimeline"`
}

// Order is the root aggregate for a customer purchase.
type Order struct {
	ID            bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        bson.ObjectID `bson:"userId"        json:"userId"`
	Items         []OrderItem   `bson:"items"         json:"items"`
	Subtotal      float64       `bson:"subtotal"      json:"subtotal"`
	Total         float64       `bson:"total"         json:"total"`
	Currency      string        `bson:"currency"      json:"currency"`
	PaymentMethod PaymentMethod `bson:"paymentMethod" json:"paymentMethod"`
	Status        OrderStatus   `bson:"status"        json:"status"`
	Fulfillment   Fulfillment   `bson:"fulfillment"   json:"fulfillment"`
	CreatedAt     time.Time     `bson:"createdAt"     json:"createdAt"`
	UpdatedAt     time.Time     `bson:"updatedAt"     json:"updatedAt"`
}

// PlaceOrderInput carries the data needed to create a new order.
type PlaceOrderInput struct {
	Items         []OrderItem   `json:"items"`
	Currency      string        `json:"currency"`
	PaymentMethod PaymentMethod `json:"paymentMethod"`
	PromoCode     string        `json:"promoCode,omitempty"`
}
