// Package loadseed reads the catalog importer's JSON seeds and upserts them into
// MongoDB via the product + category repositories, idempotently (keyed on
// legacyId). It is the api-side half of the migration; the importer (a separate
// /migration module) writes the JSON these structs decode.
package loadseed

// The Seed* types mirror the importer's output JSON (the wire contract). They are
// intentionally separate from the api domain models so the importer schema can
// evolve independently of the persisted shape.

type SeedI18n struct {
	En *string `json:"en"`
	Ar *string `json:"ar"`
	Tr *string `json:"tr"`
}

type SeedI18nLabel struct {
	Ar *string `json:"ar"`
	En *string `json:"en"`
}

type SeedPricing struct {
	Retail         float64 `json:"retail"`
	Cost           float64 `json:"cost"`
	Margin         float64 `json:"margin"`
	Currency       string  `json:"currency"`
	Mode           string  `json:"mode"`
	ModeConfidence string  `json:"modeConfidence"`
}

type SeedAmountConstraints struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type SeedInputFieldConstraints struct {
	Min     *float64 `json:"min"`
	Max     *float64 `json:"max"`
	Options []string `json:"options"`
}

type SeedInputField struct {
	Key         string                     `json:"key"`
	Label       SeedI18nLabel              `json:"label"`
	LegacyName  string                     `json:"legacyName"`
	Type        string                     `json:"type"`
	Constraints *SeedInputFieldConstraints `json:"constraints"`
	Sensitive   bool                       `json:"sensitive"`
}

type SeedVerification struct {
	Provider int    `json:"provider"`
	App      string `json:"app"`
}

type SeedFulfillment struct {
	Type        string `json:"type"`
	Mode        string `json:"mode"`
	Provider    *int   `json:"provider"`
	Confidence  string `json:"confidence"`
	Cancellable bool   `json:"cancellable"`
}

type SeedCategory struct {
	LegacyID       int      `json:"legacyId"`
	ParentLegacyID *int     `json:"parentLegacyId"`
	Slug           string   `json:"slug"`
	Name           SeedI18n `json:"name"`
	Image          string   `json:"image"`
	SortOrder      int      `json:"sortOrder"`
	RootDomain     string   `json:"rootDomain"`
	Depth          int      `json:"depth"`
	Visible        bool     `json:"visible"`
}

type SeedProduct struct {
	LegacyID             int                    `json:"legacyId"`
	LegacyCategoryID     int                    `json:"legacyCategoryId"`
	CategorySlug         string                 `json:"categorySlug"`
	Title                SeedI18n               `json:"title"`
	Description          SeedI18n               `json:"description"`
	DescriptionHadMarkup bool                   `json:"descriptionHadMarkup"`
	Images               []string               `json:"images"`
	Pricing              SeedPricing            `json:"pricing"`
	AmountConstraints    *SeedAmountConstraints `json:"amountConstraints"`
	InputFields          []SeedInputField       `json:"inputFields"`
	Verification         *SeedVerification      `json:"verification"`
	Fulfillment          SeedFulfillment        `json:"fulfillment"`
	Status               string                 `json:"status"`
	SortOrder            int                    `json:"sortOrder"`
	Flags                []string               `json:"flags"`
}
