package whish

import (
	"context"
	"fmt"
)

// StubConfig configures the dev [Stub] adapter. It has no knobs today; the
// struct exists for parity with the real adapter's config and future options.
type StubConfig struct{}

// Stub is the dev/test adapter: Initiate returns a fake collectUrl and
// GetStatus always reports success, so the redirect flow runs end-to-end with
// no gateway and no public callback tunnel. A dev typically pairs it with a
// short WHISH_INTENT_EXPIRY so the reconciliation sweep re-polls this stub and
// settles the intent within seconds (mirroring platform/tron's auto-paying
// stub). NEVER selectable outside development — config.Validate refuses
// Provider=="stub" with real credentials set.
type Stub struct{}

// NewStub constructs the stub adapter.
func NewStub(StubConfig) *Stub { return &Stub{} }

// Initiate returns a deterministic fake collect URL for the external id.
func (s *Stub) Initiate(_ context.Context, in InitiateInput) (InitiateResult, error) {
	url := fmt.Sprintf("https://sandbox.whish.money/stub/collect?externalId=%d", in.ExternalID)
	return InitiateResult{RedirectURL: url, ProviderRef: url}, nil
}

// GetStatus always reports a successful payment from a fixed sandbox phone.
func (s *Stub) GetStatus(_ context.Context, _ StatusQuery) (StatusResult, error) {
	return StatusResult{Status: CollectStatusSuccess, PayerPhone: "96170902894"}, nil
}
