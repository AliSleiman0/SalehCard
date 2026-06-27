package code

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Status is the lifecycle state of a single code in the pool.
type Status string

const (
	StatusAvailable Status = "available"
	StatusDelivered Status = "delivered"
	StatusExpired   Status = "expired"
)

// DefaultThreshold is the low-stock alert threshold used when a product has no
// explicit threshold configured.
const DefaultThreshold = 50

// Code is a single fulfillment code/PIN belonging to a code-type product.
type Code struct {
	ID          bson.ObjectID `bson:"_id,omitempty"          json:"id"`
	ProductID   string        `bson:"productId"              json:"productId"`
	Code        string        `bson:"code"                   json:"code"`
	Pin         string        `bson:"pin,omitempty"          json:"pin,omitempty"`
	Status      Status        `bson:"status"                 json:"status"`
	OrderID     string        `bson:"orderId,omitempty"      json:"orderId,omitempty"`
	DeliveredTo string        `bson:"deliveredTo,omitempty"  json:"deliveredTo,omitempty"`
	DeliveredAt *time.Time    `bson:"deliveredAt,omitempty"  json:"deliveredAt,omitempty"`
	Batch       string        `bson:"batch,omitempty"        json:"batch,omitempty"`
	CreatedAt   time.Time     `bson:"createdAt"              json:"createdAt"`
}

// UploadItem is one row of a bulk code upload.
type UploadItem struct {
	Code string `json:"code"`
	Pin  string `json:"pin,omitempty"`
}

// UploadInput is the body of POST /api/admin/products/:id/codes. UploadedBy is
// server-set from the caller's JWT (not trusted from the client body).
type UploadInput struct {
	Codes      []UploadItem `json:"codes"`
	Batch      string       `json:"batch,omitempty"`
	UploadedBy string       `json:"-"`
}

// UploadResult summarises what a bulk upload committed.
type UploadResult struct {
	Inserted   int `json:"inserted"`
	Duplicates int `json:"duplicates"`
	Invalid    int `json:"invalid"`
}

// UploadBatch is the persisted record of one bulk upload, written on each upload
// so the inventory screen can show a real upload history (codes themselves don't
// retain who uploaded them or the insert-time inserted/duplicate split).
type UploadBatch struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID  string        `bson:"productId"     json:"productId"`
	Batch      string        `bson:"batch"         json:"batch"`
	Inserted   int           `bson:"inserted"      json:"inserted"`
	Duplicates int           `bson:"duplicates"    json:"duplicates"`
	Invalid    int           `bson:"invalid"       json:"invalid"`
	UploadedBy string        `bson:"uploadedBy,omitempty" json:"uploadedBy,omitempty"`
	CreatedAt  time.Time     `bson:"createdAt"     json:"createdAt"`
}

// UploadBatchView is an UploadBatch enriched with the product title for display.
type UploadBatchView struct {
	UploadBatch
	ProductTitle string `json:"productTitle"`
}

// StockLevel mirrors the frontend's green/yellow/red coding.
type StockLevel string

const (
	LevelHi  StockLevel = "hi"
	LevelMid StockLevel = "mid"
	LevelLo  StockLevel = "lo"
)

// InventoryStats is the per-product code summary shown on the inventory screen.
type InventoryStats struct {
	ProductID string     `json:"productId"`
	Title     string     `json:"title"`
	Category  string     `json:"category"`
	Uploaded  int        `json:"uploaded"`
	Available int        `json:"available"`
	Delivered int        `json:"delivered"`
	Expired   int        `json:"expired"`
	Threshold int        `json:"threshold"`
	Level     StockLevel `json:"level"`
}

// CodeAudit is the lookup response: a code plus its owning product summary.
type CodeAudit struct {
	Code    Code            `json:"code"`
	Product *ProductSummary `json:"product,omitempty"`
}

// ProductSummary is the minimal product info attached to an audit result.
type ProductSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// SetThresholdInput is the body of PUT /api/admin/products/:id/stock-threshold.
type SetThresholdInput struct {
	Threshold int `json:"threshold"`
}

// computeLevel derives the stock level from available vs threshold.
func computeLevel(available, threshold int) StockLevel {
	if threshold <= 0 {
		threshold = DefaultThreshold
	}
	switch {
	case available <= 0 || available < threshold*2/5:
		return LevelLo
	case available < threshold:
		return LevelMid
	default:
		return LevelHi
	}
}
