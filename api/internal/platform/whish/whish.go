// Package whish is the Whish Pay redirect-payment port together with its
// adapters (the real Whish HTTP API, and a dev stub). Whish is a Lebanese
// wallet gateway: the customer is redirected to a hosted "collect" page, pays
// there, and the result comes back both as an unsigned server callback and via
// a re-pollable status endpoint (the callback carries no proof, so the caller
// re-polls GetStatus as the source of truth).
//
// The payment module depends only on the [Provider] port; [New] selects an
// adapter from [Config] so the provider swaps purely by configuration (ports &
// adapters, like platform/tron and platform/sms). The port intentionally does
// NOT reference the payment.Intent type — callers translate their aggregate
// into [InitiateInput] / [StatusQuery] — so this package never imports the
// modules it serves.
package whish

import (
	"context"
	"fmt"
)

// CollectStatus is the normalized payment status across the Whish status API.
type CollectStatus string

// Collect statuses.
const (
	CollectStatusSuccess CollectStatus = "success"
	CollectStatusFailed  CollectStatus = "failed"
	CollectStatusPending CollectStatus = "pending"
)

// InitiateInput is everything a provider needs to open a hosted payment. All
// URLs are fully formed (including any token query params); the provider echoes
// them to the gateway unchanged.
type InitiateInput struct {
	Amount             float64 // in Currency (Whish wants a float)
	Currency           string  // "USD" (SalehCard is USD-only)
	Invoice            string  // human-facing description
	ExternalID         int64   // unique per request; the gateway's lookup key
	SuccessCallbackURL string  // server-to-server GET on success
	FailureCallbackURL string  // server-to-server GET on failure
	SuccessRedirectURL string  // where the browser lands on success
	FailureRedirectURL string  // where the browser lands on failure
}

// InitiateResult is the provider's response. RedirectURL is the hosted page the
// browser must open; ProviderRef is the gateway's opaque reference.
type InitiateResult struct {
	RedirectURL string
	ProviderRef string
}

// StatusQuery identifies a payment to re-poll.
type StatusQuery struct {
	ExternalID int64
	Currency   string
}

// StatusResult is the normalized status-check response.
type StatusResult struct {
	Status     CollectStatus
	PayerPhone string
}

// Provider is the redirect-gateway port each Whish-like adapter implements.
type Provider interface {
	// Initiate opens a hosted payment and returns the redirect URL.
	Initiate(ctx context.Context, in InitiateInput) (InitiateResult, error)
	// GetStatus re-polls the gateway for the ground-truth payment status.
	GetStatus(ctx context.Context, q StatusQuery) (StatusResult, error)
}

// Config selects and configures the active Whish adapter.
type Config struct {
	// Provider is one of "whish" (the real HTTP API) or "stub" (dev — reports
	// a fake collectUrl and an immediate success, so the flow runs end-to-end
	// with no gateway and no public tunnel). Default "stub".
	Provider string
	// Whish holds the HTTP adapter's credentials/endpoint (required for
	// Provider=="whish").
	Whish ClientConfig
	// Stub configures the dev adapter.
	Stub StubConfig
}

// New builds the [Provider] for cfg.Provider. An empty or "stub" provider
// returns the dev [Stub]; "whish" returns the real adapter (erroring if its
// credentials are incomplete); any other value is an error.
func New(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case "", "stub":
		return NewStub(cfg.Stub), nil
	case "whish":
		client, err := NewClient(cfg.Whish)
		if err != nil {
			return nil, err
		}
		return NewAdapter(client), nil
	default:
		return nil, fmt.Errorf("unknown Whish provider %q", cfg.Provider)
	}
}
