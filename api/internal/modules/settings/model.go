package settings

import "time"

// settingsID is the fixed _id of the single app-settings document. The
// collection holds exactly one row, upserted by this id.
const settingsID = "app_settings"

// Settings is the platform's editable business configuration — a single
// document. Provider secrets are NEVER stored here (they live in env/config);
// the read-only integration status is derived separately (see integrationsView).
type Settings struct {
	ID                string    `bson:"_id"                json:"-"`
	StoreName         string    `bson:"storeName"          json:"storeName"`
	SupportEmail      string    `bson:"supportEmail"       json:"supportEmail"`
	SupportPhone      string    `bson:"supportPhone"       json:"supportPhone"`
	DefaultLanguage   string    `bson:"defaultLanguage"    json:"defaultLanguage"` // en / ar / tr
	DefaultCurrency   string    `bson:"defaultCurrency"    json:"defaultCurrency"` // USD / TRY
	LowStockThreshold int       `bson:"lowStockThreshold"  json:"lowStockThreshold"`
	MaintenanceMode   bool      `bson:"maintenanceMode"    json:"maintenanceMode"`
	UpdatedAt         time.Time `bson:"updatedAt"          json:"updatedAt"`
	UpdatedBy         string    `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
}

// defaults returns the baseline settings used when no document has been saved.
func defaults() *Settings {
	return &Settings{
		ID:                settingsID,
		StoreName:         "SalehCard",
		DefaultLanguage:   "en",
		DefaultCurrency:   "USD",
		LowStockThreshold: 50,
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
