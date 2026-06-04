package payments

import "context"

// PaymentProvider defines the interface that all payment backends must satisfy.
type PaymentProvider interface {
	// ProcessPayment initiates a payment and returns a transaction ID.
	ProcessPayment(ctx context.Context, amount float64, currency, ref string) (string, error)

	// RefundPayment reverses a previously processed transaction.
	RefundPayment(ctx context.Context, transactionID string) error

	// Name returns the identifier of the payment provider.
	Name() string
}
