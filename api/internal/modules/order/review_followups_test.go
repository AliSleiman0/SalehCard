package order

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

// A settler pass that read the order BEFORE an admin reroute must not settle
// it against the old supplier afterwards (no refund, no completion, no ref).
func TestSettler_StalePassLosesAfterReroute(t *testing.T) {
	newP := &fakeProvider{id: 9, err: provider.ErrUnavailable} // new attempt still retrying
	st, repo, walletSvc, _ := newRerouteSUT(&fakeProvider{id: 7}, newP)
	o := settlerOrder(repo, func(o *Order) {
		o.Fulfillment.ProviderRef = "UP_OLD"
		stuckAt(o)
	})
	stale := *o // what the settler's list query returned
	stale.Fulfillment = o.Fulfillment

	_, err := st.Reroute(context.Background(), o.ID, RerouteInput{ProviderID: 9, UpstreamProductID: "U-9"}, "a")
	require.NoError(t, err)
	require.Equal(t, 1, o.Fulfillment.SupplierAttempt)

	// The old supplier now says "reject" for the stale snapshot …
	require.NoError(t, st.settleReject(context.Background(), &stale, "rejected upstream — refunded"))
	assert.Equal(t, OrderStatusProcessing, o.Status, "stale reject must not refund the rerouted order")
	assert.Equal(t, 0, walletSvc.refundCalls)

	// … or "accept": must not complete it with the old supplier's codes.
	require.NoError(t, st.settleAccept(context.Background(), &stale, "UP_OLD", []string{"OLD"}, "completed by supplier check"))
	assert.Equal(t, OrderStatusProcessing, o.Status)
	assert.Empty(t, o.Fulfillment.DeliveredCode)

	// A stale re-dispatch result must not stamp a ref onto the new attempt.
	require.NoError(t, st.svc.repo.SetSupplierRef(context.Background(), o.ID, stale.Fulfillment.SupplierAttempt, "UP_STALE", TimelineEvent{}))
	assert.Empty(t, o.Fulfillment.ProviderRef)
}

func TestReroute_ClaimSetsFallbackRetryClock(t *testing.T) {
	// The fake provider "hangs": the dispatch result is never persisted.
	st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, &fakeProvider{id: 9})
	o := settlerOrder(repo, stuckAt)
	claimed, err := st.svc.repo.ClaimSupplierReroute(context.Background(), RerouteClaim{
		OrderID: o.ID, ItemIndex: 0, NewProvider: 9, NewUpstreamID: "U", NewOrderUUID: bson.NewObjectID().Hex(),
	})
	require.NoError(t, err)
	require.NotNil(t, claimed.Fulfillment.SupplierNextRetryAt, "settler must still find the order if dispatch never lands")
	assert.Nil(t, claimed.Fulfillment.SupplierStuckAt)
}

func TestReroute_CancelledOrRefLessPendingRetries(t *testing.T) {
	for name, perr := range map[string]error{
		"context cancelled":            context.Canceled,
		"pending with empty reference": provider.ErrPending,
	} {
		t.Run(name, func(t *testing.T) {
			newP := &fakeProvider{id: 9, err: perr}
			st, repo, walletSvc, _ := newRerouteSUT(&fakeProvider{id: 7}, newP)
			o := settlerOrder(repo, stuckAt)
			got, err := st.Reroute(context.Background(), o.ID, RerouteInput{ProviderID: 9, UpstreamProductID: "U"}, "a")
			require.NoError(t, err)
			assert.Equal(t, OrderStatusProcessing, got.Status)
			assert.Nil(t, got.Fulfillment.SupplierStuckAt, "never treated as a rejection")
			assert.Empty(t, got.Fulfillment.ProviderRef)
			require.NotNil(t, got.Fulfillment.SupplierNextRetryAt)
			assert.Equal(t, 0, walletSvc.refundCalls)
		})
	}
}

func TestLinePlayerID_NeverAPhoto(t *testing.T) {
	p := &product.Product{InputFields: []product.InputField{
		{Key: "qr", Type: product.InputFieldFile},
		{Key: "accountId", Type: product.InputFieldText},
	}}
	photo := "https://acct.blob.core.windows.net/c/order-files/6af0.jpg"

	in := PlaceOrderItemInput{PlayerID: photo, Fields: []OrderFieldInput{{Key: "qr", Value: photo}, {Key: "accountId", Value: " 12345 "}}}
	assert.Equal(t, "12345", linePlayerID(p, in), "first non-photo field becomes the player id")

	in = PlaceOrderItemInput{PlayerID: photo, Fields: []OrderFieldInput{{Key: "qr", Value: photo}}}
	assert.Equal(t, "", linePlayerID(p, in), "no URL as an account id")

	in = PlaceOrderItemInput{PlayerID: "777", Fields: []OrderFieldInput{{Key: "accountId", Value: "777"}}}
	assert.Equal(t, "777", linePlayerID(p, in), "normal ids untouched")
}
