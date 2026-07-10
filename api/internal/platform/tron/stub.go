package tron

import (
	"context"
	"strconv"
	"time"
)

// StubConfig configures the dev [Stub] adapter.
type StubConfig struct {
	// Delay is how long after intent creation the stub "sees" the payment
	// (default 15s) — long enough to exercise the pending → confirmed UI.
	Delay time.Duration
}

// Stub is the dev/test adapter: it reports a confirmed payment of exactly the
// expected amount once Delay has elapsed since the intent was created, letting
// the whole flow run end-to-end with no chain access (like payments.MockProvider).
type Stub struct {
	delay time.Duration
	now   func() time.Time // injectable for tests
}

// NewStub constructs the stub adapter.
func NewStub(cfg StubConfig) *Stub {
	d := cfg.Delay
	if d <= 0 {
		d = 15 * time.Second
	}
	return &Stub{delay: d, now: time.Now}
}

// FindPayment reports nothing until the delay elapses, then a payment of the
// expected amount. The TxHash is deterministic per address so the unique
// (network, txHash) index stays honest across restarts.
func (s *Stub) FindPayment(_ context.Context, w Watch) (*Payment, error) {
	if s.now().Before(w.CreatedAt.Add(s.delay)) {
		return nil, nil
	}
	return &Payment{
		TxHash:       "stub-" + w.Address,
		From:         "TStubSenderAddressDoesNotExist000",
		AmountMicros: w.ExpectedMicros,
		BlockTime:    s.now().UTC(),
	}, nil
}

// ListTransfers fabricates one payment per elapsed hint (shared-address mode).
// The TxHash is keyed on the salted amount — unique per open intent since
// shared-mode amounts are collision-guarded — so the "stub-"+address hash the
// FindPayment path uses can't collide across intents sharing one address.
func (s *Stub) ListTransfers(_ context.Context, _ string, _ time.Time, hints []AmountHint) ([]Payment, error) {
	var out []Payment
	for _, h := range hints {
		if s.now().Before(h.CreatedAt.Add(s.delay)) {
			continue
		}
		out = append(out, Payment{
			TxHash:       "stub-shared-" + strconv.FormatInt(h.AmountMicros, 10),
			From:         "TStubSenderAddressDoesNotExist000",
			AmountMicros: h.AmountMicros,
			BlockTime:    s.now().UTC(),
		})
	}
	return out, nil
}
