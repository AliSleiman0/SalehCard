package order

import (
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
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

// RecipientInput identifies the destination for a manual transfer fulfillment.
type RecipientInput struct {
	Name    string `bson:"name"    json:"name"`
	Country string `bson:"country" json:"country"`
	Detail  string `bson:"detail"  json:"detail"`
}

// OrderField is one customer-provided input captured at checkout (e.g. Account
// ID, Zone ID, Email), snapshotted with its localized label so the operator who
// fulfills the order sees labeled values. Label is resolved server-side from the
// product's InputFields spec — never trusted from the client. Sensitive fields
// (spec §3.2) are not persisted here.
type OrderField struct {
	Key   string            `bson:"key"   json:"key"`
	Label product.I18nLabel `bson:"label" json:"label"`
	Value string            `bson:"value" json:"value"`
}

// OrderFieldInput is one field the client submits for a line: just key + value.
// The label is looked up server-side from the product spec.
type OrderFieldInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// OrderItem represents a single line within an order. Price and FulfillmentType
// are server-derived (never trusted from the client); PlayerID/Recipient carry
// the fulfillment target for account_credit / transfer lines. Title, Denomination
// and Category are snapshots of the product at purchase time so order history
// renders correctly even if the catalog later changes.
type OrderItem struct {
	ProductID       bson.ObjectID      `bson:"productId"          json:"productId"`
	VariantID       bson.ObjectID      `bson:"variantId"          json:"variantId"`
	Title           product.I18nString `bson:"title"              json:"title"`
	Denomination    string             `bson:"denomination"       json:"denomination"`
	Category        string             `bson:"category"           json:"category"`
	Qty             int                `bson:"qty"                json:"qty"`
	Price           float64            `bson:"price"              json:"price"`
	FulfillmentType string             `bson:"fulfillmentType"    json:"fulfillmentType"`
	// FulfillmentMode/FulfillmentProvider are snapshots of the product's
	// execution path at order time; the dispatcher routes on FulfillmentMode.
	FulfillmentMode     string `bson:"fulfillmentMode,omitempty"     json:"fulfillmentMode,omitempty"`
	FulfillmentProvider *int   `bson:"fulfillmentProvider,omitempty" json:"fulfillmentProvider,omitempty"`
	// UpstreamProductID snapshots the supplier's own product id for api-mode
	// lines (the panel newOrder path segment).
	UpstreamProductID string          `bson:"upstreamProductId,omitempty" json:"upstreamProductId,omitempty"`
	PlayerID          string          `bson:"playerId,omitempty"          json:"playerId,omitempty"`
	Recipient         *RecipientInput `bson:"recipient,omitempty"         json:"recipient,omitempty"`
	// Fields are the labeled customer inputs for this line (Account ID, Zone ID,
	// Email, …), shown to the operator who fulfills the order.
	Fields []OrderField `bson:"fields,omitempty" json:"fields,omitempty"`
}

// TimelineEvent records a status transition on a fulfillment.
type TimelineEvent struct {
	Status string    `bson:"status" json:"status"`
	Note   string    `bson:"note"   json:"note"`
	At     time.Time `bson:"at"     json:"at"`
}

// Fulfillment carries delivery details for an order.
type Fulfillment struct {
	DeliveredCode string `bson:"deliveredCode,omitempty"   json:"deliveredCode,omitempty"`
	CreditedToID  string `bson:"creditedToId,omitempty"    json:"creditedToId,omitempty"`
	TransferRef   string `bson:"transferRef,omitempty"     json:"transferRef,omitempty"`
	// ProviderRef is the upstream supplier's order id for api-mode orders —
	// kept distinct from TransferRef so refunds/manual transfers stay
	// untangled; the supplier settler (design Phase 2) reconciles parked
	// orders by it via Provider.CheckStatus.
	ProviderRef    string          `bson:"providerRef,omitempty"     json:"providerRef,omitempty"`
	StatusTimeline []TimelineEvent `bson:"statusTimeline"            json:"statusTimeline"`
}

// Order is the root aggregate for a customer purchase.
type Order struct {
	ID            bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        bson.ObjectID `bson:"userId"        json:"userId"`
	Items         []OrderItem   `bson:"items"         json:"items"`
	Subtotal      float64       `bson:"subtotal"      json:"subtotal"`
	Discount      float64       `bson:"discount,omitempty"  json:"discount,omitempty"`
	PromoCode     string        `bson:"promoCode,omitempty" json:"promoCode,omitempty"`
	Total         float64       `bson:"total"         json:"total"`
	Currency      string        `bson:"currency"      json:"currency"`
	PaymentMethod PaymentMethod `bson:"paymentMethod" json:"paymentMethod"`
	// PaymentRef is the gateway transaction id for card/usdt orders (empty for
	// wallet orders); used to reverse the charge on refund / compensation.
	PaymentRef     string      `bson:"paymentRef,omitempty" json:"paymentRef,omitempty"`
	Status         OrderStatus `bson:"status"        json:"status"`
	Fulfillment    Fulfillment `bson:"fulfillment"   json:"fulfillment"`
	IdempotencyKey string      `bson:"idempotencyKey,omitempty" json:"-"`
	CreatedAt      time.Time   `bson:"createdAt"     json:"createdAt"`
	UpdatedAt      time.Time   `bson:"updatedAt"     json:"updatedAt"`
}

// PlaceOrderItemInput is a single requested line. The client sends only what it
// is allowed to choose — product, variant, quantity, and the fulfillment target
// — never the price or fulfillment type (the server derives those).
type PlaceOrderItemInput struct {
	ProductID string            `json:"productId"`
	VariantID string            `json:"variantId"`
	Qty       int               `json:"qty"`
	PlayerID  string            `json:"playerId,omitempty"`
	Recipient *RecipientInput   `json:"recipient,omitempty"`
	Fields    []OrderFieldInput `json:"fields,omitempty"`
}

// PlaceOrderInput carries the data needed to create a new order.
type PlaceOrderInput struct {
	Items         []PlaceOrderItemInput `json:"items"`
	Currency      string                `json:"currency"`
	PaymentMethod PaymentMethod         `json:"paymentMethod"`
	// UsdtNetwork picks the on-chain network for a usdt order (trc20/bep20);
	// empty means the payment module's default (old-app compatibility).
	UsdtNetwork string `json:"usdtNetwork,omitempty"`
	PromoCode   string `json:"promoCode,omitempty"`
}
