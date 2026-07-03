package payments

import (
	"context"
	"testing"
)

func TestRegistry_MockEnablesCardAndUSDT(t *testing.T) {
	r := New(Config{Provider: "mock"})
	if !r.Enabled("card") || !r.Enabled("usdt") {
		t.Fatal("mock provider should enable card + usdt")
	}
	if r.Enabled("wallet") {
		t.Fatal("wallet is not a gateway method")
	}
}

func TestRegistry_DefaultDisablesGateways(t *testing.T) {
	r := New(Config{})
	if r.Enabled("card") || r.Enabled("usdt") {
		t.Fatal("default registry should leave card/usdt disabled (wallet-only)")
	}
}

func TestMockProvider_ChargeAndRefund(t *testing.T) {
	m := NewMockProvider()
	txn, err := m.ProcessPayment(context.Background(), 10, "USD", "order123")
	if err != nil || txn == "" {
		t.Fatalf("charge failed: err=%v txn=%q", err, txn)
	}
	if err := m.RefundPayment(context.Background(), txn); err != nil {
		t.Fatalf("refund failed: %v", err)
	}
}
