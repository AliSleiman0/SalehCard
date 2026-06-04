package payments

import (
	"context"
	"errors"
)

// CardProvider is a stub implementation of PaymentProvider for card-based payments.
type CardProvider struct{}

// NewCardProvider returns a new CardProvider.
func NewCardProvider() *CardProvider {
	return &CardProvider{}
}

// ProcessPayment stubs card payment processing.
func (c *CardProvider) ProcessPayment(_ context.Context, _ float64, _, _ string) (string, error) {
	return "", errors.New("card payment: TODO implement")
}

// RefundPayment stubs card payment refunding.
func (c *CardProvider) RefundPayment(_ context.Context, _ string) error {
	return errors.New("card refund: TODO implement")
}

// Name returns the provider identifier.
func (c *CardProvider) Name() string {
	return "card"
}
