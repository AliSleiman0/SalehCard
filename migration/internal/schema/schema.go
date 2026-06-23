// Package schema defines the target (output) seed shapes the importer writes.
// These are the migration contract consumed by the api-side loader; field names
// align with the spec's §4 schema.
package schema

// I18n is a localized string in en/ar/tr. Pointers so a missing locale emits
// JSON null (tr is almost always null pending translation), which the review
// tooling keys on.
type I18n struct {
	En *string `json:"en"`
	Ar *string `json:"ar"`
	Tr *string `json:"tr"`
}

// I18nLabel is the 2-locale (ar/en) shape legacy input-field labels carry.
type I18nLabel struct {
	Ar *string `json:"ar"`
	En *string `json:"en"`
}

// Category is one migrated category.
type Category struct {
	LegacyID       int    `json:"legacyId"`
	ParentLegacyID *int   `json:"parentLegacyId"` // null for roots
	Slug           string `json:"slug"`
	Name           I18n   `json:"name"`
	Image          string `json:"image"`
	SortOrder      int    `json:"sortOrder"`
	RootDomain     string `json:"rootDomain"`
	Depth          int    `json:"depth"` // 0 = root
	Visible        bool   `json:"visible"`
}

// Pricing carries the raw economics faithfully; interpretation (mode) is
// classified and flagged, never used to normalize prices.
type Pricing struct {
	Retail         float64 `json:"retail"`
	Cost           float64 `json:"cost"`
	Margin         float64 `json:"margin"`
	Currency       string  `json:"currency"`
	Mode           string  `json:"mode"`           // unit_balance | fixed_package | currency_value
	ModeConfidence string  `json:"modeConfidence"` // high | low
}

// AmountConstraints bound a variable-amount product.
type AmountConstraints struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// InputFieldConstraints is polymorphic: numeric {min,max} OR enum {options}.
type InputFieldConstraints struct {
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Options []string `json:"options,omitempty"`
}

// InputField is one customer-input spec entry.
type InputField struct {
	Key         string                 `json:"key"`
	Label       I18nLabel              `json:"label"`
	LegacyName  string                 `json:"legacyName"`
	Type        string                 `json:"type"` // text | amount | quantity | select
	Constraints *InputFieldConstraints `json:"constraints"`
	Sensitive   bool                   `json:"sensitive"`
}

// Verification is the check_name ID-verification hook.
type Verification struct {
	Provider int    `json:"provider"`
	App      string `json:"app"`
}

// Fulfillment classifies how an order is executed.
type Fulfillment struct {
	Type        string `json:"type"`     // code | account_credit | transfer
	Mode        string `json:"mode"`     // api | manual_operator | inventory | bridge_device
	Provider    *int   `json:"provider"` // numeric provider id when known
	Confidence  string `json:"confidence"`
	Cancellable bool   `json:"cancellable"`
}

// Product is one migrated product in the target schema.
type Product struct {
	LegacyID             int                `json:"legacyId"`
	LegacyCategoryID     int                `json:"legacyCategoryId"`
	CategorySlug         string             `json:"categorySlug"`
	Title                I18n               `json:"title"`
	Description          I18n               `json:"description"`
	DescriptionHadMarkup bool               `json:"descriptionHadMarkup"`
	Images               []string           `json:"images"`
	Pricing              Pricing            `json:"pricing"`
	AmountConstraints    *AmountConstraints `json:"amountConstraints"`
	InputFields          []InputField       `json:"inputFields"`
	Verification         *Verification      `json:"verification"`
	Fulfillment          Fulfillment        `json:"fulfillment"`
	Status               string             `json:"status"` // active | unavailable
	SortOrder            int                `json:"sortOrder"`
	Flags                []string           `json:"flags"`
}

// Flag constants — the per-item review markers aggregated into _review.json.
const (
	FlagNeedsTranslation  = "needs_translation"
	FlagPricingReview     = "pricing_review"
	FlagFulfillmentReview = "fulfillment_review"
	FlagSensitiveInput    = "sensitive_input"
	FlagMarkupStripped    = "markup_stripped"
	FlagFxRateReview      = "fx_rate_review"
	FlagOrphanCategory    = "orphan_category"
	FlagAmountCorruption  = "amount_corruption" // min >= max on amount/qty
)

// Fulfillment type / mode constants (mirror the api side).
const (
	TypeCode     = "code"
	TypeCredit   = "account_credit"
	TypeTransfer = "transfer"

	ModeAPI            = "api"
	ModeManualOperator = "manual_operator"
	ModeInventory      = "inventory"
	ModeBridgeDevice   = "bridge_device"

	ConfHigh   = "high"
	ConfMedium = "medium"
	ConfLow    = "low"

	PriceUnitBalance   = "unit_balance"
	PriceFixedPackage  = "fixed_package"
	PriceCurrencyValue = "currency_value"
)
