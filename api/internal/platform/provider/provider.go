// Package provider is the upstream-fulfillment abstraction (migration spec §5):
// the system calls upstream APIs to fulfill api-mode orders without hard-coding
// vendors. This is DISTINCT from payments.PaymentProvider, which charges money;
// a Provider here delivers the purchased good (tops up a player id, etc.).
//
// Real adapters are wired when the owner supplies credentials/provider names; in
// the meantime every id resolves to a StubProvider that reports ErrNotImplemented
// so api-mode orders park cleanly (the order engine's §5 seam).
package provider

import (
	"context"
	"errors"
)

// ErrNotImplemented is returned by adapters that have no real upstream wiring
// yet. The order engine treats it as "park this order", not "fail it".
var ErrNotImplemented = errors.New("provider: not implemented")

// ErrPending is returned (wrapped) by Fulfill when the upstream ACCEPTED the
// order but has not finished it (panel status "wait"). Result.Reference is set;
// the order engine persists it and parks the order for the supplier settler to
// reconcile via CheckStatus.
var ErrPending = errors.New("provider: order pending upstream")

// Probe error sentinels categorize a Cataloger.Profile/ListProducts failure for
// the admin /suppliers health signal (DESIGN-SUPPLIERS.md Phase 3). They are
// SEPARATE from the fulfillment error contract below — a health probe never
// touches an order, so it has its own vocabulary. Any other probe failure
// (transport, decode, rate limit, unknown) falls through to ErrUnavailable →
// the UI shows "unreachable". Use HealthFromError to map an error to a label.
var (
	// ErrProbeAuth: bad/missing credentials or store access denied (panel
	// codes 120-122; umanage 401/403).
	ErrProbeAuth = errors.New("provider: supplier authentication/permission error")
	// ErrProbeIPBlocked: the caller IP is not allowlisted (panel code 123).
	ErrProbeIPBlocked = errors.New("provider: supplier IP not allowed")
	// ErrProbeMaintenance: the supplier is under maintenance (panel code 130).
	ErrProbeMaintenance = errors.New("provider: supplier under maintenance")
)

// HealthFromError maps a Cataloger probe error (or nil) to the /suppliers health
// label. A nil error is "ok" — the caller layers a "low_balance" check on top.
func HealthFromError(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrProbeIPBlocked):
		return "ip_blocked"
	case errors.Is(err, ErrProbeAuth):
		return "auth_error"
	case errors.Is(err, ErrProbeMaintenance):
		return "maintenance"
	case errors.Is(err, ErrNotImplemented):
		return "not_probed"
	default:
		return "unreachable"
	}
}

// ErrUnavailable is returned (wrapped) on environmental failures where the
// upstream did NOT accept an order and no reference exists: supplier balance
// empty, throttling, auth/IP-allowlist/maintenance errors, product withdrawn
// upstream, or transport/decode ambiguity. The order engine parks the order
// (never compensates — the customer has paid and we cannot prove the upstream
// didn't take the order); a retry with the same OrderUUID is safe because the
// upstream dedupes on it.
var ErrUnavailable = errors.New("provider: upstream unavailable")

// FulfillInput carries the data an adapter needs to fulfill one order line.
// Sensitive inputs (spec §3.2) are passed through to the upstream call and must
// never be persisted or logged by an adapter.
type FulfillInput struct {
	ProductID string // our product id — logging/tracing only, never sent upstream
	// UpstreamID is the supplier's own product id (Product.UpstreamProductID
	// snapshot). Empty on a mis-mapped product — adapters report ErrUnavailable.
	UpstreamID string
	PlayerID   string
	// Fields holds every captured input-field value by key. Keys must match the
	// upstream's parameter names; values may be sensitive — never log them.
	Fields map[string]string
	Qty    int
	// OrderUUID is our order id, passed upstream as its idempotency key
	// (panel order_uuid) so a retried dispatch can never double-charge.
	OrderUUID string
}

// Result is the outcome of a successful Fulfill.
type Result struct {
	Reference string            // upstream transaction id / confirmation reference
	Codes     []string          // delivered code payload (panel replay_api), when the good is a code
	Meta      map[string]string // optional adapter-specific detail
}

// Status is the reconciled state of a previously-pending upstream order,
// returned by CheckStatus. State is the upstream's vocabulary:
// "accept" (done — Codes carries any delivered payload), "reject", or "wait".
type Status struct {
	State string
	Codes []string
}

// AccountInfo is returned by Verify — the resolved account username for the
// check_name hook (spec §3.1).
type AccountInfo struct {
	Username string
}

// Account is a supplier's own-account snapshot (a balance probe), returned by
// Cataloger.Profile for the admin /suppliers page (DESIGN-SUPPLIERS.md Phase 3).
// Balance is denominated in the supplier's own Currency (panels: USD;
// umanage: LBP) — SalehCard does not FX-convert it.
type Account struct {
	Balance  float64
	Currency string
	Email    string
}

// CatalogProduct is one upstream product as listed by Cataloger.ListProducts,
// normalized across the different supplier shapes so the admin import/sync UI
// can treat every supplier uniformly. Quantity constraints mirror the store's
// own input-field shapes (PR #95): a discrete QtyValues list, a {QtyMin,QtyMax}
// range, or neither (single-quantity).
type CatalogProduct struct {
	UpstreamID  string // supplier product id (umanage carries a family prefix)
	Name        string
	Category    string // group/category name (tree grouping in the browse UI)
	ParentID    string // upstream parent/category id ("" or "0" = root)
	Price       float64
	BasePrice   float64
	Currency    string
	Available   bool
	Params      []string // upstream input-field prompts (→ Product.InputFields on import)
	ProductType string   // "amount"/"package"/"bundle"/"credit"/"gift"/"voucher"/"recharge"
	QtyMin      *int
	QtyMax      *int
	QtyValues   []string
}

// Cataloger is an OPTIONAL supplemental port a Provider may implement to expose
// its account balance and product catalog to the admin /suppliers page
// (DESIGN-SUPPLIERS.md Phase 3). It is reached via an interface assertion on the
// resolved Provider (prov, ok := prov.(Cataloger)), keeping the core
// fulfillment Provider port slim — the stub/reference adapters do not implement
// it, so a supplier with no live catalog reports "unavailable" in the UI.
type Cataloger interface {
	// Profile probes the supplier's own account (balance/health signal).
	Profile(ctx context.Context) (Account, error)
	// ListProducts returns the supplier's full product catalog for browse/import.
	ListProducts(ctx context.Context) ([]CatalogProduct, error)
}

// Provider is a thin per-vendor adapter. Routing: an api-mode product with
// FulfillmentProvider == N dispatches to the adapter registered under id N.
type Provider interface {
	// ID is the numeric upstream-provider id this adapter handles.
	ID() int
	// Fulfill performs the upstream delivery for one order line.
	Fulfill(ctx context.Context, in FulfillInput) (Result, error)
	// CheckStatus reconciles a previously-pending order by its upstream
	// reference (Result.Reference from a Fulfill that returned ErrPending).
	// Polled by the supplier settler (design Phase 2).
	CheckStatus(ctx context.Context, ref string) (Status, error)
	// Verify resolves the target account/username before purchase (check_name).
	// Not wired into the order flow yet — products carry no verification config.
	Verify(ctx context.Context, in FulfillInput) (AccountInfo, error)
}
