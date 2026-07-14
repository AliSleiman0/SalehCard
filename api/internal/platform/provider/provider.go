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
