package tron

import (
	"context"
	"time"
)

// Payment is a confirmed inbound USDT transfer observed on-chain.
type Payment struct {
	TxHash       string
	From         string
	AmountMicros int64 // USDT has 6 decimals; 1 USDT = 1_000_000 micros
	BlockTime    time.Time
}

// Watch describes one deposit address the watcher is monitoring.
type Watch struct {
	// Address is the TRON base58 deposit address (unique per payment intent).
	Address string
	// CreatedAt is the intent creation time; transfers before it are ignored
	// (a fresh derived address should have no history, this is a safety belt).
	CreatedAt time.Time
	// ExpectedMicros is the amount the intent asks for. Adapters may use it as
	// a hint (the stub pays exactly this); it is NOT a match filter — partial
	// and over-payments must still be reported.
	ExpectedMicros int64
}

// Reader is the chain-detection port: it finds the first confirmed USDT-TRC20
// transfer into w.Address at or after w.CreatedAt. (nil, nil) means no payment
// has been observed yet.
type Reader interface {
	FindPayment(ctx context.Context, w Watch) (*Payment, error)
}
