package order

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/modules/offer"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/promo"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/payments"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// --- in-memory fakes -------------------------------------------------------

type fakeOrderRepo struct {
	byID map[bson.ObjectID]*Order
}

func newFakeOrderRepo() *fakeOrderRepo { return &fakeOrderRepo{byID: map[bson.ObjectID]*Order{}} }

func (f *fakeOrderRepo) FindByID(_ context.Context, id bson.ObjectID) (*Order, error) {
	if o, ok := f.byID[id]; ok {
		return o, nil
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeOrderRepo) FindByUserID(_ context.Context, userID bson.ObjectID) ([]*Order, error) {
	out := []*Order{}
	for _, o := range f.byID {
		if o.UserID == userID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (f *fakeOrderRepo) FindByIdempotencyKey(_ context.Context, userID bson.ObjectID, key string) (*Order, error) {
	for _, o := range f.byID {
		if o.UserID == userID && o.IdempotencyKey != "" && o.IdempotencyKey == key {
			return o, nil
		}
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeOrderRepo) Create(_ context.Context, o *Order) error {
	if o.IdempotencyKey != "" {
		for _, ex := range f.byID {
			if ex.UserID == o.UserID && ex.IdempotencyKey == o.IdempotencyKey {
				return apperrors.ErrConflict
			}
		}
	}
	if o.ID.IsZero() {
		o.ID = bson.NewObjectID()
	}
	f.byID[o.ID] = o
	return nil
}

func (f *fakeOrderRepo) UpdateStatus(_ context.Context, id bson.ObjectID, status OrderStatus) error {
	if o, ok := f.byID[id]; ok {
		o.Status = status
		return nil
	}
	return apperrors.ErrNotFound
}

func (f *fakeOrderRepo) SetPaymentRef(_ context.Context, id bson.ObjectID, ref string) error {
	if o, ok := f.byID[id]; ok {
		o.PaymentRef = ref
		return nil
	}
	return apperrors.ErrNotFound
}

func (f *fakeOrderRepo) UpdateFulfillment(_ context.Context, id bson.ObjectID, status OrderStatus, fulfillment Fulfillment) error {
	if o, ok := f.byID[id]; ok {
		o.Status = status
		o.Fulfillment = fulfillment
		return nil
	}
	return apperrors.ErrNotFound
}

func (f *fakeOrderRepo) TransitionStatus(_ context.Context, id bson.ObjectID, from []OrderStatus, to OrderStatus, event TimelineEvent, _ bson.D) (*Order, error) {
	o, ok := f.byID[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	if !slices.Contains(from, o.Status) {
		return nil, apperrors.ErrConflict
	}
	before := *o
	o.Status = to
	o.Fulfillment.StatusTimeline = append(o.Fulfillment.StatusTimeline, event)
	return &before, nil
}

func (f *fakeOrderRepo) AppendTimelineEvent(_ context.Context, id bson.ObjectID, event TimelineEvent) error {
	o, ok := f.byID[id]
	if !ok {
		return apperrors.ErrNotFound
	}
	o.Fulfillment.StatusTimeline = append(o.Fulfillment.StatusTimeline, event)
	return nil
}

func (f *fakeOrderRepo) ListAll(_ context.Context, _ OrderFilter, _ pagination.Params) ([]*Order, int64, error) {
	out := []*Order{}
	for _, o := range f.byID {
		out = append(out, o)
	}
	return out, int64(len(out)), nil
}

func (f *fakeOrderRepo) DayStats(_ context.Context, _ time.Time) (int, float64, error) {
	return 0, 0, nil
}

func (f *fakeOrderRepo) CountByStatus(_ context.Context, status OrderStatus) (int64, error) {
	var n int64
	for _, o := range f.byID {
		if o.Status == status {
			n++
		}
	}
	return n, nil
}

func (f *fakeOrderRepo) CountPendingTransfers(_ context.Context) (int64, error) {
	var n int64
	for _, o := range f.byID {
		if o.Status != OrderStatusProcessing {
			continue
		}
		for _, it := range o.Items {
			if it.FulfillmentType == "transfer" {
				n++
				break
			}
		}
	}
	return n, nil
}

func (f *fakeOrderRepo) FulfillmentBreakdown(_ context.Context) (map[string]int, error) {
	return map[string]int{}, nil
}

func (f *fakeOrderRepo) RevenueSeries(_ context.Context, _ string) ([]string, []float64, error) {
	return []string{}, []float64{}, nil
}

func (f *fakeOrderRepo) KpiSparkSeries(_ context.Context) ([]float64, []int, error) {
	return []float64{}, []int{}, nil
}

func (f *fakeOrderRepo) RevenueSummary(_ context.Context) (RevenueSummary, error) {
	return RevenueSummary{}, nil
}

// fakeProductSvc serves a fixed catalog; only FindByID is exercised.
type fakeProductSvc struct {
	byID map[string]*product.Product
}

func (f *fakeProductSvc) FindByID(_ context.Context, id string) (*product.Product, error) {
	if p, ok := f.byID[id]; ok {
		return p, nil
	}
	return nil, apperrors.ErrNotFound
}
func (f *fakeProductSvc) FindAll(context.Context, product.ListFilter, pagination.Params) ([]product.Product, int64, error) {
	return nil, 0, nil
}
func (f *fakeProductSvc) Create(context.Context, product.CreateProductInput) (*product.Product, error) {
	return nil, nil
}
func (f *fakeProductSvc) Update(context.Context, string, product.UpdateProductInput) (*product.Product, error) {
	return nil, nil
}
func (f *fakeProductSvc) Delete(context.Context, string) error                   { return nil }
func (f *fakeProductSvc) Bulk(context.Context, product.BulkInput) (int64, error) { return 0, nil }

// fakeCodeSvc tracks available stock and whether claims should fail.
type fakeCodeSvc struct {
	available  map[string]int
	failClaim  bool
	released   []string
	claimCalls int
}

func (f *fakeCodeSvc) CountAvailable(_ context.Context, productID string) (int, error) {
	return f.available[productID], nil
}
func (f *fakeCodeSvc) ClaimForOrder(_ context.Context, productID, orderID, _ string, qty int) ([]code.Code, error) {
	f.claimCalls++
	if f.failClaim {
		return nil, code.ErrOutOfStock
	}
	if f.available[productID] < qty {
		return nil, code.ErrOutOfStock
	}
	f.available[productID] -= qty
	out := make([]code.Code, qty)
	for i := range out {
		out[i] = code.Code{ProductID: productID, OrderID: orderID, Code: "CODE-" + orderID, Status: code.StatusDelivered}
	}
	return out, nil
}
func (f *fakeCodeSvc) ReleaseForOrder(_ context.Context, orderID string, productIDs []string) error {
	f.released = append(f.released, orderID)
	return nil
}
func (f *fakeCodeSvc) Upload(context.Context, string, code.UploadInput) (*code.UploadResult, error) {
	return nil, nil
}
func (f *fakeCodeSvc) ListCodes(context.Context, string, code.Status, pagination.Params) ([]code.Code, int64, error) {
	return nil, 0, nil
}
func (f *fakeCodeSvc) Lookup(context.Context, string) (*code.CodeAudit, error) { return nil, nil }
func (f *fakeCodeSvc) SetThreshold(context.Context, string, int) (*code.InventoryStats, error) {
	return nil, nil
}
func (f *fakeCodeSvc) Inventory(context.Context) ([]code.InventoryStats, error) { return nil, nil }
func (f *fakeCodeSvc) LowStock(context.Context) ([]code.InventoryStats, error)  { return nil, nil }
func (f *fakeCodeSvc) InventoryPaged(context.Context, pagination.Params, bool, string) ([]code.InventoryStats, code.InventoryTotals, int64, error) {
	return nil, code.InventoryTotals{}, 0, nil
}

// fakeWalletSvc tracks a single balance and records debit/refund calls.
type fakeWalletSvc struct {
	balance     float64
	debited     float64
	refunded    float64
	debitCalls  int
	refundCalls int
}

func (f *fakeWalletSvc) Debit(_ context.Context, _ bson.ObjectID, amount float64, _ string) (*wallet.WalletTransaction, error) {
	f.debitCalls++
	if f.balance < amount {
		return nil, wallet.ErrInsufficientFunds
	}
	f.balance -= amount
	f.debited += amount
	return &wallet.WalletTransaction{Amount: -amount, BalanceAfter: f.balance}, nil
}
func (f *fakeWalletSvc) Refund(_ context.Context, _ bson.ObjectID, amount float64, _ string) (*wallet.WalletTransaction, error) {
	f.refundCalls++
	f.balance += amount
	f.refunded += amount
	return &wallet.WalletTransaction{Amount: amount, BalanceAfter: f.balance}, nil
}
func (f *fakeWalletSvc) TopUp(context.Context, bson.ObjectID, wallet.TopUpInput) (*wallet.WalletTransaction, error) {
	return nil, nil
}
func (f *fakeWalletSvc) GetBalance(context.Context, bson.ObjectID) (float64, error) {
	return f.balance, nil
}
func (f *fakeWalletSvc) ListTransactions(context.Context, bson.ObjectID) ([]*wallet.WalletTransaction, error) {
	return nil, nil
}

// --- fixtures --------------------------------------------------------------

func codeProduct(price float64, resellerPrice *float64) *product.Product {
	return &product.Product{
		ID:              bson.NewObjectID(),
		FulfillmentType: product.FulfillmentCode,
		FulfillmentMode: product.FulfillmentModeInventory,
		Variants:        []product.Variant{{ID: bson.NewObjectID(), Denomination: "1x", Price: price, ResellerPrice: resellerPrice}},
	}
}

func creditProduct(price float64) *product.Product {
	return &product.Product{
		ID:              bson.NewObjectID(),
		FulfillmentType: product.FulfillmentCredit,
		FulfillmentMode: product.FulfillmentModeManualOperator,
		Variants:        []product.Variant{{ID: bson.NewObjectID(), Denomination: "1800 UC", Price: price}},
	}
}

func apiProduct(price float64, providerID *int) *product.Product {
	return &product.Product{
		ID:                  bson.NewObjectID(),
		FulfillmentType:     product.FulfillmentCredit,
		FulfillmentMode:     product.FulfillmentModeAPI,
		FulfillmentProvider: providerID,
		Variants:            []product.Variant{{ID: bson.NewObjectID(), Denomination: "60 UC", Price: price}},
	}
}

func bridgeProduct(price float64) *product.Product {
	face := price
	return &product.Product{
		ID:              bson.NewObjectID(),
		FulfillmentType: product.FulfillmentCredit,
		FulfillmentMode: product.FulfillmentModeBridgeDevice,
		Bridge:          &product.BridgeSpec{Provider: product.BridgeProviderTouch, Method: product.BridgeMethodTransferCredit},
		Variants:        []product.Variant{{ID: bson.NewObjectID(), Denomination: "Touch $5", Price: price, FaceValue: &face}},
	}
}

func transferProduct(price float64) *product.Product {
	return &product.Product{
		ID:              bson.NewObjectID(),
		FulfillmentType: product.FulfillmentTransfer,
		FulfillmentMode: product.FulfillmentModeManualOperator,
		Variants:        []product.Variant{{ID: bson.NewObjectID(), Denomination: "Variable", Price: price}},
	}
}

func newSUT(p *product.Product, codeSvc *fakeCodeSvc, walletSvc *fakeWalletSvc) (*OrderService, *fakeOrderRepo) {
	return newSUTMulti([]*product.Product{p}, codeSvc, walletSvc)
}

func newSUTMulti(prods []*product.Product, codeSvc *fakeCodeSvc, walletSvc *fakeWalletSvc) (*OrderService, *fakeOrderRepo) {
	return newSUTWithOffers(prods, codeSvc, walletSvc, &fakeOfferSvc{})
}

// fakeKycGate approves or rejects everyone (the checkout KYC gate).
type fakeKycGate struct{ approved bool }

func (f *fakeKycGate) IsApproved(_ context.Context, _ bson.ObjectID) (bool, error) {
	return f.approved, nil
}

func newSUTWithOffers(prods []*product.Product, codeSvc *fakeCodeSvc, walletSvc *fakeWalletSvc, offerSvc *fakeOfferSvc) (*OrderService, *fakeOrderRepo) {
	repo := newFakeOrderRepo()
	byID := make(map[string]*product.Product, len(prods))
	for _, p := range prods {
		byID[p.ID.Hex()] = p
	}
	prodSvc := &fakeProductSvc{byID: byID}
	svc := NewOrderService(repo, prodSvc, codeSvc, walletSvc, &fakePromoSvc{}, offerSvc, provider.NewRegistry(), payments.New(payments.Config{}), nil, &fakeKycGate{approved: true}, nil, notification.Nop{}, &fakeAwarder{}, nil)
	return svc, repo
}

// fakeAwarder records loyalty-award calls so completion tests can assert points
// are minted with the final order total.
type fakeAwarder struct {
	calls     int
	lastUser  bson.ObjectID
	lastTotal float64
}

func (f *fakeAwarder) Award(_ context.Context, userID bson.ObjectID, orderTotal float64) {
	f.calls++
	f.lastUser = userID
	f.lastTotal = orderTotal
}

// fakeMargins returns a fixed reseller-tier margin percent for every user, plus
// an optional per-variant custom price map.
type fakeMargins struct {
	pct    float64
	prices map[string]float64
}

func (f fakeMargins) MarginForUser(_ context.Context, _ bson.ObjectID) (float64, error) {
	return f.pct, nil
}

func (f fakeMargins) PricesForUser(_ context.Context, _ bson.ObjectID) (map[string]float64, error) {
	return f.prices, nil
}

// fakeOfferSvc serves at most one live offer per product; an absent product
// yields ErrNotFound (the "no offer" case the order engine treats as full price).
type fakeOfferSvc struct {
	byProduct map[bson.ObjectID]*offer.Offer
}

func (f *fakeOfferSvc) FindLiveByProduct(_ context.Context, productID bson.ObjectID, _ time.Time) (*offer.Offer, error) {
	if o, ok := f.byProduct[productID]; ok {
		return o, nil
	}
	return nil, apperrors.ErrNotFound
}

// fakePromoSvc is a no-op promo service; the order tests never set a promo code,
// so Validate/Redeem are not exercised (a benign zero promo keeps it safe).
type fakePromoSvc struct{}

func (f *fakePromoSvc) Validate(_ context.Context, _ promo.ValidateInput) (*promo.PromoCode, error) {
	return &promo.PromoCode{}, nil
}

func (f *fakePromoSvc) Redeem(_ context.Context, _ string) error { return nil }

func itemFor(p *product.Product, qty int) PlaceOrderItemInput {
	return PlaceOrderItemInput{ProductID: p.ID.Hex(), VariantID: p.Variants[0].ID.Hex(), Qty: qty}
}

// --- tests -----------------------------------------------------------------

func TestPlaceOrder_CodeFulfillment_Success(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, codeSvc, walletSvc)

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 2)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, OrderStatusCompleted, order.Status)
	assert.Equal(t, 20.0, order.Total) // re-priced server-side: 10 * 2
	assert.NotEmpty(t, order.Fulfillment.DeliveredCode)
	assert.Equal(t, 80.0, walletSvc.balance) // 100 - 20
	assert.Equal(t, 3, codeSvc.available[p.ID.Hex()])
}

func TestPlaceOrder_AwardsLoyaltyOnCompletion(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, _ := newSUT(p, codeSvc, &fakeWalletSvc{balance: 100})
	userID := bson.NewObjectID()

	order, err := svc.PlaceOrder(context.Background(), userID, false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 2)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, order.Status)

	aw := svc.loyalty.(*fakeAwarder)
	assert.Equal(t, 1, aw.calls) // awarded exactly once, on completion
	assert.Equal(t, userID, aw.lastUser)
	assert.Equal(t, order.Total, aw.lastTotal) // minted off the final, re-priced total (20.0)
}

func TestPlaceOrder_UsesResellerPriceWhenReseller(t *testing.T) {
	resellerPrice := 7.0
	p := codeProduct(10, &resellerPrice)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, _ := newSUT(p, codeSvc, &fakeWalletSvc{balance: 100})

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), true, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, 7.0, order.Total) // reseller price, not retail 10
}

func TestPlaceOrder_AppliesTierMarginForReseller(t *testing.T) {
	p := codeProduct(10, nil) // no per-variant override
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, _ := newSUT(p, codeSvc, &fakeWalletSvc{balance: 100})
	svc.margins = fakeMargins{pct: 20} // 20% tier margin → 10 * 0.8 = 8

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), true, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, 8.0, order.Total) // tier-margin price, not retail 10
}

func TestPlaceOrder_ResellerPaysLowestOfMarginAndOverride(t *testing.T) {
	override := 7.0
	p := codeProduct(10, &override) // explicit override 7 beats 20%-margin 8
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, _ := newSUT(p, codeSvc, &fakeWalletSvc{balance: 100})
	svc.margins = fakeMargins{pct: 20}

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), true, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, 7.0, order.Total) // min(retail 10, margin 8, override 7)
}

func TestPlaceOrder_CardPaymentViaGateway(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	walletSvc := &fakeWalletSvc{balance: 0} // wallet not used for a card order
	svc, _ := newSUT(p, codeSvc, walletSvc)
	svc.payments = payments.New(payments.Config{Provider: "mock"})

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodCard,
	})
	require.NoError(t, err)
	assert.Equal(t, OrderStatusCompleted, order.Status)
	assert.NotEmpty(t, order.PaymentRef)    // gateway transaction id stored
	assert.Equal(t, 0.0, walletSvc.balance) // wallet untouched
}

func TestPlaceOrder_CardRejectedWhenNoGateway(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, _ := newSUT(p, codeSvc, &fakeWalletSvc{balance: 0}) // default: empty payments registry

	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodCard,
	})
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "PAYMENT_METHOD_UNAVAILABLE", appErr.Code)
}

func TestPlaceOrder_ResellerCustomPriceWins(t *testing.T) {
	p := codeProduct(10, nil) // retail 10, no per-variant override
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, _ := newSUT(p, codeSvc, &fakeWalletSvc{balance: 100})
	// 20% margin → 8, but a per-reseller custom price of 6 for this variant wins.
	svc.margins = fakeMargins{pct: 20, prices: map[string]float64{p.Variants[0].ID.Hex(): 6}}

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), true, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, 6.0, order.Total) // per-reseller custom price, the lowest layer
}

func TestPlaceOrder_AppliesLiveOffer(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	walletSvc := &fakeWalletSvc{balance: 100}
	off := &offer.Offer{ProductID: p.ID, DiscountType: offer.DiscountPercent, DiscountValue: 30, Active: true}
	offerSvc := &fakeOfferSvc{byProduct: map[bson.ObjectID]*offer.Offer{p.ID: off}}
	svc, _ := newSUTWithOffers([]*product.Product{p}, codeSvc, walletSvc, offerSvc)

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 2)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, 14.0, order.Total)         // 10 → 7 (30% off) × 2
	assert.Equal(t, 7.0, order.Items[0].Price) // sale price snapshotted on the line
	assert.Equal(t, 86.0, walletSvc.balance)   // 100 - 14
}

func TestPlaceOrder_OfferNotStackedForReseller(t *testing.T) {
	resellerPrice := 7.0
	p := codeProduct(10, &resellerPrice)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	off := &offer.Offer{ProductID: p.ID, DiscountType: offer.DiscountPercent, DiscountValue: 50, Active: true}
	offerSvc := &fakeOfferSvc{byProduct: map[bson.ObjectID]*offer.Offer{p.ID: off}}
	svc, _ := newSUTWithOffers([]*product.Product{p}, codeSvc, &fakeWalletSvc{balance: 100}, offerSvc)

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), true, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, 7.0, order.Total) // reseller price wins; the 50% offer is not applied
}

func TestPlaceOrder_InsufficientFunds(t *testing.T) {
	p := codeProduct(50, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	walletSvc := &fakeWalletSvc{balance: 10}
	svc, repo := newSUT(p, codeSvc, walletSvc)

	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.ErrorIs(t, err, wallet.ErrInsufficientFunds)
	// Order persisted then marked failed; no code claimed.
	require.Len(t, repo.byID, 1)
	for _, o := range repo.byID {
		assert.Equal(t, OrderStatusFailed, o.Status)
	}
	assert.Equal(t, 0, codeSvc.claimCalls)
	assert.Equal(t, 5, codeSvc.available[p.ID.Hex()])
}

func TestPlaceOrder_OutOfStock_CompensatesRefund(t *testing.T) {
	p := codeProduct(10, nil)
	// Pre-check passes (available high) but the atomic claim fails — exercises
	// the compensation path: release + refund the already-charged wallet.
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 99}, failClaim: true}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, repo := newSUT(p, codeSvc, walletSvc)

	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.ErrorIs(t, err, code.ErrOutOfStock)
	assert.Equal(t, 1, walletSvc.debitCalls)
	assert.Equal(t, 1, walletSvc.refundCalls)
	assert.Equal(t, 100.0, walletSvc.balance) // charged then refunded → unchanged
	assert.Len(t, codeSvc.released, 1)
	for _, o := range repo.byID {
		assert.Equal(t, OrderStatusFailed, o.Status)
	}
}

func TestPlaceOrder_OutOfStock_PreCheckBeforeCharge(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 0}} // empty
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, repo := newSUT(p, codeSvc, walletSvc)

	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.ErrorIs(t, err, code.ErrOutOfStock)
	assert.Equal(t, 0, walletSvc.debitCalls) // never charged
	assert.Len(t, repo.byID, 0)              // no order created
}

func TestPlaceOrder_Idempotent(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, repo := newSUT(p, codeSvc, walletSvc)
	uid := bson.NewObjectID()
	input := PlaceOrderInput{Items: []PlaceOrderItemInput{itemFor(p, 1)}, PaymentMethod: PaymentMethodWallet}

	first, err := svc.PlaceOrder(context.Background(), uid, false, "same-key", input)
	require.NoError(t, err)
	second, err := svc.PlaceOrder(context.Background(), uid, false, "same-key", input)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)     // same order returned
	assert.Len(t, repo.byID, 1)              // only one order
	assert.Equal(t, 90.0, walletSvc.balance) // charged exactly once
	assert.Equal(t, 1, walletSvc.debitCalls)
}

func TestPlaceOrder_CreditGoesProcessing(t *testing.T) {
	p := creditProduct(15)
	codeSvc := &fakeCodeSvc{available: map[string]int{}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, codeSvc, walletSvc)

	item := itemFor(p, 1)
	item.PlayerID = "player-123"
	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{item},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, OrderStatusProcessing, order.Status)
	assert.Equal(t, "player-123", order.Fulfillment.CreditedToID)
	assert.Equal(t, 0, codeSvc.claimCalls) // no codes claimed for credit
}

// TestPlaceOrder_RejectsQtyOutOfRange covers a product whose legacy
// "quantity" input field carries a real minimum-purchase bound — the order
// must be rejected server-side rather than silently accepted at a quantity
// below what the product spec allows.
func TestPlaceOrder_RejectsQtyOutOfRange(t *testing.T) {
	p := creditProduct(10)
	min, max := 30.0, 5000.0
	p.InputFields = []product.InputField{
		{Key: "qty", Label: product.I18nLabel{En: "Enter Quantity"}, Type: product.InputFieldQuantity,
			Constraints: &product.InputFieldConstraints{Min: &min, Max: &max}},
	}
	codeSvc := &fakeCodeSvc{available: map[string]int{}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, repo := newSUT(p, codeSvc, walletSvc)

	item := itemFor(p, 5) // below the product's minimum of 30
	item.PlayerID = "player-1"
	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{item},
		PaymentMethod: PaymentMethodWallet,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrBadRequest))
	assert.Len(t, repo.byID, 0) // nothing persisted for a rejected order
}

// TestPlaceOrder_OverridesQuantityFieldValue is the direct regression test
// for the checkout bug: a customer typing an unrelated value into the legacy
// "Enter Quantity" box must not change what they're charged, and the value
// persisted on the order for that field must reflect the real quantity.
func TestPlaceOrder_OverridesQuantityFieldValue(t *testing.T) {
	p := creditProduct(10)
	p.InputFields = []product.InputField{
		{Key: "id", Label: product.I18nLabel{En: "ID"}, Type: product.InputFieldText},
		{Key: "qty", Label: product.I18nLabel{En: "Enter Quantity"}, Type: product.InputFieldQuantity},
	}
	codeSvc := &fakeCodeSvc{available: map[string]int{}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, codeSvc, walletSvc)

	item := itemFor(p, 3)
	item.PlayerID = "player-1"
	item.Fields = []OrderFieldInput{
		{Key: "id", Value: "28472"},
		{Key: "qty", Value: "999"}, // bogus value typed by the customer
	}
	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{item},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, 30.0, order.Total) // 10 * 3, never affected by the qty field's typed value

	require.Len(t, order.Items, 1)
	var qtyField *OrderField
	for i := range order.Items[0].Fields {
		if order.Items[0].Fields[i].Key == "qty" {
			qtyField = &order.Items[0].Fields[i]
		}
	}
	require.NotNil(t, qtyField, "qty field must be present even though the client value is discarded")
	assert.Equal(t, "3", qtyField.Value)
}

func TestPlaceOrder_RejectsUnverifiedUser(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, repo := newSUT(p, codeSvc, walletSvc)
	svc.kyc = &fakeKycGate{approved: false}

	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.ErrorIs(t, err, ErrKYCRequired)
	assert.Len(t, repo.byID, 0)              // rejected before any order exists
	assert.Equal(t, 0, walletSvc.debitCalls) // never charged
}

func TestPlaceOrder_RejectsCardAndUSDT(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, repo := newSUT(p, codeSvc, &fakeWalletSvc{balance: 100})

	for _, method := range []PaymentMethod{PaymentMethodCard, PaymentMethodUSDT} {
		_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k-"+string(method), PlaceOrderInput{
			Items:         []PlaceOrderItemInput{itemFor(p, 1)},
			PaymentMethod: method,
		})
		require.Error(t, err, string(method))
		assert.True(t, errors.Is(err, apperrors.ErrBadRequest), string(method))
	}
	assert.Len(t, repo.byID, 0) // nothing persisted for rejected methods
}

func TestPlaceOrder_RejectsUnknownVariant(t *testing.T) {
	p := codeProduct(10, nil)
	svc, _ := newSUT(p, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{balance: 100})

	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{{ProductID: p.ID.Hex(), VariantID: bson.NewObjectID().Hex(), Qty: 1}},
		PaymentMethod: PaymentMethodWallet,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrBadRequest))
}

func hasTimelineNote(o *Order, note string) bool {
	for _, e := range o.Fulfillment.StatusTimeline {
		if e.Note == note {
			return true
		}
	}
	return false
}

func TestPlaceOrder_APIModeParks(t *testing.T) {
	pid := 5
	p := apiProduct(2.5, &pid)
	codeSvc := &fakeCodeSvc{available: map[string]int{}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, codeSvc, walletSvc)

	item := itemFor(p, 1)
	item.PlayerID = "player-9"
	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{item},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	// No real provider wired → the stub reports not-implemented → order parks.
	assert.Equal(t, OrderStatusProcessing, order.Status)
	assert.Equal(t, 0, codeSvc.claimCalls)
	assert.True(t, hasTimelineNote(order, "awaiting provider integration"))
}

func TestPlaceOrder_BridgeModeParks(t *testing.T) {
	p := bridgeProduct(5)
	codeSvc := &fakeCodeSvc{available: map[string]int{}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, codeSvc, walletSvc)

	item := itemFor(p, 1)
	item.PlayerID = "71123456"
	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{item},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, OrderStatusProcessing, order.Status)
	assert.Equal(t, 0, codeSvc.claimCalls)
	assert.True(t, hasTimelineNote(order, "queued for bridge device"))
}

func TestPlaceOrder_MixedCartGoesProcessing(t *testing.T) {
	codeP := codeProduct(10, nil)
	transferP := transferProduct(20)
	codeSvc := &fakeCodeSvc{available: map[string]int{codeP.ID.Hex(): 5}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUTMulti([]*product.Product{codeP, transferP}, codeSvc, walletSvc)

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(codeP, 1), itemFor(transferP, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	// A mixed cart parks the whole order; no partial code delivery.
	assert.Equal(t, OrderStatusProcessing, order.Status)
	assert.Equal(t, 0, codeSvc.claimCalls)
}

func TestPlaceOrder_EmptyModeFallsBackToInventory(t *testing.T) {
	p := codeProduct(10, nil)
	p.FulfillmentMode = "" // legacy product persisted before the mode field existed
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, codeSvc, walletSvc)

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	// Read-time DeriveMode fallback still routes type=code to inventory.
	assert.Equal(t, OrderStatusCompleted, order.Status)
	assert.NotEmpty(t, order.Fulfillment.DeliveredCode)
	assert.Equal(t, 1, codeSvc.claimCalls)
}

func TestDeriveMode(t *testing.T) {
	cases := []struct {
		in   product.FulfillmentType
		want product.FulfillmentMode
	}{
		{product.FulfillmentCode, product.FulfillmentModeInventory},
		{product.FulfillmentCredit, product.FulfillmentModeManualOperator},
		{product.FulfillmentTransfer, product.FulfillmentModeManualOperator},
		{product.FulfillmentType("unknown"), product.FulfillmentModeManualOperator},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, product.DeriveMode(c.in))
	}
}

func TestGetOrder_EnforcesOwnership(t *testing.T) {
	p := codeProduct(10, nil)
	svc, repo := newSUT(p, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{})
	owner := bson.NewObjectID()
	o := &Order{UserID: owner}
	require.NoError(t, repo.Create(context.Background(), o))

	_, err := svc.GetOrder(context.Background(), owner, o.ID)
	require.NoError(t, err)

	_, err = svc.GetOrder(context.Background(), bson.NewObjectID(), o.ID) // different user
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

// --- supplier (api-mode) fulfillment -----------------------------------------

// fakeProvider is a scripted upstream adapter: it returns the configured
// result/err from Fulfill and records the FulfillInput it received, so tests
// can drive every fulfillAPI branch and assert the plumbing.
type fakeProvider struct {
	id     int
	res    provider.Result
	err    error
	lastIn *provider.FulfillInput
	calls  int
}

func (f *fakeProvider) ID() int { return f.id }
func (f *fakeProvider) Fulfill(_ context.Context, in provider.FulfillInput) (provider.Result, error) {
	f.calls++
	f.lastIn = &in
	return f.res, f.err
}
func (f *fakeProvider) CheckStatus(context.Context, string) (provider.Status, error) {
	return provider.Status{}, provider.ErrNotImplemented
}
func (f *fakeProvider) Verify(context.Context, provider.FulfillInput) (provider.AccountInfo, error) {
	return provider.AccountInfo{}, provider.ErrNotImplemented
}

// newSUTWithProviders is newSUTWithOffers with real adapters in the registry.
func newSUTWithProviders(prods []*product.Product, codeSvc *fakeCodeSvc, walletSvc *fakeWalletSvc, adapters ...provider.Provider) (*OrderService, *fakeOrderRepo) {
	repo := newFakeOrderRepo()
	byID := make(map[string]*product.Product, len(prods))
	for _, p := range prods {
		byID[p.ID.Hex()] = p
	}
	prodSvc := &fakeProductSvc{byID: byID}
	svc := NewOrderService(repo, prodSvc, codeSvc, walletSvc, &fakePromoSvc{}, &fakeOfferSvc{}, provider.NewRegistry(adapters...), payments.New(payments.Config{}), nil, &fakeKycGate{approved: true}, nil, notification.Nop{}, &fakeAwarder{}, nil)
	return svc, repo
}

// supplierProduct is an api-mode product mapped to a supplier: provider id,
// upstream product id, and two input fields the panel expects as params.
func supplierProduct(price float64, providerID int, upstreamID string) *product.Product {
	p := apiProduct(price, &providerID)
	p.UpstreamProductID = upstreamID
	p.InputFields = []product.InputField{
		{Key: "playerId", Label: product.I18nLabel{En: "Player ID"}, Type: product.InputFieldText},
		{Key: "zoneId", Label: product.I18nLabel{En: "Zone"}, Type: product.InputFieldText},
	}
	return p
}

func supplierItem(p *product.Product, qty int) PlaceOrderItemInput {
	item := itemFor(p, qty)
	item.PlayerID = "player-9"
	item.Fields = []OrderFieldInput{
		{Key: "playerId", Value: "player-9"},
		{Key: "zoneId", Value: "77"},
	}
	return item
}

func TestPlaceOrder_APISuccessCompletes(t *testing.T) {
	p := supplierProduct(2.5, 7, "364")
	fake := &fakeProvider{id: 7, res: provider.Result{Reference: "ID_up1", Codes: []string{"C1", "C2"}}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUTWithProviders([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, walletSvc, fake)
	userID := bson.NewObjectID()

	order, err := svc.PlaceOrder(context.Background(), userID, false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{supplierItem(p, 2)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, OrderStatusCompleted, order.Status)
	assert.Equal(t, "ID_up1", order.Fulfillment.TransferRef)
	assert.Equal(t, "ID_up1", order.Fulfillment.ProviderRef)
	assert.Equal(t, "C1\nC2", order.Fulfillment.DeliveredCode)
	assert.Equal(t, 95.0, walletSvc.balance) // 100 - 2.5*2, never refunded
	aw := svc.loyalty.(*fakeAwarder)
	assert.Equal(t, 1, aw.calls)

	// Input plumbing: the adapter received the snapshot, not client-trusted data.
	require.NotNil(t, fake.lastIn)
	assert.Equal(t, "364", fake.lastIn.UpstreamID)
	assert.Equal(t, p.ID.Hex(), fake.lastIn.ProductID)
	assert.Equal(t, order.ID.Hex(), fake.lastIn.OrderUUID)
	assert.Equal(t, 2, fake.lastIn.Qty)
	assert.Equal(t, "player-9", fake.lastIn.PlayerID)
	assert.Equal(t, map[string]string{"playerId": "player-9", "zoneId": "77"}, fake.lastIn.Fields)
}

func TestPlaceOrder_APIPendingParksWithProviderRef(t *testing.T) {
	p := supplierProduct(2.5, 7, "364")
	fake := &fakeProvider{
		id:  7,
		res: provider.Result{Reference: "ID_wait"},
		err: fmt.Errorf("panel test: order pending: %w", provider.ErrPending),
	}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUTWithProviders([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, walletSvc, fake)

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{supplierItem(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, OrderStatusProcessing, order.Status)
	assert.Equal(t, "ID_wait", order.Fulfillment.ProviderRef) // settler reconciles by this
	assert.True(t, hasTimelineNote(order, "awaiting upstream provider"))
	assert.Equal(t, 97.5, walletSvc.balance) // charged, NOT refunded
	assert.Equal(t, 0, walletSvc.refundCalls)
}

func TestPlaceOrder_APIUnavailableParksWithoutRef(t *testing.T) {
	p := supplierProduct(2.5, 7, "364")
	fake := &fakeProvider{
		id:  7,
		err: fmt.Errorf("panel test: upstream error code 100: %w", provider.ErrUnavailable),
	}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUTWithProviders([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, walletSvc, fake)

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{supplierItem(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, OrderStatusProcessing, order.Status)
	assert.Empty(t, order.Fulfillment.ProviderRef) // no acceptance → no ref
	assert.True(t, hasTimelineNote(order, "upstream temporarily unavailable"))
	assert.Equal(t, 97.5, walletSvc.balance) // charged, NOT refunded
	assert.Equal(t, 0, walletSvc.refundCalls)
}

func TestPlaceOrder_APIHardFailCompensates(t *testing.T) {
	p := supplierProduct(2.5, 7, "364")
	fake := &fakeProvider{id: 7, err: errors.New("panel test: player id blocked (code 107)")}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, repo := newSUTWithProviders([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, walletSvc, fake)

	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{supplierItem(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.Error(t, err)
	assert.Equal(t, 100.0, walletSvc.balance) // debit reversed
	assert.Equal(t, 1, walletSvc.refundCalls)
	for _, o := range repo.byID {
		assert.Equal(t, OrderStatusFailed, o.Status)
	}
}

func TestPlaceOrder_APISingleItemRule(t *testing.T) {
	apiP := supplierProduct(2.5, 7, "364")
	codeP := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{codeP.ID.Hex(): 5}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUTWithProviders([]*product.Product{apiP, codeP}, codeSvc, walletSvc, &fakeProvider{id: 7})

	// An api-mode item mixed with anything else is refused outright (multi-item
	// upstream dispatch doesn't exist; silent partial fulfillment is worse).
	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{supplierItem(apiP, 1), itemFor(codeP, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrBadRequest))
	assert.Equal(t, 100.0, walletSvc.balance) // refused before any charge
}
