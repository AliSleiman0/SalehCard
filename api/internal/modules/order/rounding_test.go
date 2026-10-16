package order

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/promo"
)

// Catalog prices carry 3 decimals on prod (15.063, 0.909) while every client
// renders cents. The server must charge the displayed (cent-rounded) price so
// no sub-cent dust is left in wallets.

func placeWallet(t *testing.T, svc *OrderService, reseller bool, promoCode string, items ...PlaceOrderItemInput) (*Order, error) {
	t.Helper()
	return svc.PlaceOrder(context.Background(), bson.NewObjectID(), reseller, "", PlaceOrderInput{
		Items:         items,
		PaymentMethod: PaymentMethodWallet,
		PromoCode:     promoCode,
	})
}

func TestPlaceOrder_RoundsThreeDecimalCatalogPrice(t *testing.T) {
	p := creditProduct(15.063)
	walletSvc := &fakeWalletSvc{balance: 20}
	svc, _ := newSUT(p, &fakeCodeSvc{available: map[string]int{}}, walletSvc)

	order, err := placeWallet(t, svc, false, "", itemFor(p, 1))
	require.NoError(t, err)
	assert.Equal(t, 15.06, order.Items[0].Price)
	assert.Equal(t, 15.06, order.Subtotal)
	assert.Equal(t, 15.06, order.Total)
	assert.Equal(t, 15.06, walletSvc.debited, "wallet debit = displayed price")
}

func TestPlaceOrder_RoundsUnitPriceBeforeQty(t *testing.T) {
	// 0.909 displays as $0.91; 3 × $0.91 = $2.73 (not 3 × 0.909 = 2.727).
	p := codeProduct(0.909, nil)
	walletSvc := &fakeWalletSvc{balance: 10}
	svc, _ := newSUT(p, &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}, walletSvc)

	order, err := placeWallet(t, svc, false, "", itemFor(p, 3))
	require.NoError(t, err)
	assert.Equal(t, 0.91, order.Items[0].Price)
	assert.Equal(t, 2.73, order.Total)
	assert.Equal(t, 2.73, walletSvc.debited)
}

func TestPlaceOrder_RoundsThreeDecimalResellerPrice(t *testing.T) {
	override := 7.4449 // per-variant reseller override with sub-cent precision
	p := codeProduct(10, &override)
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}, walletSvc)
	svc.margins = fakeMargins{pct: 0.333} // 10 × (1-0.00333) = 9.9667 — override wins

	order, err := placeWallet(t, svc, true, "", itemFor(p, 2))
	require.NoError(t, err)
	assert.Equal(t, 7.44, order.Items[0].Price)
	assert.Equal(t, 14.88, order.Total)
	assert.Equal(t, 14.88, walletSvc.debited)
}

func TestPlaceOrder_RoundsTierMarginResellerPrice(t *testing.T) {
	p := codeProduct(15.063, nil)
	svc, _ := newSUT(p, &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}, &fakeWalletSvc{balance: 100})
	svc.margins = fakeMargins{pct: 10} // 15.063 × 0.9 = 13.5567 → 13.56

	order, err := placeWallet(t, svc, true, "", itemFor(p, 1))
	require.NoError(t, err)
	assert.Equal(t, 13.56, order.Total)
}

// percentPromo serves one fixed promo for discount-rounding tests.
type percentPromo struct{ pct float64 }

func (f percentPromo) Validate(context.Context, promo.ValidateInput) (*promo.PromoCode, error) {
	return &promo.PromoCode{Type: promo.PromoTypePercent, Value: f.pct}, nil
}
func (f percentPromo) Redeem(context.Context, string) error { return nil }

func TestPlaceOrder_RoundsPromoDiscountAndTotal(t *testing.T) {
	// 15.06 × 15% = 2.259 → discount 2.26, total 12.80; debit equals total.
	p := creditProduct(15.063)
	walletSvc := &fakeWalletSvc{balance: 20}
	svc, _ := newSUT(p, &fakeCodeSvc{available: map[string]int{}}, walletSvc)
	svc.promo = percentPromo{pct: 15}

	order, err := placeWallet(t, svc, false, "SAVE15", itemFor(p, 1))
	require.NoError(t, err)
	assert.Equal(t, 15.06, order.Subtotal)
	assert.Equal(t, 2.26, order.Discount)
	assert.Equal(t, 12.8, order.Total)
	assert.Equal(t, 12.8, walletSvc.debited)
	assert.Equal(t, order.Total, product.RoundCents(order.Subtotal-order.Discount))
}

func TestPlaceOrder_MultiLineSubtotalHasNoFloatDrift(t *testing.T) {
	a := codeProduct(0.1, nil)
	b := codeProduct(0.2, nil)
	walletSvc := &fakeWalletSvc{balance: 10}
	svc, _ := newSUTMulti([]*product.Product{a, b},
		&fakeCodeSvc{available: map[string]int{a.ID.Hex(): 5, b.ID.Hex(): 5}}, walletSvc)

	order, err := placeWallet(t, svc, false, "", itemFor(a, 1), itemFor(b, 1))
	require.NoError(t, err)
	assert.Equal(t, 0.3, order.Total) // not 0.30000000000000004
}
