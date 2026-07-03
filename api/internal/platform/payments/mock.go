package payments

import "context"

// MockProvider is a sandbox gateway: it approves every charge and returns a
// synthetic transaction id. It lets the card/usdt order path be exercised
// end-to-end before a real gateway (Stripe / a local Lebanese processor / a
// crypto processor) is integrated — swap it for a real PaymentProvider adapter
// and register that in New.
type MockProvider struct{}

// NewMockProvider constructs the sandbox provider.
func NewMockProvider() *MockProvider { return &MockProvider{} }

// ProcessPayment approves the charge and returns a synthetic transaction id
// derived from the order reference.
func (MockProvider) ProcessPayment(_ context.Context, _ float64, _, ref string) (string, error) {
	return "mock_txn_" + ref, nil
}

// RefundPayment is a no-op for the sandbox provider.
func (MockProvider) RefundPayment(_ context.Context, _ string) error { return nil }

// Name identifies the provider.
func (MockProvider) Name() string { return "mock" }
