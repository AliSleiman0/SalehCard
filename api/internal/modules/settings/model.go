package settings

import (
	"strings"
	"time"
)

// settingsID is the fixed _id of the single app-settings document. The
// collection holds exactly one row, upserted by this id.
const settingsID = "app_settings"

// Exchange-rate limits: a small, admin-curated list of free-text currency rates
// (e.g. "SYP" / "89,500 per $1") shown to customers on the top-up screen.
const (
	maxExchangeRates    = 20
	maxExchangeRateText = 40
)

// ExchangeRate is one admin-defined currency rate line — both fields are free
// text, shown verbatim to the customer (no numeric parsing or formatting).
type ExchangeRate struct {
	Label string `bson:"label" json:"label"`
	Value string `bson:"value" json:"value"`
}

// Settings is the platform's editable business configuration — a single
// document. Provider secrets are NEVER stored here (they live in env/config);
// the read-only integration status is derived separately (see integrationsView).
type Settings struct {
	ID                string `bson:"_id"                json:"-"`
	StoreName         string `bson:"storeName"          json:"storeName"`
	SupportEmail      string `bson:"supportEmail"       json:"supportEmail"`
	SupportPhone      string `bson:"supportPhone"       json:"supportPhone"`
	DefaultLanguage   string `bson:"defaultLanguage"    json:"defaultLanguage"` // en / ar / tr
	DefaultCurrency   string `bson:"defaultCurrency"    json:"defaultCurrency"` // USD / TRY
	LowStockThreshold int    `bson:"lowStockThreshold"  json:"lowStockThreshold"`
	MaintenanceMode   bool   `bson:"maintenanceMode"    json:"maintenanceMode"`
	// Loyalty program. LoyaltyEnabled gates whether completed orders earn points;
	// LoyaltyEarnUsdPerPoint is the spend (in order-total dollars) that mints one
	// point, i.e. points = floor(orderTotal / LoyaltyEarnUsdPerPoint).
	LoyaltyEnabled         bool    `bson:"loyaltyEnabled"          json:"loyaltyEnabled"`
	LoyaltyEarnUsdPerPoint float64 `bson:"loyaltyEarnUsdPerPoint"  json:"loyaltyEarnUsdPerPoint"`
	// AdminSmsTwoFactorEnabled gates whether admin logins require a second factor
	// (an SMS one-time code) after the password. Off by default so the phoneless
	// dev seed admin is unaffected; enabling it fails admin logins closed when an
	// admin has no phone on file.
	AdminSmsTwoFactorEnabled bool `bson:"adminSmsTwoFactorEnabled" json:"adminSmsTwoFactorEnabled"`
	// ExchangeRates is the admin-maintained list of currency rates surfaced on the
	// customer top-up screen (global, shown for any manual payment method).
	ExchangeRates []ExchangeRate `bson:"exchangeRates,omitempty" json:"exchangeRates"`
	UpdatedAt     time.Time      `bson:"updatedAt"          json:"updatedAt"`
	UpdatedBy     string         `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
}

// defaults returns the baseline settings used when no document has been saved.
func defaults() *Settings {
	return &Settings{
		ID:                     settingsID,
		StoreName:              "SalehCard",
		DefaultLanguage:        "en",
		DefaultCurrency:        "USD",
		LowStockThreshold:      50,
		LoyaltyEnabled:         true,
		LoyaltyEarnUsdPerPoint: 5.0,
	}
}

// UpdateInput carries the editable fields for PUT /settings. Fields are pointers
// so an omitted field leaves the stored value unchanged.
type UpdateInput struct {
	StoreName         *string `json:"storeName"`
	SupportEmail      *string `json:"supportEmail"`
	SupportPhone      *string `json:"supportPhone"`
	DefaultLanguage   *string `json:"defaultLanguage"`
	DefaultCurrency   *string `json:"defaultCurrency"`
	LowStockThreshold *int    `json:"lowStockThreshold"`
	MaintenanceMode   *bool   `json:"maintenanceMode"`

	LoyaltyEnabled         *bool    `json:"loyaltyEnabled"`
	LoyaltyEarnUsdPerPoint *float64 `json:"loyaltyEarnUsdPerPoint"`

	AdminSmsTwoFactorEnabled *bool `json:"adminSmsTwoFactorEnabled"`

	ExchangeRates *[]ExchangeRate `json:"exchangeRates"`
}

// normalizeExchangeRates trims each rate, drops all-blank rows, enforces the
// count and per-field length caps, and rejects a half-filled row. Returns the
// cleaned list and an error message (empty when valid).
func normalizeExchangeRates(in []ExchangeRate) ([]ExchangeRate, string) {
	if len(in) > maxExchangeRates {
		return nil, "too many exchange rates (max 20)"
	}
	out := make([]ExchangeRate, 0, len(in))
	for _, r := range in {
		label := strings.TrimSpace(r.Label)
		value := strings.TrimSpace(r.Value)
		if label == "" && value == "" {
			continue // drop fully blank rows
		}
		if label == "" || value == "" {
			return nil, "each exchange rate needs both a label and a value"
		}
		if len(label) > maxExchangeRateText || len(value) > maxExchangeRateText {
			return nil, "exchange rate label and value must be 40 characters or fewer"
		}
		out = append(out, ExchangeRate{Label: label, Value: value})
	}
	return out, ""
}

// Integration reports whether an external provider is configured, without ever
// exposing a secret — only the selected provider name and a boolean.
type Integration struct {
	Provider   string `json:"provider"`
	Configured bool   `json:"configured"`
}

// integrationsView derives the read-only integration status from runtime config.
// A provider is "configured" when it is set to something other than the inert
// `log` dev adapter.
func integrationsView(smsProvider, pushProvider string) map[string]Integration {
	live := func(p string) bool { return p != "" && p != "log" }
	return map[string]Integration{
		"sms":  {Provider: smsProvider, Configured: live(smsProvider)},
		"push": {Provider: pushProvider, Configured: live(pushProvider)},
	}
}

// settingsView is the GET response: the stored settings plus the derived,
// secret-free integration status.
type settingsView struct {
	*Settings
	Integrations map[string]Integration `json:"integrations"`
}
