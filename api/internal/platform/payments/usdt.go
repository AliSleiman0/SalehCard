package payments

import (
	"context"
	"errors"
)

// USDTProvider is a stub implementation of PaymentProvider for USDT payments.
type USDTProvider struct{}

// NewUSDTProvider returns a new USDTProvider.
func NewUSDTProvider() *USDTProvider {
	return &USDTProvider{}
}

// ProcessPayment stubs USDT payment processing.
func (u *USDTProvider) ProcessPayment(_ context.Context, _ float64, _, _ string) (string, error) {
	return "", errors.New("usdt payment: TODO implement")
}

// RefundPayment stubs USDT payment refunding.
func (u *USDTProvider) RefundPayment(_ context.Context, _ string) error {
	return errors.New("usdt refund: TODO implement")
}

// Name returns the provider identifier.
func (u *USDTProvider) Name() string {
	return "usdt"
}
