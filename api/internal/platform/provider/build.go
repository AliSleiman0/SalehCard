package provider

import "fmt"

// SupplierSpec is a transport-agnostic description of one upstream supplier,
// mapped from config.SupplierConfig at the call sites (order routes for the
// fulfillment registry, server for the admin /suppliers Cataloger map). Keeping
// it in this package lets New dispatch on Kind without the provider package
// importing config.
type SupplierSpec struct {
	ID       int
	Name     string
	Kind     string // "panel" | "telecom"
	Currency string // "USD" | "LBP"
	BaseURL  string
	// Panel auth.
	Token string
	// Telecom (umanage) auth.
	APIKey    string
	APISecret string
	StoreID   int
}

// New builds the adapter for one supplier, dispatching on Kind. A panel is the
// default (jentel/speedcard/gift4card share one adapter); "telecom" builds the
// umanage adapter. The error is returned to the caller, which skips the
// supplier (its id then falls back to the parking stub) rather than failing
// boot — the same safe-rollout contract the individual constructors document.
func New(spec SupplierSpec) (Provider, error) {
	switch spec.Kind {
	case "telecom":
		return NewUmanage(UmanageConfig{
			ID:        spec.ID,
			Name:      spec.Name,
			Currency:  spec.Currency,
			BaseURL:   spec.BaseURL,
			APIKey:    spec.APIKey,
			APISecret: spec.APISecret,
			StoreID:   spec.StoreID,
		})
	case "panel", "":
		return NewPanel(PanelConfig{
			ID:       spec.ID,
			Name:     spec.Name,
			Currency: spec.Currency,
			BaseURL:  spec.BaseURL,
			Token:    spec.Token,
		})
	default:
		return nil, fmt.Errorf("provider: unknown supplier kind %q for %q", spec.Kind, spec.Name)
	}
}
