package order

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/payment"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// fakeUSDT is the order module's view of the payment service (usdtIntents port).
type fakeUSDT struct {
	enabled      bool
	whishEnabled bool
	createErr    error
	intents      map[bson.ObjectID]*payment.Intent // keyed by orderID
	created      []bson.ObjectID
}

func (f *fakeUSDT) Enabled() bool      { return f.enabled }
func (f *fakeUSDT) WhishEnabled() bool { return f.whishEnabled }

// CreateWhishOrderIntent shares the fake's create path (the order service treats
// both providers identically — create async intent, return pending order).
func (f *fakeUSDT) CreateWhishOrderIntent(ctx context.Context, userID, orderID bson.ObjectID, amountUSD float64) (*payment.Intent, error) {
	return f.CreateOrderIntent(ctx, userID, orderID, amountUSD)
}

func (f *fakeUSDT) CreateOrderIntent(_ context.Context, userID, orderID bson.ObjectID, amountUSD float64) (*payment.Intent, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	oid := orderID
	in := &payment.Intent{
		ID:                   bson.NewObjectID(),
		UserID:               userID,
		Purpose:              payment.PurposeOrder,
		OrderID:              &oid,
		Network:              payment.NetworkTRC20,
		Address:              "TDepositAddrForOrder0000000000000",
		AmountExpectedMicros: payment.USDToMicros(amountUSD),
		Status:               payment.StatusPending,
	}
	if f.intents == nil {
		f.intents = map[bson.ObjectID]*payment.Intent{}
	}
	f.intents[orderID] = in
	f.created = append(f.created, orderID)
	return in, nil
}

func (f *fakeUSDT) GetActiveOrderIntent(_ context.Context, orderID bson.ObjectID) (*payment.Intent, error) {
	if in, ok := f.intents[orderID]; ok {
		return in, nil
	}
	return nil, apperrors.ErrNotFound
}

// findOnly returns the single order in the repo (helper for cases where
// PlaceOrder returned nil after marking the order failed).
func (r *fakeOrderRepo) findOnly(t *testing.T) *Order {
	t.Helper()
	require.Len(t, r.byID, 1)
	for _, o := range r.byID {
		return o
	}
	return nil
}

func TestPlaceOrder_USDT_CreatesIntentAndReturnsPending(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, _ := newSUT(p, codeSvc, &fakeWalletSvc{balance: 0})
	fu := &fakeUSDT{enabled: true}
	svc.usdt = fu

	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{itemFor(p, 1)},
		PaymentMethod: PaymentMethodUSDT,
	})
	require.NoError(t, err)
	assert.Equal(t, OrderStatusPending, order.Status)
	require.Len(t, fu.created, 1)
	assert.Equal(t, order.ID, fu.created[0])
	// No stock is reserved while awaiting payment (the no-reservation decision).
	assert.Equal(t, 5, codeSvc.available[p.ID.Hex()])
	assert.Zero(t, codeSvc.claimCalls)
}

func TestPlaceOrder_USDT_RejectedWhenDisabled(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}

	// (a) usdt port present but disabled.
	svc, _ := newSUT(p, codeSvc, &fakeWalletSvc{balance: 0})
	svc.usdt = &fakeUSDT{enabled: false}
	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items: []PlaceOrderItemInput{itemFor(p, 1)}, PaymentMethod: PaymentMethodUSDT,
	})
	assertUnavailable(t, err)

	// (b) no usdt port wired at all (default newSUT passes nil).
	svc2, _ := newSUT(p, codeSvc, &fakeWalletSvc{balance: 0})
	_, err = svc2.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k2", PlaceOrderInput{
		Items: []PlaceOrderItemInput{itemFor(p, 1)}, PaymentMethod: PaymentMethodUSDT,
	})
	assertUnavailable(t, err)
}

func assertUnavailable(t *testing.T, err error) {
	t.Helper()
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "PAYMENT_METHOD_UNAVAILABLE", appErr.Code)
}

func TestPlaceOrder_USDT_IntentCreateFailureMarksOrderFailed(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, repo := newSUT(p, codeSvc, &fakeWalletSvc{balance: 0})
	svc.usdt = &fakeUSDT{enabled: true, createErr: errors.New("chain provider down")}

	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items: []PlaceOrderItemInput{itemFor(p, 1)}, PaymentMethod: PaymentMethodUSDT,
	})
	require.Error(t, err)
	assert.Equal(t, OrderStatusFailed, repo.findOnly(t).Status)
}

// placeUSDTOrder is the common Phase-2 setup: a pending usdt order + its fake.
func placeUSDTOrder(t *testing.T, codeSvc *fakeCodeSvc, p *product.Product) (*OrderService, *fakeOrderRepo, *Order) {
	t.Helper()
	svc, repo := newSUT(p, codeSvc, &fakeWalletSvc{balance: 0})
	svc.usdt = &fakeUSDT{enabled: true}
	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items: []PlaceOrderItemInput{itemFor(p, 1)}, PaymentMethod: PaymentMethodUSDT,
	})
	require.NoError(t, err)
	require.Equal(t, OrderStatusPending, order.Status)
	return svc, repo, order
}

func TestFulfillPaidOrder_HappyPath(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, repo, order := placeUSDTOrder(t, codeSvc, p)

	err := svc.FulfillPaidOrder(context.Background(), order.ID, 10, "tx-hash-1")
	require.NoError(t, err)

	o := repo.byID[order.ID]
	assert.Equal(t, OrderStatusCompleted, o.Status)
	assert.NotEmpty(t, o.Fulfillment.DeliveredCode)
	assert.Equal(t, 1, codeSvc.claimCalls)
	assert.Equal(t, 4, codeSvc.available[p.ID.Hex()])
}

func TestFulfillPaidOrder_StockoutCreditsWallet(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, repo, order := placeUSDTOrder(t, codeSvc, p)

	// Stock depletes between placement and payment confirmation.
	codeSvc.failClaim = true

	err := svc.FulfillPaidOrder(context.Background(), order.ID, 10, "tx-hash-1")
	// The order can't be fulfilled → the payment module is told to credit the
	// wallet instead of retrying.
	require.ErrorIs(t, err, payment.ErrOrderNotPayable)
	assert.Equal(t, OrderStatusFailed, repo.byID[order.ID].Status)
	assert.Contains(t, codeSvc.released, order.ID.Hex())
}

func TestFulfillPaidOrder_Idempotent(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, repo, order := placeUSDTOrder(t, codeSvc, p)

	require.NoError(t, svc.FulfillPaidOrder(context.Background(), order.ID, 10, "tx-1"))
	require.Equal(t, OrderStatusCompleted, repo.byID[order.ID].Status)

	// A watcher retry on an already-completed order is a no-op (no second claim).
	err := svc.FulfillPaidOrder(context.Background(), order.ID, 10, "tx-1")
	require.NoError(t, err)
	assert.Equal(t, 1, codeSvc.claimCalls, "order was re-fulfilled on retry")
}

func TestFulfillPaidOrder_AfterExpiryFailsNotPayable(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, _, order := placeUSDTOrder(t, codeSvc, p)

	// Expiry sweep failed the order first.
	require.NoError(t, svc.FailUnpaidOrder(context.Background(), order.ID, "payment expired"))

	// A late payment then tries to fulfill → not payable → credit the wallet.
	err := svc.FulfillPaidOrder(context.Background(), order.ID, 10, "tx-late")
	assert.ErrorIs(t, err, payment.ErrOrderNotPayable)
	assert.Zero(t, codeSvc.claimCalls, "a failed order must not claim stock")
}

func TestFailUnpaidOrder_Idempotent(t *testing.T) {
	p := codeProduct(10, nil)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 5}}
	svc, repo, order := placeUSDTOrder(t, codeSvc, p)

	require.NoError(t, svc.FailUnpaidOrder(context.Background(), order.ID, "payment expired"))
	assert.Equal(t, OrderStatusFailed, repo.byID[order.ID].Status)

	// Second call (e.g. a duplicate sweep) is a no-op, not an error.
	require.NoError(t, svc.FailUnpaidOrder(context.Background(), order.ID, "payment expired"))
	// An unknown order is likewise a no-op.
	require.NoError(t, svc.FailUnpaidOrder(context.Background(), bson.NewObjectID(), "x"))
}
