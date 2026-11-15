package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// MaxSupplierLinks caps how many suppliers one product can be linked to.
const MaxSupplierLinks = 6

// legacyLinkID is the id of the virtual link EffectiveLinks builds for a
// product that has the single legacy supplier mapping but no SupplierLinks yet
// (not migrated). Saving the product turns it into a real link.
const legacyLinkID = "legacy"

// SupplierLink is one supplier product this product can be fulfilled by (the
// old admin's "API link" rows). Product.SupplierLinks is ordered: the admin's
// order is the fallback priority. Exactly one link is active; the legacy
// FulfillmentProvider/UpstreamProductID fields always mirror it, so order
// placement, the settler and every existing reader keep working unchanged.
type SupplierLink struct {
	ID         string    `bson:"id"                json:"id"`
	Provider   int       `bson:"provider"          json:"provider"`
	UpstreamID string    `bson:"upstreamId"        json:"upstreamId"`
	Enabled    bool      `bson:"enabled"           json:"enabled"`
	AddedAt    time.Time `bson:"addedAt"           json:"addedAt"`
	AddedBy    string    `bson:"addedBy,omitempty" json:"addedBy,omitempty"`
}

// SupplierLinkInput is one row of the editor's links table. ID is empty for a
// row added in this edit; Active marks the one the product is fulfilled by.
type SupplierLinkInput struct {
	ID         string `json:"id,omitempty"`
	Provider   int    `json:"provider"`
	UpstreamID string `json:"upstreamId"`
	Enabled    bool   `json:"enabled"`
	Active     bool   `json:"active"`
}

// SupplierLinksInput replaces a product's whole links table. Rev is the
// supplierLinksRev the editor loaded; a stale one is refused
// (SUPPLIER_LINKS_CHANGED) so two admins can't silently overwrite each other.
type SupplierLinksInput struct {
	Links []SupplierLinkInput `json:"links"`
	Rev   int                 `json:"rev"`
}

// AdminProductView is the admin read of a product: the public product JSON
// plus the supplier links table (which never appears in public JSON).
type AdminProductView struct {
	*Product
	SupplierLinks        []SupplierLink `json:"supplierLinks"`
	ActiveSupplierLinkID string         `json:"activeSupplierLinkId"`
	SupplierLinksRev     int            `json:"supplierLinksRev"`
	// SupplierFallback: the order engine falls back to the next link when the
	// active one definitely refuses an order.
	SupplierFallback bool `json:"supplierFallback"`
}

// NewAdminProductView builds the admin read of p (links resolved through
// EffectiveLinks, so an unmigrated product shows its single legacy link).
func NewAdminProductView(p *Product) AdminProductView {
	links, active := EffectiveLinks(p)
	if links == nil {
		links = []SupplierLink{}
	}
	return AdminProductView{Product: p, SupplierLinks: links, ActiveSupplierLinkID: active, SupplierLinksRev: p.SupplierLinksRev, SupplierFallback: !p.SupplierFallbackOff}
}

// EffectiveLinks returns a product's supplier links and the active link's id.
// A product with no SupplierLinks but a legacy mapping (FulfillmentProvider
// set) gets one virtual, active link (id "legacy") so every reader sees the
// same shape before and after the supplierlinks migration.
func EffectiveLinks(p *Product) ([]SupplierLink, string) {
	if len(p.SupplierLinks) > 0 {
		return p.SupplierLinks, p.ActiveSupplierLinkID
	}
	if p.FulfillmentProvider != nil && *p.FulfillmentProvider != 0 {
		return []SupplierLink{{
			ID:         legacyLinkID,
			Provider:   *p.FulfillmentProvider,
			UpstreamID: p.UpstreamProductID,
			Enabled:    true,
			AddedAt:    p.CreatedAt,
		}}, legacyLinkID
	}
	return nil, ""
}

// SupplierCatalog is the narrow port the product module uses to check a
// supplier link against the supplier's live catalog (provider.Probes
// satisfies it). Has reports whether a supplier is configured at all.
type SupplierCatalog interface {
	Has(id int) bool
	Catalog(ctx context.Context, id int) (provider.CatalogSnapshot, error)
}

// WithSupplierCatalog enables the live checks on newly added supplier links.
// Without it (supplier import, tests) links are only shape-validated.
func WithSupplierCatalog(c SupplierCatalog) ServiceOption {
	return func(s *ProductService) { s.catalog = c }
}

// errServiceUnavailable marks a refusal caused by a supplier catalog that
// couldn't be loaded (HTTP 503): retrying shortly can succeed.
var errServiceUnavailable = errors.New("service unavailable")

func linkErr(code, msg string, base error) error {
	return &apperrors.AppError{Code: code, Message: msg, Err: base}
}

// errLinksChanged is returned when the stored links moved on since the editor
// loaded them (stale rev / concurrent save).
func errLinksChanged() error {
	return linkErr("SUPPLIER_LINKS_CHANGED", "someone else changed this product's suppliers — reload the page and try again", apperrors.ErrConflict)
}

// linksPlan is the resolved result of a links update: what to store and the
// legacy mirror that goes with it.
type linksPlan struct {
	Links    []SupplierLink
	ActiveID string
	Rev      int // the new rev to store
	// Mirror of the active link (nil provider / "" upstream = clear).
	MirrorProvider *int
	MirrorUpstream string
}

// planSupplierLinks validates a links table against the product's current
// links (cur nil on create) and resolves ids, the active link and the mirror.
// New or changed rows are checked against the supplier's live catalog when a
// SupplierCatalog is wired; unchanged rows are not re-checked, so a supplier
// outage never blocks an unrelated edit.
func (s *ProductService) planSupplierLinks(ctx context.Context, cur *Product, in *SupplierLinksInput) (*linksPlan, error) {
	var curLinks []SupplierLink
	curRev := 0
	if cur != nil {
		curLinks, _ = EffectiveLinks(cur)
		curRev = cur.SupplierLinksRev
	}
	if in.Rev != curRev {
		return nil, errLinksChanged()
	}
	if len(in.Links) > MaxSupplierLinks {
		return nil, linkErr("LINK_INVALID", fmt.Sprintf("a product can be linked to at most %d supplier products", MaxSupplierLinks), apperrors.ErrBadRequest)
	}

	byID := make(map[string]SupplierLink, len(curLinks))
	known := make(map[string]bool, len(curLinks)) // provider:upstream already linked
	for _, l := range curLinks {
		byID[l.ID] = l
		known[linkKey(l.Provider, l.UpstreamID)] = true
	}

	actor := ""
	if claims, ok := auth.ClaimsFromContext(ctx); ok {
		actor = auth.ActorLabel(claims)
	}
	now := time.Now().UTC()
	seen := map[string]bool{}
	plan := &linksPlan{Links: make([]SupplierLink, 0, len(in.Links)), Rev: curRev + 1}
	activeCount := 0
	var toCheck []SupplierLink
	for i, row := range in.Links {
		up := strings.TrimSpace(row.UpstreamID)
		if row.Provider <= 0 || up == "" {
			return nil, linkErr("LINK_INVALID", fmt.Sprintf("supplier link #%d needs a supplier and a supplier product", i+1), apperrors.ErrBadRequest)
		}
		key := linkKey(row.Provider, up)
		if seen[key] {
			return nil, linkErr("LINK_INVALID", fmt.Sprintf("supplier product %s is linked twice", up), apperrors.ErrBadRequest)
		}
		seen[key] = true

		link := SupplierLink{Provider: row.Provider, UpstreamID: up, Enabled: row.Enabled}
		if old, ok := byID[row.ID]; ok && row.ID != legacyLinkID {
			link.ID, link.AddedAt, link.AddedBy = old.ID, old.AddedAt, old.AddedBy
		} else {
			link.ID = bson.NewObjectID().Hex()
			link.AddedAt, link.AddedBy = now, actor
			if old, ok := byID[row.ID]; ok && row.ID == legacyLinkID {
				link.AddedAt = old.AddedAt // the legacy mapping's age, not today
			}
		}
		if !known[key] {
			toCheck = append(toCheck, link)
		}
		if row.Active {
			activeCount++
			if !row.Enabled {
				return nil, linkErr("LINK_INVALID", "the active supplier link can't be disabled — make another link active first", apperrors.ErrBadRequest)
			}
			plan.ActiveID = link.ID
		}
		plan.Links = append(plan.Links, link)
	}
	switch {
	case len(plan.Links) > 0 && activeCount != 1:
		return nil, linkErr("LINK_INVALID", "choose exactly one active supplier link", apperrors.ErrBadRequest)
	case len(plan.Links) == 0 && activeCount > 0:
		return nil, linkErr("LINK_INVALID", "no supplier link to make active", apperrors.ErrBadRequest)
	}

	if err := s.checkNewLinks(ctx, toCheck); err != nil {
		return nil, err
	}

	for _, l := range plan.Links {
		if l.ID == plan.ActiveID {
			p := l.Provider
			plan.MirrorProvider = &p
			plan.MirrorUpstream = l.UpstreamID
		}
	}
	return plan, nil
}

// checkNewLinks verifies newly added links: the supplier must be configured,
// and (when its adapter lists a catalog) the product id must be in it.
func (s *ProductService) checkNewLinks(ctx context.Context, links []SupplierLink) error {
	if s.catalog == nil {
		return nil
	}
	for _, l := range links {
		if !s.catalog.Has(l.Provider) {
			return linkErr("LINK_UNKNOWN_SUPPLIER", fmt.Sprintf("supplier #%d is not configured", l.Provider), apperrors.ErrBadRequest)
		}
		snap, err := s.catalog.Catalog(ctx, l.Provider)
		switch {
		case errors.Is(err, provider.ErrNotImplemented):
			continue // this supplier can't list its catalog: can't check
		case err != nil:
			return linkErr("LINK_CATALOG_UNAVAILABLE", fmt.Sprintf("couldn't load supplier #%d's product list to check %s — try again in a minute", l.Provider, l.UpstreamID), errServiceUnavailable)
		}
		if _, ok := snap.Find(l.UpstreamID); !ok {
			return linkErr("LINK_UPSTREAM_UNKNOWN", fmt.Sprintf("product %q is not in supplier #%d's catalog — pick it from the supplier search", l.UpstreamID, l.Provider), apperrors.ErrBadRequest)
		}
	}
	return nil
}

// checkLegacySupplierFields refuses an edit of the single legacy supplier
// mapping (fulfillmentProvider / upstreamProductId without supplierLinks) on a
// product that has a links table — e.g. a cached old admin bundle. Re-sending
// the current mirror unchanged is fine.
func checkLegacySupplierFields(cur *Product, in UpdateProductInput) error {
	if len(cur.SupplierLinks) == 0 || in.SupplierLinks != nil {
		return nil
	}
	if in.FulfillmentProvider != nil {
		curP := 0
		if cur.FulfillmentProvider != nil {
			curP = *cur.FulfillmentProvider
		}
		if *in.FulfillmentProvider != curP {
			return errLinksManaged()
		}
	}
	if in.UpstreamProductID != nil && strings.TrimSpace(*in.UpstreamProductID) != cur.UpstreamProductID {
		return errLinksManaged()
	}
	return nil
}

func errLinksManaged() error {
	return linkErr("SUPPLIER_LINKS_MANAGED", "this product's suppliers are managed in the supplier links table — reload the page to edit them", apperrors.ErrConflict)
}

func linkKey(provider int, upstream string) string {
	return fmt.Sprintf("%d:%s", provider, upstream)
}

// linksFromLegacy builds the links table for a product created with only the
// legacy single mapping (supplier catalog import): one enabled, active link.
func linksFromLegacy(providerID *int, upstream string) ([]SupplierLink, string) {
	if providerID == nil || *providerID == 0 || strings.TrimSpace(upstream) == "" {
		return nil, ""
	}
	id := bson.NewObjectID().Hex()
	return []SupplierLink{{ID: id, Provider: *providerID, UpstreamID: strings.TrimSpace(upstream), Enabled: true, AddedAt: time.Now().UTC()}}, id
}
