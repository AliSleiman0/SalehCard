package product

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// FulfillmentType describes what the customer experiences after purchase
// (the outcome). It is orthogonal to FulfillmentMode (how the order is executed).
type FulfillmentType string

const (
	FulfillmentCode     FulfillmentType = "code"
	FulfillmentCredit   FulfillmentType = "account_credit"
	FulfillmentTransfer FulfillmentType = "transfer"
)

// FulfillmentMode describes how an order is actually executed — the path the
// order engine dispatches down. It is orthogonal to FulfillmentType: e.g. a
// standard top-up and a special-offer top-up are both account_credit but route
// via api vs manual_operator respectively. See migration spec §2.2.
type FulfillmentMode string

const (
	FulfillmentModeAPI            FulfillmentMode = "api"             // upstream provider call
	FulfillmentModeManualOperator FulfillmentMode = "manual_operator" // human fulfillment queue
	FulfillmentModeInventory      FulfillmentMode = "inventory"       // dispense a pre-loaded code
	FulfillmentModeBridgeDevice   FulfillmentMode = "bridge_device"   // Mobile Bridge APK
)

// DeriveMode maps a legacy FulfillmentType to a behavior-preserving default
// FulfillmentMode. It is a safety net for products/orders that predate the
// fulfillmentMode field (and for admin-created products that omit it); the real
// per-product intent is set explicitly (seed / admin / catalog importer). The
// mapping preserves today's runtime behavior: code dispenses from inventory,
// everything else parks in the manual queue.
func DeriveMode(t FulfillmentType) FulfillmentMode {
	switch t {
	case FulfillmentCode:
		return FulfillmentModeInventory
	case FulfillmentCredit, FulfillmentTransfer:
		return FulfillmentModeManualOperator
	default:
		return FulfillmentModeManualOperator
	}
}

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

// --- Migration-rich schema (spec §3/§4) -----------------------------------
// These types carry catalog-importer fields. They are additive: existing
// products decode with zero values and the order engine/frontends ignore them.

// PricingMode classifies how a product's price should be interpreted (ambiguous
// in the legacy source — classified + flagged, never used to normalize prices).
type PricingMode string

const (
	PricingModeUnitBalance   PricingMode = "unit_balance"
	PricingModeFixedPackage  PricingMode = "fixed_package"
	PricingModeCurrencyValue PricingMode = "currency_value"
)

// Confidence is a high|medium|low (fulfillment) / high|low (pricing) marker.
type Confidence string

// Pricing carries the raw economics and the classified pricing mode.
type Pricing struct {
	Retail         float64     `bson:"retail"                   json:"retail"`
	Cost           float64     `bson:"cost"                     json:"cost"`
	Margin         float64     `bson:"margin"                   json:"margin"`
	Currency       string      `bson:"currency"                 json:"currency"`
	Mode           PricingMode `bson:"mode"                     json:"mode"`
	ModeConfidence Confidence  `bson:"modeConfidence,omitempty" json:"modeConfidence,omitempty"`
}

// AmountConstraints bound a variable-amount product.
type AmountConstraints struct {
	Min float64 `bson:"min" json:"min"`
	Max float64 `bson:"max" json:"max"`
}

// InputFieldConstraints is polymorphic: numeric {min,max} OR enum {options}.
type InputFieldConstraints struct {
	Min     *float64 `bson:"min,omitempty"     json:"min,omitempty"`
	Max     *float64 `bson:"max,omitempty"     json:"max,omitempty"`
	Options []string `bson:"options,omitempty" json:"options,omitempty"`
}

// InputFieldType is the customer-input field kind.
type InputFieldType string

const (
	InputFieldText     InputFieldType = "text"
	InputFieldAmount   InputFieldType = "amount"
	InputFieldQuantity InputFieldType = "quantity"
	InputFieldSelect   InputFieldType = "select"
)

// I18nLabel is the 2-locale (ar/en) shape input-field labels carry.
type I18nLabel struct {
	En string `bson:"en,omitempty" json:"en,omitempty"`
	Ar string `bson:"ar,omitempty" json:"ar,omitempty"`
}

// InputField is one customer-input spec entry. Values captured at runtime for a
// field with Sensitive=true must be masked, not stored long-term, and scrubbed
// from logs (spec §3.2).
type InputField struct {
	Key         string                 `bson:"key"                   json:"key"`
	Label       I18nLabel              `bson:"label"                 json:"label"`
	LegacyName  string                 `bson:"legacyName,omitempty"  json:"legacyName,omitempty"`
	Type        InputFieldType         `bson:"type"                  json:"type"`
	Constraints *InputFieldConstraints `bson:"constraints,omitempty" json:"constraints,omitempty"`
	Sensitive   bool                   `bson:"sensitive"             json:"sensitive"`
}

// Verification is the check_name ID-verification hook (spec §3.1).
type Verification struct {
	Provider int    `bson:"provider" json:"provider"`
	App      string `bson:"app"      json:"app"`
}

// Product is the root entity for a digital gift-card or top-up item.
type Product struct {
	ID    bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Title I18nString    `bson:"title"         json:"title"`
	// Migration: legacy identity (traceable, re-runnable import).
	LegacyID         *int   `bson:"legacyId,omitempty"         json:"legacyId,omitempty"`
	LegacyCategoryID *int   `bson:"legacyCategoryId,omitempty" json:"legacyCategoryId,omitempty"`
	CategorySlug     string `bson:"categorySlug,omitempty"     json:"categorySlug,omitempty"`
	// RootDomain is the top-level domain (games, app_topups, …) the product's
	// category resolves to — denormalized at load time so the storefront can
	// browse a whole domain with one indexed filter. Empty for non-migration products.
	RootDomain string    `bson:"rootDomain,omitempty" json:"rootDomain,omitempty"`
	Category   string    `bson:"category" json:"category"`
	Images     []string  `bson:"images"   json:"images"`
	Variants   []Variant `bson:"variants" json:"variants"`
	// Migration: descriptive content (HTML stripped at import).
	Description          I18nString      `bson:"description,omitempty"           json:"description,omitempty"`
	DescriptionHadMarkup bool            `bson:"descriptionHadMarkup,omitempty" json:"descriptionHadMarkup,omitempty"`
	FulfillmentType      FulfillmentType `bson:"fulfillmentType" json:"fulfillmentType"`
	// FulfillmentMode is the execution path the order engine dispatches on.
	// Optional on the wire; an empty value is read as DeriveMode(FulfillmentType).
	FulfillmentMode FulfillmentMode `bson:"fulfillmentMode,omitempty" json:"fulfillmentMode,omitempty"`
	// FulfillmentProvider is the numeric upstream-provider id for api-mode
	// products (spec §3); nil when not applicable.
	FulfillmentProvider    *int       `bson:"fulfillmentProvider,omitempty"    json:"fulfillmentProvider,omitempty"`
	FulfillmentConfidence  Confidence `bson:"fulfillmentConfidence,omitempty"  json:"fulfillmentConfidence,omitempty"`
	FulfillmentCancellable *bool      `bson:"fulfillmentCancellable,omitempty" json:"fulfillmentCancellable,omitempty"`
	// Migration: rich economics + checkout schema (spec §3/§4).
	Pricing           *Pricing           `bson:"pricing,omitempty"           json:"pricing,omitempty"`
	AmountConstraints *AmountConstraints `bson:"amountConstraints,omitempty" json:"amountConstraints,omitempty"`
	InputFields       []InputField       `bson:"inputFields,omitempty"       json:"inputFields,omitempty"`
	Verification      *Verification      `bson:"verification,omitempty"      json:"verification,omitempty"`
	Stock             int                `bson:"stock"     json:"stock"`
	Available         bool               `bson:"available" json:"available"`
	// Migration: status fidelity + ordering + review markers.
	Status    string         `bson:"status,omitempty"    json:"status,omitempty"`
	SortOrder int            `bson:"sortOrder,omitempty" json:"sortOrder,omitempty"`
	Flags     []string       `bson:"flags,omitempty"     json:"flags,omitempty"`
	Ratings   RatingsSummary `bson:"ratings"   json:"ratings"`
	CreatedAt time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time      `bson:"updatedAt" json:"updatedAt"`
}

// UpsertProductInput is the loader-only input for the catalog migration
// (api/cmd/loadseed). It is keyed on LegacyID and carries the full rich schema
// plus the synthesized Variants — kept distinct from the admin Create/Update
// DTOs, which are validated for the product editor.
type UpsertProductInput struct {
	LegacyID               int
	LegacyCategoryID       *int
	CategorySlug           string
	RootDomain             string
	Title                  I18nString
	Description            I18nString
	DescriptionHadMarkup   bool
	Category               string
	Images                 []string
	Variants               []Variant
	FulfillmentType        FulfillmentType
	FulfillmentMode        FulfillmentMode
	FulfillmentProvider    *int
	FulfillmentConfidence  Confidence
	FulfillmentCancellable *bool
	Pricing                *Pricing
	AmountConstraints      *AmountConstraints
	InputFields            []InputField
	Verification           *Verification
	Available              bool
	Status                 string
	SortOrder              int
	Flags                  []string
}

// CreateProductInput carries all the data required to create a new product.
type CreateProductInput struct {
	Title               I18nString      `json:"title"`
	Description         I18nString      `json:"description"`
	Category            string          `json:"category"`
	Images              []string        `json:"images"`
	Variants            []Variant       `json:"variants"`
	FulfillmentType     FulfillmentType `json:"fulfillmentType"`
	FulfillmentMode     FulfillmentMode `json:"fulfillmentMode,omitempty"`
	FulfillmentProvider *int            `json:"fulfillmentProvider,omitempty"`
	Stock               int             `json:"stock"`
	Available           bool            `json:"available"`
	Ratings             RatingsSummary  `json:"ratings"`
}

// UpdateProductInput carries the optional fields that can be patched on a product.
// A nil pointer means "leave unchanged".
type UpdateProductInput struct {
	Title               *I18nString      `json:"title,omitempty"`
	Description         *I18nString      `json:"description,omitempty"`
	Category            *string          `json:"category,omitempty"`
	Images              []string         `json:"images,omitempty"`
	Variants            []Variant        `json:"variants,omitempty"`
	FulfillmentType     *FulfillmentType `json:"fulfillmentType,omitempty"`
	FulfillmentMode     *FulfillmentMode `json:"fulfillmentMode,omitempty"`
	FulfillmentProvider *int             `json:"fulfillmentProvider,omitempty"`
	Stock               *int             `json:"stock,omitempty"`
	Available           *bool            `json:"available,omitempty"`
	Ratings             *RatingsSummary  `json:"ratings,omitempty"`
}

// ListFilter holds the optional query filters for listing products.
// Category/Available are used by the customer-facing list; FulfillmentType,
// Status, and Search are additional filters used by the admin list.
type ListFilter struct {
	Category        string
	RootDomain      string
	Available       *bool
	FulfillmentType FulfillmentType
	FulfillmentMode FulfillmentMode
	Status          string // "active" | "draft" | "out"
	Search          string
}

// BulkAction is a bulk operation applied to a set of product IDs.
type BulkAction string

const (
	BulkActivate   BulkAction = "activate"
	BulkDeactivate BulkAction = "deactivate"
	BulkDelete     BulkAction = "delete"
)

// BulkInput is the request body for POST /api/admin/products/bulk.
type BulkInput struct {
	IDs    []string   `json:"ids"`
	Action BulkAction `json:"action"`
}
