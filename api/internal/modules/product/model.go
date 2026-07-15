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
	// FaceValue is the amount the bridge transfers on a transfer_credit recharge
	// (e.g. a "$5 Alfa recharge" variant has FaceValue 5). It is the operator
	// credit moved to the customer's line, distinct from Price (what the customer
	// pays). Only meaningful for bridge_device products with the transfer_credit
	// method; nil otherwise.
	FaceValue *float64 `bson:"faceValue,omitempty" json:"faceValue,omitempty"`
	// OfferPrice is the discounted unit price when a live offer applies. It is
	// transient (never persisted) and only set on catalog reads. A pointer, not
	// a bare float, so a legitimate 0 sale price still serializes.
	OfferPrice *float64 `bson:"-" json:"offerPrice,omitempty"`
}

// BridgeProvider is the Lebanese mobile operator a bridge_device recharge targets.
type BridgeProvider string

const (
	BridgeProviderTouch BridgeProvider = "touch" // MTC Touch
	BridgeProviderAlfa  BridgeProvider = "alfa"
)

// BridgeMethod is how a bridge_device recharge is delivered on the SIM: a credit
// transfer from the SIM's own prepaid balance (by amount), or applying a
// scratch-card code (claimed from the code inventory) to the customer's line.
type BridgeMethod string

const (
	BridgeMethodTransferCredit BridgeMethod = "transfer_credit"
	BridgeMethodRechargeLine   BridgeMethod = "recharge_line"
)

// BridgeSpec is the bridge_device fulfillment configuration on a product: which
// operator and which delivery method. Present only on bridge_device products.
type BridgeSpec struct {
	Provider BridgeProvider `bson:"provider" json:"provider"`
	Method   BridgeMethod   `bson:"method"   json:"method"`
}

// Valid reports whether the spec names a supported operator and method.
func (b BridgeSpec) Valid() bool {
	switch b.Provider {
	case BridgeProviderTouch, BridgeProviderAlfa:
	default:
		return false
	}
	switch b.Method {
	case BridgeMethodTransferCredit, BridgeMethodRechargeLine:
		return true
	default:
		return false
	}
}

// OfferInfo is the product-level live-offer summary attached to catalog reads.
// It is transient (never persisted). Field names match the GET /api/v1/offers
// payload so clients can reuse one parser shape.
type OfferInfo struct {
	DiscountType      string     `json:"discountType"`
	DiscountValue     float64    `json:"discountValue"`
	OriginalFromPrice float64    `json:"originalFromPrice"`
	OfferFromPrice    float64    `json:"offerFromPrice"`
	EndsAt            *time.Time `json:"endsAt,omitempty"`
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
	RootDomain string `bson:"rootDomain,omitempty" json:"rootDomain,omitempty"`
	Category   string `bson:"category" json:"category"`
	// CategoryID references a node in the managed category taxonomy (nil for
	// unassigned / pre-migration products). The flat Category/RootDomain strings
	// are kept denormalized from it so the legacy flat filters keep working.
	CategoryID *bson.ObjectID `bson:"categoryId,omitempty" json:"categoryId,omitempty"`
	Images     []string       `bson:"images"   json:"images"`
	// Thumbnail is the 256px compressed preview URL (see platform/imaging);
	// lists/chips prefer it, falling back to Images[0]. Images[0] stays the
	// 1024px display URL for backward compatibility (Flutter reads images.first).
	Thumbnail string    `bson:"thumbnail,omitempty" json:"thumbnail"`
	Variants  []Variant `bson:"variants" json:"variants"`
	// Migration: descriptive content (HTML stripped at import).
	Description          I18nString      `bson:"description,omitempty"           json:"description,omitempty"`
	DescriptionHadMarkup bool            `bson:"descriptionHadMarkup,omitempty" json:"descriptionHadMarkup,omitempty"`
	FulfillmentType      FulfillmentType `bson:"fulfillmentType" json:"fulfillmentType"`
	// FulfillmentMode is the execution path the order engine dispatches on.
	// Optional on the wire; an empty value is read as DeriveMode(FulfillmentType).
	FulfillmentMode FulfillmentMode `bson:"fulfillmentMode,omitempty" json:"fulfillmentMode,omitempty"`
	// FulfillmentProvider is the numeric upstream-provider id for api-mode
	// products (spec §3); nil when not applicable.
	FulfillmentProvider *int `bson:"fulfillmentProvider,omitempty"    json:"fulfillmentProvider,omitempty"`
	// UpstreamProductID is the supplier's own product id (e.g. a panel's "364")
	// for api-mode products — the path segment of the panel newOrder call.
	// Meaningful only when FulfillmentMode == api; empty otherwise.
	UpstreamProductID      string     `bson:"upstreamProductId,omitempty"      json:"upstreamProductId,omitempty"`
	FulfillmentConfidence  Confidence `bson:"fulfillmentConfidence,omitempty"  json:"fulfillmentConfidence,omitempty"`
	FulfillmentCancellable *bool      `bson:"fulfillmentCancellable,omitempty" json:"fulfillmentCancellable,omitempty"`
	// Migration: rich economics + checkout schema (spec §3/§4).
	Pricing           *Pricing           `bson:"pricing,omitempty"           json:"pricing,omitempty"`
	AmountConstraints *AmountConstraints `bson:"amountConstraints,omitempty" json:"amountConstraints,omitempty"`
	InputFields       []InputField       `bson:"inputFields,omitempty"       json:"inputFields,omitempty"`
	Verification      *Verification      `bson:"verification,omitempty"      json:"verification,omitempty"`
	// Bridge configures Lebanese mobile-recharge fulfillment for bridge_device
	// products (operator + delivery method); nil for all other products.
	Bridge    *BridgeSpec `bson:"bridge,omitempty"            json:"bridge,omitempty"`
	Stock     int         `bson:"stock"     json:"stock"`
	Available bool        `bson:"available" json:"available"`
	// Migration: status fidelity + ordering + review markers.
	Status    string         `bson:"status,omitempty"    json:"status,omitempty"`
	SortOrder int            `bson:"sortOrder,omitempty" json:"sortOrder,omitempty"`
	Flags     []string       `bson:"flags,omitempty"     json:"flags,omitempty"`
	Ratings   RatingsSummary `bson:"ratings"   json:"ratings"`
	CreatedAt time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time      `bson:"updatedAt" json:"updatedAt"`
	// Offer is the live sale on this product, if any. Transient (never persisted);
	// set only on catalog reads by the product service's offer enrichment.
	Offer *OfferInfo `bson:"-" json:"offer,omitempty"`
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
	Thumbnail              string
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

// normalize repairs invariants JSON consumers rely on: Images is always an
// array. Legacy-import documents can store images as null/absent, which decodes
// to a nil slice and would marshal as JSON null — crashing clients that index
// images[0] (the admin product list did exactly that). Call after every decode
// (and before insert) in the repository.
func (p *Product) normalize() {
	if p.Images == nil {
		p.Images = []string{}
	}
}

// CreateProductInput carries all the data required to create a new product.
type CreateProductInput struct {
	Title       I18nString `json:"title"`
	Description  I18nString `json:"description"`
	Category     string     `json:"category"`
	// CategoryID assigns the product to a taxonomy node. When set, the service
	// resolves it and denormalizes Category (slug) + RootDomain onto the product.
	CategoryID          *string         `json:"categoryId,omitempty"`
	RootDomain          string          `json:"-"` // derived server-side from CategoryID; never client-set
	// Cost is the admin-set unit cost (Pricing.Cost). Nil = unset; a real 0 is a
	// legal value. Only Cost is admin-editable — Retail/Margin/Mode stay importer-owned.
	Cost                *float64        `json:"cost,omitempty"`
	Images              []string        `json:"images"`
	Thumbnail           string          `json:"thumbnail,omitempty"`
	Variants            []Variant       `json:"variants"`
	FulfillmentType     FulfillmentType `json:"fulfillmentType"`
	FulfillmentMode     FulfillmentMode `json:"fulfillmentMode,omitempty"`
	FulfillmentProvider *int            `json:"fulfillmentProvider,omitempty"`
	// UpstreamProductID maps an api-mode product onto the supplier's catalog
	// (DESIGN-SUPPLIERS.md). Deliberately unvalidated: a provider-less or
	// id-less api product parks safely at order time (stub / ErrUnavailable).
	UpstreamProductID string          `json:"upstreamProductId,omitempty"`
	Bridge            *BridgeSpec     `json:"bridge,omitempty"`
	InputFields       []InputField    `json:"inputFields,omitempty"`
	Stock             int             `json:"stock"`
	Available         bool            `json:"available"`
	Ratings           RatingsSummary  `json:"ratings"`
}

// UpdateProductInput carries the optional fields that can be patched on a product.
// A nil pointer means "leave unchanged".
type UpdateProductInput struct {
	Title       *I18nString `json:"title,omitempty"`
	Description  *I18nString `json:"description,omitempty"`
	Category     *string     `json:"category,omitempty"`
	// CategoryID re-assigns the product to a taxonomy node. When set, the service
	// resolves it and denormalizes Category (slug) + RootDomain from it.
	CategoryID          *string          `json:"categoryId,omitempty"`
	RootDomain          *string          `json:"-"` // derived server-side from CategoryID; never client-set
	// Cost patches the admin-set unit cost (Pricing.Cost) via the dot-path
	// `pricing.cost`, preserving importer-owned Retail/Margin/Currency. Nil = unchanged.
	Cost                *float64         `json:"cost,omitempty"`
	Images              []string         `json:"images,omitempty"`
	Thumbnail           *string          `json:"thumbnail,omitempty"`
	Variants            []Variant        `json:"variants,omitempty"`
	FulfillmentType *FulfillmentType `json:"fulfillmentType,omitempty"`
	FulfillmentMode *FulfillmentMode `json:"fulfillmentMode,omitempty"`
	// FulfillmentProvider: nil = unchanged; a non-nil 0 clears the stored id
	// ($unset — 0 is never a valid registry id, mirroring the Verification
	// clear-sentinel convention).
	FulfillmentProvider *int `json:"fulfillmentProvider,omitempty"`
	// UpstreamProductID: nil = unchanged; non-nil "" clears.
	UpstreamProductID *string `json:"upstreamProductId,omitempty"`
	Stock             *int    `json:"stock,omitempty"`
	Available           *bool            `json:"available,omitempty"`
	Ratings             *RatingsSummary  `json:"ratings,omitempty"`
	InputFields         []InputField     `json:"inputFields,omitempty"`
	// Verification configures the check_name ID-verification hook. A non-nil value
	// with a non-empty App sets it; a non-nil value with an empty App clears it
	// (disables verification). Nil leaves it unchanged.
	Verification *Verification `json:"verification,omitempty"`
	// Bridge configures bridge_device recharge fulfillment. A non-nil value with a
	// non-empty Provider sets it; a non-nil value with an empty Provider clears it
	// (product is no longer bridge-fulfilled). Nil leaves it unchanged.
	Bridge *BridgeSpec `json:"bridge,omitempty"`
}

// ListFilter holds the optional query filters for listing products.
// Category/Available are used by the customer-facing list; FulfillmentType,
// Status, and Search are additional filters used by the admin list.
type ListFilter struct {
	Category string
	// CategoryID is the taxonomy-node filter (raw hex from ?categoryId=). The
	// service expands it to the node + its descendants and sets CategoryIDs,
	// which buildFilter matches with $in (tree-aware listing).
	CategoryID      string
	CategoryIDs     []bson.ObjectID
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
	BulkActivate       BulkAction = "activate"
	BulkDeactivate     BulkAction = "deactivate"
	BulkDelete         BulkAction = "delete"
	BulkAssignCategory BulkAction = "assign-category"
)

// BulkInput is the request body for POST /api/admin/products/bulk.
type BulkInput struct {
	IDs    []string   `json:"ids"`
	Action BulkAction `json:"action"`
	// CategoryID is the taxonomy node to assign for the "assign-category"
	// action; empty clears the assignment (unassign). Ignored otherwise.
	CategoryID string `json:"categoryId"`
}
