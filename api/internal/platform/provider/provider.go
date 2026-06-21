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

// FulfillInput carries the data an adapter needs to fulfill one order line.
// Sensitive inputs (spec §3.2) are passed through to the upstream call and must
// never be persisted or logged by an adapter.
type FulfillInput struct {
	ProductID string
	PlayerID  string
	Qty       int
}

// Result is the outcome of a successful Fulfill.
type Result struct {
	Reference string            // upstream transaction id / confirmation reference
	Meta      map[string]string // optional adapter-specific detail
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
	// Verify resolves the target account/username before purchase (check_name).
	// Not wired into the order flow yet — products carry no verification config.
	Verify(ctx context.Context, in FulfillInput) (AccountInfo, error)
}
