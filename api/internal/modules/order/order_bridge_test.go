package order

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/bridge"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

// fakeBridge records the last dispatch and can be told it's disabled or to fail.
type fakeBridge struct {
	enabled bool
	err     error
	last    *bridge.DispatchInput
	calls   int
}

func (f *fakeBridge) Enabled() bool { return f.enabled }
func (f *fakeBridge) DispatchOrder(_ context.Context, in bridge.DispatchInput) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	cp := in
	f.last = &cp
	return nil
}

func rechargeLineProduct(price float64) *product.Product {
	return &product.Product{
		ID:              bson.NewObjectID(),
		FulfillmentType: product.FulfillmentCredit,
		FulfillmentMode: product.FulfillmentModeBridgeDevice,
		Bridge:          &product.BridgeSpec{Provider: product.BridgeProviderAlfa, Method: product.BridgeMethodRechargeLine},
		Variants:        []product.Variant{{ID: bson.NewObjectID(), Denomination: "Alfa $5 card", Price: price}},
	}
}

func TestFulfillBridge_TransferCreditDispatches(t *testing.T) {
	p := bridgeProduct(5) // touch, transfer_credit, face value 5
	codeSvc := &fakeCodeSvc{available: map[string]int{}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, codeSvc, walletSvc)
	fb := &fakeBridge{enabled: true}
	svc.bridge = fb

	item := itemFor(p, 1)
	item.PlayerID = "+961 71 123 456"
	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{item},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, OrderStatusProcessing, order.Status)
	require.NotNil(t, fb.last)
	assert.Equal(t, "touch", fb.last.Provider)
	assert.Equal(t, "transfer_credit", fb.last.Method)
	assert.Equal(t, "71123456", fb.last.Phone) // normalized
	require.NotNil(t, fb.last.Amount)
	assert.Equal(t, 5.0, *fb.last.Amount)
	assert.Empty(t, fb.last.CardCode)
	assert.Equal(t, 0, codeSvc.claimCalls) // transfer_credit claims no code
}

func TestFulfillBridge_RechargeLineClaimsCode(t *testing.T) {
	p := rechargeLineProduct(5)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 3}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, codeSvc, walletSvc)
	fb := &fakeBridge{enabled: true}
	svc.bridge = fb

	item := itemFor(p, 1)
	item.PlayerID = "70111222"
	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{item},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, OrderStatusProcessing, order.Status)
	assert.Equal(t, 1, codeSvc.claimCalls)
	require.NotNil(t, fb.last)
	assert.Equal(t, "alfa", fb.last.Provider)
	assert.Equal(t, "recharge_line", fb.last.Method)
	assert.NotEmpty(t, fb.last.CardCode) // the claimed code rode into the command
}

func TestFulfillBridge_DispatchFailureCompensates(t *testing.T) {
	p := rechargeLineProduct(5)
	codeSvc := &fakeCodeSvc{available: map[string]int{p.ID.Hex(): 3}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, codeSvc, walletSvc)
	svc.bridge = &fakeBridge{enabled: true, err: errors.New("queue down")}

	item := itemFor(p, 1)
	item.PlayerID = "70111222"
	_, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{item},
		PaymentMethod: PaymentMethodWallet,
	})
	require.Error(t, err)
	// The claimed code was released and the wallet refunded on failure.
	assert.Len(t, codeSvc.released, 1)
	assert.InDelta(t, 100, walletSvc.balance, 0.001)
}

func TestFulfillBridge_DisabledFallsBackToPark(t *testing.T) {
	p := bridgeProduct(5)
	codeSvc := &fakeCodeSvc{available: map[string]int{}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, _ := newSUT(p, codeSvc, walletSvc)
	svc.bridge = &fakeBridge{enabled: false}

	item := itemFor(p, 1)
	item.PlayerID = "71123456"
	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{item},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)
	assert.Equal(t, OrderStatusProcessing, order.Status)
	assert.True(t, hasTimelineNote(order, "queued for bridge device"))
}

func TestCompleteAndFlagBridgeOrder(t *testing.T) {
	p := bridgeProduct(5)
	codeSvc := &fakeCodeSvc{available: map[string]int{}}
	walletSvc := &fakeWalletSvc{balance: 100}
	svc, repo := newSUT(p, codeSvc, walletSvc)
	svc.bridge = &fakeBridge{enabled: true}

	item := itemFor(p, 1)
	item.PlayerID = "71123456"
	order, err := svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k1", PlaceOrderInput{
		Items:         []PlaceOrderItemInput{item},
		PaymentMethod: PaymentMethodWallet,
	})
	require.NoError(t, err)

	// Flag keeps it processing (manual queue); complete moves it to completed.
	require.NoError(t, svc.FlagBridgeOrder(context.Background(), order.ID, "device rejected"))
	fresh := repo.byID[order.ID]
	assert.Equal(t, OrderStatusProcessing, fresh.Status)
	assert.True(t, hasTimelineNote(order, "device rejected") || hasFlagged(fresh))

	require.NoError(t, svc.CompleteBridgeOrder(context.Background(), order.ID, "bridge:xyz"))
	assert.Equal(t, OrderStatusCompleted, repo.byID[order.ID].Status)
	// Idempotent: completing an already-completed order is a no-op.
	require.NoError(t, svc.CompleteBridgeOrder(context.Background(), order.ID, "bridge:xyz"))
}

func hasFlagged(o *Order) bool {
	for _, e := range o.Fulfillment.StatusTimeline {
		if e.Status == "bridge_failed" {
			return true
		}
	}
	return false
}
