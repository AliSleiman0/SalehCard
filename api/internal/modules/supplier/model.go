// Package supplier is the admin surface for the upstream fulfillment suppliers
// (DESIGN-SUPPLIERS.md Phase 3). It CONSUMES platform/provider — it does not
// replace it: each configured supplier id resolves (through the order module's
// provider registry) to an adapter that optionally implements provider.Cataloger
// (balance probe + catalog listing). The module adds per-supplier operational
// settings (a supplier_settings collection), catalog browse/import, price-drift
// sync, and a recent-orders trace. Supplier existence stays env-driven (ids +
// credentials are deploy config); only thresholds/markup are runtime-editable.
package supplier

import "time"

// Settings is the per-supplier operational config, one document per provider id
// (keyed by the int id as _id in supplier_settings). Zero values are the
// defaults (no low-balance alert, no default markup).
type Settings struct {
	ID                  int       `bson:"_id"                 json:"id"`
	LowBalanceThreshold float64   `bson:"lowBalanceThreshold" json:"lowBalanceThreshold"`
	MarkupPercent       float64   `bson:"markupPercent"       json:"markupPercent"`
	UpdatedAt           time.Time `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	UpdatedBy           string    `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
}

// SettingsInput carries the editable fields for PUT /suppliers/{id}/settings.
// Pointers so an omitted field leaves the stored value unchanged.
type SettingsInput struct {
	LowBalanceThreshold *float64 `json:"lowBalanceThreshold"`
	MarkupPercent       *float64 `json:"markupPercent"`
}

// SupplierView is one row of GET /suppliers: identity + live balance/health +
// mapped-product count + operational settings.
type SupplierView struct {
	ID                  int      `json:"id"`
	Name                string   `json:"name"`
	Kind                string   `json:"kind"`
	Currency            string   `json:"currency"`
	BaseURL             string   `json:"baseUrl"`
	Health              string   `json:"health"`
	Balance             *float64 `json:"balance"`
	BalanceText         string   `json:"balanceText"`
	MappedProducts      int64    `json:"mappedProducts"`
	LowBalanceThreshold float64  `json:"lowBalanceThreshold"`
	MarkupPercent       float64  `json:"markupPercent"`
}

// CatalogProductView is one row of GET /suppliers/{id}/catalog.
type CatalogProductView struct {
	UpstreamID  string   `json:"upstreamId"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	ParentID    string   `json:"parentId"`
	Price       float64  `json:"price"`
	BasePrice   float64  `json:"basePrice"`
	Currency    string   `json:"currency"`
	Available   bool     `json:"available"`
	ProductType string   `json:"productType"`
	Params      []string `json:"params"`
	QtyMin      *int     `json:"qtyMin"`
	QtyMax      *int     `json:"qtyMax"`
	QtyValues   []string `json:"qtyValues"`
	Mapped      bool     `json:"mapped"`
}

// ImportInput is the body of POST /suppliers/{id}/import.
type ImportInput struct {
	Items []ImportItem `json:"items"`
}

// ImportItem selects one upstream product to import, with optional per-item
// category assignment and markup override.
type ImportItem struct {
	UpstreamID    string   `json:"upstreamId"`
	CategoryID    *string  `json:"categoryId"`
	MarkupPercent *float64 `json:"markupPercent"`
}

// ImportResult reports how an import went.
type ImportResult struct {
	Created int             `json:"created"`
	Skipped []ImportSkipped `json:"skipped"`
}

// ImportSkipped names an item that was not imported and why.
type ImportSkipped struct {
	UpstreamID string `json:"upstreamId"`
	Reason     string `json:"reason"`
}

// SyncResult reports a catalog sync over the already-mapped products.
type SyncResult struct {
	Checked     int          `json:"checked"`
	Updated     int          `json:"updated"`
	Unavailable int          `json:"unavailable"`
	Drift       []PriceDrift `json:"drift"`
}

// PriceDrift records a mapped product whose stored price no longer matches the
// upstream base price (informational — sync never rewrites the sell price).
type PriceDrift struct {
	UpstreamID string  `json:"upstreamId"`
	Name       string  `json:"name"`
	OldPrice   float64 `json:"oldPrice"`
	NewPrice   float64 `json:"newPrice"`
}

// RecentOrderView is one row of GET /suppliers/{id}/orders.
type RecentOrderView struct {
	ID            string     `json:"id"`
	Status        string     `json:"status"`
	Total         float64    `json:"total"`
	Currency      string     `json:"currency"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpstreamRef   string     `json:"upstreamRef"`
	DeliveredCode string     `json:"deliveredCode,omitempty"`
	PlayerID      string     `json:"playerId,omitempty"`
	StuckAt       *time.Time `json:"stuckAt"`
}
