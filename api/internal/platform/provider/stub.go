package provider

import "context"

// StubProvider is a placeholder adapter: it resolves (so dispatch never dead-ends)
// but performs no real upstream call, returning ErrNotImplemented from both
// methods. It is what every provider id resolves to until real adapters land.
type StubProvider struct{ id int }

// NewStub returns a StubProvider bound to the given provider id.
func NewStub(id int) *StubProvider { return &StubProvider{id: id} }

// ID returns the provider id this stub stands in for.
func (p *StubProvider) ID() int { return p.id }

// Fulfill always reports ErrNotImplemented so the order engine parks the order.
func (p *StubProvider) Fulfill(_ context.Context, _ FulfillInput) (Result, error) {
	return Result{}, ErrNotImplemented
}

// CheckStatus always reports ErrNotImplemented.
func (p *StubProvider) CheckStatus(_ context.Context, _ string) (Status, error) {
	return Status{}, ErrNotImplemented
}

// Verify always reports ErrNotImplemented.
func (p *StubProvider) Verify(_ context.Context, _ FulfillInput) (AccountInfo, error) {
	return AccountInfo{}, ErrNotImplemented
}
