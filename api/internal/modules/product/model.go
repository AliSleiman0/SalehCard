package product

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// FulfillmentType describes how a product is delivered after purchase.
type FulfillmentType string

const (
	FulfillmentCode     FulfillmentType = "code"
	FulfillmentCredit   FulfillmentType = "account_credit"
	FulfillmentTransfer FulfillmentType = "transfer"
)

// I18nString holds a localised string in the three supported locales.
type I18nString struct {
	En string `bson:"en" json:"en"`
	Ar string `bson:"ar" json:"ar"`
	Tr string `bson:"tr" json:"tr"`
}

// Variant represents a purchasable denomination of a product.
type Variant struct {
	ID            bson.ObjectID `bson:"_id"                     json:"id"`
	Denomination  string        `bson:"denomination"            json:"denomination"`
	Price         float64       `bson:"price"                   json:"price"`
	ResellerPrice *float64      `bson:"resellerPrice,omitempty" json:"resellerPrice,omitempty"`
}

// RatingsSummary holds the aggregated rating data for a product.
type RatingsSummary struct {
	Average float64 `bson:"average" json:"average"`
	Count   int     `bson:"count"   json:"count"`
}

// Product is the root entity for a digital gift-card or top-up item.
type Product struct {
	ID              bson.ObjectID   `bson:"_id,omitempty"   json:"id"`
	Title           I18nString      `bson:"title"           json:"title"`
	Category        string          `bson:"category"        json:"category"`
	Images          []string        `bson:"images"          json:"images"`
	Variants        []Variant       `bson:"variants"        json:"variants"`
	FulfillmentType FulfillmentType `bson:"fulfillmentType" json:"fulfillmentType"`
	Stock           int             `bson:"stock"           json:"stock"`
	Available       bool            `bson:"available"       json:"available"`
	Ratings         RatingsSummary  `bson:"ratings"         json:"ratings"`
	CreatedAt       time.Time       `bson:"createdAt"       json:"createdAt"`
	UpdatedAt       time.Time       `bson:"updatedAt"       json:"updatedAt"`
}

// CreateProductInput carries all the data required to create a new product.
type CreateProductInput struct {
	Title           I18nString      `json:"title"`
	Category        string          `json:"category"`
	Images          []string        `json:"images"`
	Variants        []Variant       `json:"variants"`
	FulfillmentType FulfillmentType `json:"fulfillmentType"`
	Stock           int             `json:"stock"`
	Available       bool            `json:"available"`
	Ratings         RatingsSummary  `json:"ratings"`
}

// UpdateProductInput carries the optional fields that can be patched on a product.
// A nil pointer means "leave unchanged".
type UpdateProductInput struct {
	Title           *I18nString      `json:"title,omitempty"`
	Category        *string          `json:"category,omitempty"`
	Images          []string         `json:"images,omitempty"`
	Variants        []Variant        `json:"variants,omitempty"`
	FulfillmentType *FulfillmentType `json:"fulfillmentType,omitempty"`
	Stock           *int             `json:"stock,omitempty"`
	Available       *bool            `json:"available,omitempty"`
	Ratings         *RatingsSummary  `json:"ratings,omitempty"`
}

// ListFilter holds the optional query filters for listing products.
type ListFilter struct {
	Category  string
	Available *bool
}
