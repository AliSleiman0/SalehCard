package order

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

func f64p(v float64) *float64 { return &v }

func TestSupplierQty(t *testing.T) {
	tests := []struct {
		name   string
		it     OrderItem
		want   int
		wantOK bool
	}{
		{"fixed line sends cart qty", OrderItem{Qty: 3}, 3, true},
		{"per-unit line sends the amount", OrderItem{Qty: 1, Amount: f64p(1000)}, 1000, true},
		{"zero cart qty refused", OrderItem{Qty: 0}, 0, false},
		{"zero amount refused", OrderItem{Qty: 1, Amount: f64p(0)}, 0, false},
		{"fractional amount refused", OrderItem{Qty: 1, Amount: f64p(1000.5)}, 0, false},
		{"absurd amount refused", OrderItem{Qty: 1, Amount: f64p(1e12)}, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := supplierQty(tc.it)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// supplierUnitProduct is a per-unit coin product fulfilled by supplier 7
// (upstream "2"): $1.98 buys 1,000 coins, the customer types 1,000–1,000,000.
func supplierUnitProduct() *product.Product {
	p := unitProduct(1.98, 1000, 1000, 1000000)
	pid := 7
	p.FulfillmentMode = product.FulfillmentModeAPI
	p.FulfillmentProvider = &pid
	p.UpstreamProductID = "2"
	return p
}

// The prod SOULCHILL case (2026-11-13): the customer asked for 1,000 coins and
// the supplier was sent qty 1 → "Qty is Small". It must be sent the amount.
func TestPlaceOrder_SupplierUnitPricedSendsAmountAsQty(t *testing.T) {
	p := supplierUnitProduct()
	fake := &fakeProvider{id: 7, res: provider.Result{Reference: "UP1"}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUTWithProviders([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, walletSvc, fake)

	order, err := placeUnit(t, svc, false, amountItem(p, 5000))
	require.NoError(t, err)
	assert.Equal(t, OrderStatusCompleted, order.Status)
	assert.Equal(t, 9.9, order.Total) // 1.98 × 5,000 / 1,000

	require.NotNil(t, fake.lastIn)
	assert.Equal(t, 5000, fake.lastIn.Qty)
	assert.Equal(t, "2", fake.lastIn.UpstreamID)
	assert.Equal(t, "5000", fake.lastIn.Fields["amount"])
}

// The settler's re-dispatch rebuilds the same input, so it sends the amount too.
func TestSettler_RedispatchSendsUnitAmount(t *testing.T) {
	fake := &fakeProvider{id: 7, res: provider.Result{Reference: "UP2"}}
	st, repo, _, _, _, _ := newSettlerSUT(fake, 24*time.Hour)
	due := time.Now().UTC().Add(-time.Second)
	settlerOrder(repo, func(o *Order) {
		o.Items[0].Amount = f64p(25000)
		o.Fulfillment.SupplierNextRetryAt = &due
	})

	st.Tick(context.Background())

	require.NotNil(t, fake.lastIn)
	assert.Equal(t, 25000, fake.lastIn.Qty)
}

// A corrupt stored amount is never dispatched (no qty 0 / truncation): the
// retry flags the order for an admin and the supplier is not called.
func TestSettler_RedispatchUnsendableAmountFlagsStuck(t *testing.T) {
	fake := &fakeProvider{id: 7, res: provider.Result{Reference: "UP3"}}
	st, repo, walletSvc, _, _, _ := newSettlerSUT(fake, 24*time.Hour)
	due := time.Now().UTC().Add(-time.Second)
	o := settlerOrder(repo, func(o *Order) {
		o.Items[0].Amount = f64p(0.5)
		o.Fulfillment.SupplierNextRetryAt = &due
	})

	st.Tick(context.Background())

	assert.Equal(t, 0, fake.calls)
	assert.Equal(t, OrderStatusProcessing, o.Status)
	assert.NotNil(t, o.Fulfillment.SupplierStuckAt)
	assert.Zero(t, walletSvc.refundCalls)
}
