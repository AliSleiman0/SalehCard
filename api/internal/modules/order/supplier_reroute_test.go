package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/payments"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// newRerouteSUT wires a settler over two scripted suppliers: old (#7, where the
// seeded orders live) and new (#9, the reroute target).
func newRerouteSUT(oldP, newP *fakeProvider) (*SupplierSettler, *fakeOrderRepo, *fakeWalletSvc, *fakeRecorder) {
	repo := newFakeOrderRepo()
	walletSvc := &fakeWalletSvc{}
	svc := NewOrderService(repo, &fakeProductSvc{byID: map[string]*product.Product{}},
		&fakeCodeSvc{available: map[string]int{}}, walletSvc, &fakePromoSvc{}, &fakeOfferSvc{},
		provider.NewRegistry(oldP, newP), payments.New(payments.Config{}), nil,
		&fakeKycGate{approved: true}, nil, &fakeNotifier{}, &fakeAwarder{}, nil)
	rec := &fakeRecorder{}
	return NewSupplierSettler(svc, rec, time.Second, 24*time.Hour), repo, walletSvc, rec
}

func stuckAt(o *Order) {
	now := time.Now().UTC()
	o.Fulfillment.SupplierStuckAt = &now
}

func appCode(err error) string {
	var ae *apperrors.AppError
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func TestReroute_StuckOrderCompletesAtNewSupplier(t *testing.T) {
	oldP := &fakeProvider{id: 7}
	newP := &fakeProvider{id: 9, res: provider.Result{Reference: "NEW_1", Codes: []string{"C9"}}}
	st, repo, walletSvc, rec := newRerouteSUT(oldP, newP)
	o := settlerOrder(repo, stuckAt)

	got, err := st.Reroute(context.Background(), o.ID, RerouteInput{ProviderID: 9, UpstreamProductID: "U-900"}, "admin@x")
	require.NoError(t, err)

	assert.Equal(t, OrderStatusCompleted, got.Status)
	assert.Equal(t, "C9", got.Fulfillment.DeliveredCode)
	require.Equal(t, 1, newP.calls)
	assert.Equal(t, "U-900", newP.lastIn.UpstreamID)
	assert.NotEqual(t, o.ID.Hex(), newP.lastIn.OrderUUID, "a reroute needs a fresh upstream idempotency key")
	assert.Len(t, newP.lastIn.OrderUUID, 24, "same shape as the order id upstream already accepts")
	assert.Equal(t, 0, oldP.calls, "the old supplier is never called again")

	require.Len(t, got.Fulfillment.SupplierAttempts, 1)
	h := got.Fulfillment.SupplierAttempts[0]
	assert.Equal(t, 7, *h.Provider)
	assert.Equal(t, "364", h.UpstreamProductID)
	assert.Equal(t, o.ID.Hex(), h.OrderUUID)
	assert.Equal(t, "admin@x", h.ReroutedBy)
	assert.Equal(t, 1, got.Fulfillment.SupplierAttempt)
	assert.Equal(t, 9, *got.Items[0].FulfillmentProvider)
	assert.Equal(t, 0, walletSvc.refundCalls)

	actions := []string{}
	for _, e := range rec.entries {
		actions = append(actions, e.Action)
	}
	assert.Contains(t, actions, audit.ActionOrderSupplierReroute)
}

func TestReroute_Eligibility(t *testing.T) {
	ctx := context.Background()
	in := RerouteInput{ProviderID: 9, UpstreamProductID: "U-900"}

	t.Run("supplier still working (ref, not stuck) → refused", func(t *testing.T) {
		st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, &fakeProvider{id: 9})
		o := settlerOrder(repo, func(o *Order) { o.Fulfillment.ProviderRef = "UP_1" })
		_, err := st.Reroute(ctx, o.ID, in, "a")
		assert.Equal(t, "REROUTE_NOT_STUCK", appCode(err))
	})

	t.Run("ref-less park (no stuck flag needed) → allowed", func(t *testing.T) {
		newP := &fakeProvider{id: 9, err: provider.ErrPending, res: provider.Result{Reference: "P9"}}
		st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, newP)
		o := settlerOrder(repo, nil)
		got, err := st.Reroute(ctx, o.ID, in, "a")
		require.NoError(t, err)
		assert.Equal(t, OrderStatusProcessing, got.Status)
		assert.Equal(t, "P9", got.Fulfillment.ProviderRef, "pending at the new supplier → pollable by its ref")
	})

	t.Run("not processing → refused", func(t *testing.T) {
		st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, &fakeProvider{id: 9})
		o := settlerOrder(repo, func(o *Order) { o.Status = OrderStatusRefunded })
		_, err := st.Reroute(ctx, o.ID, in, "a")
		assert.Equal(t, "REROUTE_NOT_PROCESSING", appCode(err))
	})

	t.Run("unknown supplier / missing product → refused", func(t *testing.T) {
		st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, &fakeProvider{id: 9})
		o := settlerOrder(repo, stuckAt)
		_, err := st.Reroute(ctx, o.ID, RerouteInput{ProviderID: 42, UpstreamProductID: "x"}, "a")
		assert.Equal(t, "REROUTE_UNKNOWN_SUPPLIER", appCode(err))
		_, err = st.Reroute(ctx, o.ID, RerouteInput{ProviderID: 9}, "a")
		assert.Equal(t, "REROUTE_UPSTREAM_REQUIRED", appCode(err))
	})

	t.Run("ambiguous umanage attempt needs explicit acknowledgement", func(t *testing.T) {
		newP := &fakeProvider{id: 9, res: provider.Result{Reference: "N"}}
		st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, newP)
		o := settlerOrder(repo, func(o *Order) {
			o.Fulfillment.ProviderRef = "search:ALFA:" + o.ID.Hex()
			stuckAt(o)
		})
		_, err := st.Reroute(ctx, o.ID, in, "a")
		assert.Equal(t, "REROUTE_AMBIGUOUS", appCode(err))
		assert.Equal(t, 0, newP.calls)

		ack := in
		ack.AcknowledgeAmbiguous = true
		got, err := st.Reroute(ctx, o.ID, ack, "a")
		require.NoError(t, err)
		assert.Equal(t, "search:ALFA:"+o.ID.Hex(), got.Fulfillment.SupplierAttempts[0].ProviderRef, "old ref archived for reconciliation")
	})
}

func TestReroute_StaleClaimConflicts(t *testing.T) {
	st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, &fakeProvider{id: 9})
	o := settlerOrder(repo, stuckAt)
	// The state changed between the admin's read and the claim (e.g. a
	// concurrent reroute already bumped the attempt).
	_, err := st.svc.repo.ClaimSupplierReroute(context.Background(), RerouteClaim{
		OrderID: o.ID, ExpectedAttempt: 1, NewProvider: 9, NewUpstreamID: "x", NewOrderUUID: "u",
	})
	assert.ErrorIs(t, err, apperrors.ErrConflict)
	// A settler that promoted the park meanwhile (ref changed) also wins.
	o.Fulfillment.ProviderRef = "UP_LATE"
	_, err = st.svc.repo.ClaimSupplierReroute(context.Background(), RerouteClaim{
		OrderID: o.ID, ExpectedAttempt: 0, ExpectedRef: "", NewProvider: 9, NewUpstreamID: "x", NewOrderUUID: "u",
	})
	assert.ErrorIs(t, err, apperrors.ErrConflict)
}

func TestReroute_RejectionFlagsStuckWithoutRefund_ThenRerouteAgain(t *testing.T) {
	newP := &fakeProvider{id: 9, err: errors.New("panel: invalid player id")}
	st, repo, walletSvc, _ := newRerouteSUT(&fakeProvider{id: 7}, newP)
	o := settlerOrder(repo, stuckAt)
	ctx := context.Background()

	got, err := st.Reroute(ctx, o.ID, RerouteInput{ProviderID: 9, UpstreamProductID: "U-1"}, "a")
	require.NoError(t, err)
	assert.Equal(t, OrderStatusProcessing, got.Status, "no automatic refund after an admin reroute")
	assert.NotNil(t, got.Fulfillment.SupplierStuckAt, "flagged stuck again for the admin")
	assert.Equal(t, 0, walletSvc.refundCalls)
	assert.Equal(t, 1, countTimelineNote(got, "stuck upstream — needs attention: rejected by the new supplier"))
	firstKey := newP.lastIn.OrderUUID

	newP.err, newP.res = nil, provider.Result{Reference: "OK2"}
	got, err = st.Reroute(ctx, o.ID, RerouteInput{ProviderID: 9, UpstreamProductID: "U-2"}, "a")
	require.NoError(t, err)
	assert.Equal(t, OrderStatusCompleted, got.Status)
	assert.Equal(t, 2, got.Fulfillment.SupplierAttempt)
	assert.Len(t, got.Fulfillment.SupplierAttempts, 2)
	assert.NotEqual(t, firstKey, newP.lastIn.OrderUUID, "every attempt gets its own key")
}

func TestReroute_UnavailableRetriesWithFreshGiveUpWindow(t *testing.T) {
	newP := &fakeProvider{id: 9, err: provider.ErrUnavailable}
	st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, newP)
	// A three-day-old order: measured from CreatedAt it is long past give-up.
	o := settlerOrder(repo, func(o *Order) {
		o.CreatedAt = time.Now().UTC().Add(-72 * time.Hour)
		stuckAt(o)
	})
	ctx := context.Background()

	got, err := st.Reroute(ctx, o.ID, RerouteInput{ProviderID: 9, UpstreamProductID: "U-1"}, "a")
	require.NoError(t, err)
	require.NotNil(t, got.Fulfillment.SupplierNextRetryAt, "environmental failure → normal retry clock")
	assert.Nil(t, got.Fulfillment.SupplierStuckAt)
	key := newP.lastIn.OrderUUID

	// The retry comes due: the settler must re-dispatch (same new key), not
	// immediately flag it stuck because the ORDER is three days old.
	past := time.Now().UTC().Add(-time.Second)
	got.Fulfillment.SupplierNextRetryAt = &past
	newP.err, newP.res = nil, provider.Result{Reference: "DONE"}
	st.Tick(ctx)
	assert.Equal(t, OrderStatusCompleted, got.Status)
	assert.Equal(t, key, newP.lastIn.OrderUUID, "retries within one attempt reuse its key (deduped upstream)")
}
