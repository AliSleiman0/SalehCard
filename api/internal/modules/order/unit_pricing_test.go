package order

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/offer"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

// unitProduct is a live-app coin product: blockPrice buys blockUnits coins, and
// the customer types an amount in [lo, hi] into the "amount" field.
func unitProduct(blockPrice, blockUnits, lo, hi float64) *product.Product {
	return &product.Product{
		ID:              bson.NewObjectID(),
		FulfillmentType: product.FulfillmentCredit,
		FulfillmentMode: product.FulfillmentModeManualOperator,
		Variants:        []product.Variant{{ID: bson.NewObjectID(), Denomination: "per 90,000", Price: blockPrice}},
		InputFields: []product.InputField{
			{Key: "id", Type: product.InputFieldText, Label: product.I18nLabel{En: "ID"}},
			{Key: "amount", Type: product.InputFieldAmount, Label: product.I18nLabel{En: "Enter Amount"},
				Constraints: &product.InputFieldConstraints{Min: &lo, Max: &hi}},
		},
		UnitPricing: &product.UnitPricing{FieldKey: "amount", BlockUnits: blockUnits},
	}
}

func amountItem(p *product.Product, amount float64) PlaceOrderItemInput {
	it := itemFor(p, 1)
	it.Amount = &amount
	it.Fields = []OrderFieldInput{{Key: "id", Value: "user123"}, {Key: "amount", Value: "999999"}}
	return it
}

func placeUnit(t *testing.T, svc *OrderService, reseller bool, it PlaceOrderItemInput) (*Order, error) {
	t.Helper()
	return svc.PlaceOrder(context.Background(), bson.NewObjectID(), reseller, "", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{it},
		PaymentMethod: PaymentMethodWallet,
	})
}

func TestPlaceOrder_UnitPricing_ProportionalPrice(t *testing.T) {
	p := unitProduct(10, 90000, 9000, 900000)
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, &fakeCodeSvc{available: map[string]int{}}, walletSvc)

	order, err := placeUnit(t, svc, false, amountItem(p, 45000))
	require.NoError(t, err)
	assert.Equal(t, 5.0, order.Total)
	assert.Equal(t, 95.0, walletSvc.balance)
	it := order.Items[0]
	assert.Equal(t, 1, it.Qty)
	assert.Equal(t, 5.0, it.Price)
	require.NotNil(t, it.Amount)
	assert.Equal(t, 45000.0, *it.Amount)
	assert.Equal(t, "45,000", it.Denomination)
	// The driving field echoes the priced amount, never the client-typed value.
	var got string
	for _, f := range it.Fields {
		if f.Key == "amount" {
			got = f.Value
		}
	}
	assert.Equal(t, "45000", got)
}

func TestPlaceOrder_UnitPricing_RoundsToCents(t *testing.T) {
	p := unitProduct(10, 90000, 1, 900000)
	svc, _ := newSUT(p, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{balance: 100})

	order, err := placeUnit(t, svc, false, amountItem(p, 10000)) // 1.1111… → 1.11
	require.NoError(t, err)
	assert.Equal(t, 1.11, order.Total)
}

func TestPlaceOrder_UnitPricing_ResellerBlockPrice(t *testing.T) {
	p := unitProduct(10, 90000, 9000, 900000)
	rp := 9.7
	p.Variants[0].ResellerPrice = &rp
	svc, _ := newSUT(p, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{balance: 100})

	order, err := placeUnit(t, svc, true, amountItem(p, 90000))
	require.NoError(t, err)
	assert.Equal(t, 9.7, order.Total)
}

func TestPlaceOrder_UnitPricing_LiveOffer(t *testing.T) {
	p := unitProduct(10, 90000, 9000, 900000)
	off := &offer.Offer{ProductID: p.ID, DiscountType: offer.DiscountPercent, DiscountValue: 20, Active: true}
	offerSvc := &fakeOfferSvc{byProduct: map[bson.ObjectID]*offer.Offer{p.ID: off}}
	svc, _ := newSUTWithOffers([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{balance: 100}, offerSvc)

	order, err := placeUnit(t, svc, false, amountItem(p, 45000))
	require.NoError(t, err)
	assert.Equal(t, 4.0, order.Total) // 10 → 8 per 90,000, half of it
}

// A fixed-amount offer is ignored on a per-unit product: applied to the block
// price it would come off every block of the typed amount (here $0.50 × 5,000
// blocks), and the catalog doesn't advertise it either.
func TestPlaceOrder_UnitPricing_FixedOfferIgnored(t *testing.T) {
	p := unitProduct(1, 100, 100, 1000000)
	off := &offer.Offer{ProductID: p.ID, DiscountType: offer.DiscountFixed, DiscountValue: 0.5, Active: true}
	offerSvc := &fakeOfferSvc{byProduct: map[bson.ObjectID]*offer.Offer{p.ID: off}}
	svc, _ := newSUTWithOffers([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{balance: 10000}, offerSvc)

	order, err := placeUnit(t, svc, false, amountItem(p, 500000))
	require.NoError(t, err)
	assert.Equal(t, 5000.0, order.Total) // full price, not $2,500
}

// A fixed offer still applies to a normal (fixed-package) product.
func TestPlaceOrder_FixedOfferStillAppliesToFixedProduct(t *testing.T) {
	p := creditProduct(10)
	off := &offer.Offer{ProductID: p.ID, DiscountType: offer.DiscountFixed, DiscountValue: 3, Active: true}
	offerSvc := &fakeOfferSvc{byProduct: map[bson.ObjectID]*offer.Offer{p.ID: off}}
	svc, _ := newSUTWithOffers([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{balance: 100}, offerSvc)

	order, err := placeUnit(t, svc, false, itemFor(p, 1))
	require.NoError(t, err)
	assert.Equal(t, 7.0, order.Total)
}

func TestPlaceOrder_UnitPricing_Rejections(t *testing.T) {
	cases := map[string]func(p *product.Product) PlaceOrderItemInput{
		"below min":      func(p *product.Product) PlaceOrderItemInput { return amountItem(p, 8999) },
		"above max":      func(p *product.Product) PlaceOrderItemInput { return amountItem(p, 900001) },
		"missing amount": func(p *product.Product) PlaceOrderItemInput { return itemFor(p, 1) },
		"zero amount":    func(p *product.Product) PlaceOrderItemInput { return amountItem(p, 0) },
		"fractional":     func(p *product.Product) PlaceOrderItemInput { return amountItem(p, 45000.5) },
		"qty above one": func(p *product.Product) PlaceOrderItemInput {
			it := amountItem(p, 45000)
			it.Qty = 2
			return it
		},
	}
	for name, mk := range cases {
		t.Run(name, func(t *testing.T) {
			p := unitProduct(10, 90000, 9000, 900000)
			walletSvc := &fakeWalletSvc{balance: 100}
			svc, repo := newSUT(p, &fakeCodeSvc{available: map[string]int{}}, walletSvc)
			_, err := placeUnit(t, svc, false, mk(p))
			require.Error(t, err)
			assert.Equal(t, 100.0, walletSvc.balance)
			assert.Empty(t, repo.byID)
		})
	}
}

func TestPlaceOrder_UnitPricing_StaleConfigRefused(t *testing.T) {
	p := unitProduct(10, 90000, 9000, 900000)
	p.InputFields = p.InputFields[:1] // driving field removed after configuration
	svc, _ := newSUT(p, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{balance: 100})

	_, err := placeUnit(t, svc, false, amountItem(p, 45000))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestPlaceOrder_AmountOnFixedProductRejected(t *testing.T) {
	p := creditProduct(10)
	svc, _ := newSUT(p, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{balance: 100})

	it := itemFor(p, 1)
	a := 5.0
	it.Amount = &a
	_, err := placeUnit(t, svc, false, it)
	require.Error(t, err)
}

func TestFormatUnits(t *testing.T) {
	assert.Equal(t, "100", formatUnits(100))
	assert.Equal(t, "45,000", formatUnits(45000))
	assert.Equal(t, "1,234,567", formatUnits(1234567))
	assert.Equal(t, "1,000.5", formatUnits(1000.5))
}
