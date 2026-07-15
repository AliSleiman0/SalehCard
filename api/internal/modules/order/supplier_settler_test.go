package order

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/payments"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

// fakeNotifier records every note so tests can assert exactly-once delivery.
type fakeNotifier struct{ notes []notification.Note }

func (f *fakeNotifier) Notify(_ context.Context, _ bson.ObjectID, n notification.Note) {
	f.notes = append(f.notes, n)
}
func (f *fakeNotifier) NotifyMany(_ context.Context, _ []bson.ObjectID, _ notification.Note) int {
	return 0
}

// fakeRecorder records audit entries (asserts action + system actor).
type fakeRecorder struct{ entries []audit.Entry }

func (f *fakeRecorder) Record(_ context.Context, e audit.Entry) { f.entries = append(f.entries, e) }

// newSettlerSUT builds a SupplierSettler over an in-memory SUT with one
// scripted provider (id 7) and a short give-up window supplied per test.
func newSettlerSUT(fake *fakeProvider, giveUp time.Duration) (*SupplierSettler, *fakeOrderRepo, *fakeWalletSvc, *fakeNotifier, *fakeRecorder, *fakeAwarder) {
	repo := newFakeOrderRepo()
	walletSvc := &fakeWalletSvc{}
	ntf := &fakeNotifier{}
	aw := &fakeAwarder{}
	svc := NewOrderService(repo, &fakeProductSvc{byID: map[string]*product.Product{}},
		&fakeCodeSvc{available: map[string]int{}}, walletSvc, &fakePromoSvc{}, &fakeOfferSvc{},
		provider.NewRegistry(fake), payments.New(payments.Config{}), nil,
		&fakeKycGate{approved: true}, nil, ntf, aw, nil)
	rec := &fakeRecorder{}
	return NewSupplierSettler(svc, rec, time.Second, giveUp), repo, walletSvc, ntf, rec, aw
}

// settlerOrder seeds one processing api-mode order (provider 7, upstream 364)
// into the fake repo; mut customizes it before insertion.
func settlerOrder(repo *fakeOrderRepo, mut func(*Order)) *Order {
	pid := 7
	o := &Order{
		ID:     bson.NewObjectID(),
		UserID: bson.NewObjectID(),
		Items: []OrderItem{{
			ProductID:           bson.NewObjectID(),
			VariantID:           bson.NewObjectID(),
			Qty:                 1,
			Price:               5,
			FulfillmentType:     "account_credit",
			FulfillmentMode:     "api",
			FulfillmentProvider: &pid,
			UpstreamProductID:   "364",
			PlayerID:            "player-9",
			Fields:              []OrderField{{Key: "playerId", Value: "player-9"}},
		}},
		Subtotal:      5,
		Total:         5,
		Currency:      "USD",
		PaymentMethod: PaymentMethodWallet,
		Status:        OrderStatusProcessing,
		CreatedAt:     time.Now().UTC(),
	}
	if mut != nil {
		mut(o)
	}
	repo.byID[o.ID] = o
	return o
}

func countTimelineNote(o *Order, note string) int {
	n := 0
	for _, e := range o.Fulfillment.StatusTimeline {
		if e.Note == note {
			n++
		}
	}
	return n
}

func TestSettler_AcceptCompletes(t *testing.T) {
	fake := &fakeProvider{id: 7, checkRes: provider.Status{State: "accept", Codes: []string{"C1", "C2"}}}
	st, repo, walletSvc, ntf, rec, aw := newSettlerSUT(fake, 24*time.Hour)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.ProviderRef = "UP_1" })

	st.Tick(context.Background())

	assert.Equal(t, "UP_1", fake.lastCheckRef)
	assert.Equal(t, OrderStatusCompleted, o.Status)
	assert.Equal(t, "C1\nC2", o.Fulfillment.DeliveredCode)
	assert.Equal(t, "UP_1", o.Fulfillment.TransferRef)
	assert.Len(t, ntf.notes, 1)
	assert.Equal(t, 1, aw.calls)
	require.Len(t, rec.entries, 1)
	assert.Equal(t, audit.ActionOrderSupplierSettle, rec.entries[0].Action)
	assert.Equal(t, "system:supplier-settler", rec.entries[0].ActorID)
	assert.Equal(t, 0, walletSvc.refundCalls)
}

func TestSettler_AcceptIdempotentOnMovedOrder(t *testing.T) {
	fake := &fakeProvider{id: 7, checkRes: provider.Status{State: "accept"}}
	st, repo, _, ntf, rec, aw := newSettlerSUT(fake, 24*time.Hour)
	settlerOrder(repo, func(o *Order) {
		o.Fulfillment.ProviderRef = "UP_1"
		o.Status = OrderStatusRefunded // admin refunded before the supplier delivered
	})

	// The pollable query excludes non-processing orders, so drive settleAccept
	// directly to exercise the lost-race branch.
	o := repo.byID[func() bson.ObjectID {
		for id := range repo.byID {
			return id
		}
		panic("empty")
	}()]
	require.NoError(t, st.settleAccept(context.Background(), o, "UP_1", nil, "completed by supplier check"))

	assert.Equal(t, OrderStatusRefunded, o.Status) // untouched
	assert.Empty(t, ntf.notes)
	assert.Empty(t, rec.entries)
	assert.Equal(t, 0, aw.calls)
	assert.Equal(t, 1, countTimelineNote(o, "supplier delivered after order left processing — reconcile manually"))
}

func TestSettler_RejectRefundsOnce(t *testing.T) {
	fake := &fakeProvider{id: 7, checkRes: provider.Status{State: "reject"}}
	st, repo, walletSvc, ntf, rec, _ := newSettlerSUT(fake, 24*time.Hour)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.ProviderRef = "UP_r" })

	st.Tick(context.Background())

	assert.Equal(t, OrderStatusRefunded, o.Status)
	assert.Equal(t, 1, walletSvc.refundCalls)
	assert.Equal(t, 5.0, walletSvc.refunded)
	assert.Equal(t, 1, countTimelineNote(o, "rejected upstream — refunded"))
	require.Len(t, rec.entries, 1)
	assert.Equal(t, audit.ActionOrderRefund, rec.entries[0].Action)
	assert.Equal(t, "system:supplier-settler", rec.entries[0].ActorID)
	require.Len(t, ntf.notes, 1)
	assert.Equal(t, notification.KindOrderRefunded, ntf.notes[0].Kind)

	// A second tick must not refund again (order no longer processing).
	st.Tick(context.Background())
	assert.Equal(t, 1, walletSvc.refundCalls)
	assert.Len(t, ntf.notes, 1)
}

func TestSettler_RejectLosesRaceToAdminRefund(t *testing.T) {
	fake := &fakeProvider{id: 7, checkRes: provider.Status{State: "reject"}}
	st, repo, walletSvc, ntf, rec, _ := newSettlerSUT(fake, 24*time.Hour)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.ProviderRef = "UP_r" })

	// Admin refund wins between the list query and the settle.
	o.Status = OrderStatusRefunded
	require.NoError(t, st.settleReject(context.Background(), o, "rejected upstream — refunded"))

	assert.Equal(t, 0, walletSvc.refundCalls) // the lock held: zero money moved
	assert.Empty(t, ntf.notes)
	assert.Empty(t, rec.entries)
}

func TestSettler_RejectWalletFailureRevertsAndRetries(t *testing.T) {
	fake := &fakeProvider{id: 7, checkRes: provider.Status{State: "reject"}}
	st, repo, walletSvc, ntf, _, _ := newSettlerSUT(fake, 24*time.Hour)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.ProviderRef = "UP_r" })

	walletSvc.refundErr = errors.New("ledger down")
	st.Tick(context.Background())

	// Reverted to processing so the next tick can retry; no notification sent.
	assert.Equal(t, OrderStatusProcessing, o.Status)
	assert.Equal(t, 1, countTimelineNote(o, "charge reversal failed"))
	assert.Empty(t, ntf.notes)

	// Wallet healed → exactly one successful refund on the next tick.
	walletSvc.refundErr = nil
	st.Tick(context.Background())
	assert.Equal(t, OrderStatusRefunded, o.Status)
	assert.Equal(t, 5.0, walletSvc.refunded)
	assert.Len(t, ntf.notes, 1)
}

func TestSettler_WaitLeavesParked(t *testing.T) {
	fake := &fakeProvider{id: 7, checkRes: provider.Status{State: "wait"}}
	st, repo, walletSvc, ntf, rec, _ := newSettlerSUT(fake, 24*time.Hour)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.ProviderRef = "UP_w" })

	st.Tick(context.Background())

	assert.Equal(t, OrderStatusProcessing, o.Status)
	assert.Nil(t, o.Fulfillment.SupplierStuckAt)
	assert.Equal(t, 0, walletSvc.refundCalls)
	assert.Empty(t, ntf.notes)
	assert.Empty(t, rec.entries)

	// Still pollable next tick.
	st.Tick(context.Background())
	assert.Equal(t, 2, fake.checkCalls)
}

func TestSettler_WaitPastGiveUpFlagsStuckOnce(t *testing.T) {
	fake := &fakeProvider{id: 7, checkRes: provider.Status{State: "wait"}}
	st, repo, walletSvc, _, rec, _ := newSettlerSUT(fake, time.Minute)
	o := settlerOrder(repo, func(o *Order) {
		o.Fulfillment.ProviderRef = "UP_w"
		o.CreatedAt = time.Now().UTC().Add(-2 * time.Minute) // past give-up
	})

	st.Tick(context.Background())
	st.Tick(context.Background()) // second tick must not double-flag

	assert.Equal(t, OrderStatusProcessing, o.Status) // never auto-refunded
	require.NotNil(t, o.Fulfillment.SupplierStuckAt)
	assert.Equal(t, 1, countTimelineNote(o, "stuck upstream — needs attention: still waiting upstream past give-up window"))
	require.Len(t, rec.entries, 1)
	assert.Equal(t, audit.ActionOrderSupplierSettle, rec.entries[0].Action)
	assert.Equal(t, 0, walletSvc.refundCalls)
	// Stuck-but-ref'd orders keep being polled — the supplier may still deliver.
	assert.Equal(t, 2, fake.checkCalls)
}

func TestSettler_CheckErrorNeverRejects(t *testing.T) {
	fake := &fakeProvider{id: 7, checkErr: fmt.Errorf("boom: %w", provider.ErrUnavailable)}
	st, repo, walletSvc, ntf, _, _ := newSettlerSUT(fake, 24*time.Hour)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.ProviderRef = "UP_w" })

	st.Tick(context.Background())

	assert.Equal(t, OrderStatusProcessing, o.Status)
	assert.Equal(t, 0, walletSvc.refundCalls)
	assert.Empty(t, ntf.notes)
}

func TestSettler_RedispatchSuccessCompletes(t *testing.T) {
	fake := &fakeProvider{id: 7, res: provider.Result{Reference: "UP_new", Codes: []string{"C9"}}}
	st, repo, _, ntf, rec, aw := newSettlerSUT(fake, 24*time.Hour)
	due := time.Now().UTC().Add(-time.Second)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.SupplierNextRetryAt = &due })

	st.Tick(context.Background())

	require.NotNil(t, fake.lastIn)
	assert.Equal(t, o.ID.Hex(), fake.lastIn.OrderUUID) // SAME idempotency key
	assert.Equal(t, "364", fake.lastIn.UpstreamID)
	assert.Equal(t, OrderStatusCompleted, o.Status)
	assert.Equal(t, "C9", o.Fulfillment.DeliveredCode)
	assert.Equal(t, "UP_new", o.Fulfillment.ProviderRef)
	assert.Len(t, ntf.notes, 1)
	assert.Equal(t, 1, aw.calls)
	assert.Len(t, rec.entries, 1)
}

func TestSettler_RedispatchPendingPromotesToPollable(t *testing.T) {
	fake := &fakeProvider{
		id:  7,
		res: provider.Result{Reference: "UP_wait2"},
		err: fmt.Errorf("pending: %w", provider.ErrPending),
	}
	st, repo, _, _, _, _ := newSettlerSUT(fake, 24*time.Hour)
	due := time.Now().UTC().Add(-time.Second)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.SupplierNextRetryAt = &due })

	st.Tick(context.Background())

	assert.Equal(t, OrderStatusProcessing, o.Status)
	assert.Equal(t, "UP_wait2", o.Fulfillment.ProviderRef)
	assert.Nil(t, o.Fulfillment.SupplierNextRetryAt) // left the retryable class
	assert.Equal(t, 1, countTimelineNote(o, "awaiting upstream provider"))

	// Next tick polls CheckStatus instead of Fulfill.
	fake.checkRes = provider.Status{State: "wait"}
	st.Tick(context.Background())
	assert.Equal(t, 1, fake.calls) // no second Fulfill
	assert.Equal(t, 1, fake.checkCalls)
}

func TestSettler_BackoffProgression(t *testing.T) {
	// Pure schedule.
	for i, want := range []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, time.Hour, time.Hour} {
		assert.Equal(t, want, backoffDelay(i+1), "attempt %d", i+1)
	}

	// Integration: an unavailable re-dispatch bumps count + clock, quietly.
	fake := &fakeProvider{id: 7, err: fmt.Errorf("balance: %w", provider.ErrUnavailable)}
	st, repo, _, _, _, _ := newSettlerSUT(fake, 24*time.Hour)
	due := time.Now().UTC().Add(-time.Second)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.SupplierNextRetryAt = &due })
	timelineLen := len(o.Fulfillment.StatusTimeline)

	st.Tick(context.Background())

	assert.Equal(t, 1, o.Fulfillment.SupplierRetryCount)
	require.NotNil(t, o.Fulfillment.SupplierNextRetryAt)
	assert.True(t, o.Fulfillment.SupplierNextRetryAt.After(time.Now().UTC()))
	assert.Len(t, o.Fulfillment.StatusTimeline, timelineLen) // quiet — no timeline growth

	// Not due yet → next tick skips it entirely.
	st.Tick(context.Background())
	assert.Equal(t, 1, fake.calls)
}

func TestSettler_RedispatchExhaustedFlagsStuck(t *testing.T) {
	fake := &fakeProvider{id: 7, err: fmt.Errorf("balance: %w", provider.ErrUnavailable)}
	st, repo, walletSvc, _, rec, _ := newSettlerSUT(fake, time.Minute)
	due := time.Now().UTC().Add(-time.Second)
	o := settlerOrder(repo, func(o *Order) {
		o.Fulfillment.SupplierNextRetryAt = &due
		o.Fulfillment.SupplierRetryCount = 9
		o.CreatedAt = time.Now().UTC().Add(-2 * time.Minute) // past give-up
	})

	st.Tick(context.Background())
	st.Tick(context.Background())

	assert.Equal(t, 0, fake.calls) // gave up BEFORE dispatching again
	require.NotNil(t, o.Fulfillment.SupplierStuckAt)
	assert.Nil(t, o.Fulfillment.SupplierNextRetryAt) // permanently out of the retry class
	assert.Equal(t, OrderStatusProcessing, o.Status)
	assert.Equal(t, 0, walletSvc.refundCalls)
	assert.Len(t, rec.entries, 1) // flagged exactly once
}

func TestSettler_RedispatchHardErrorRefunds(t *testing.T) {
	fake := &fakeProvider{id: 7, err: errors.New("panel jentel: player id blocked (code 107)")}
	st, repo, walletSvc, ntf, _, _ := newSettlerSUT(fake, 24*time.Hour)
	due := time.Now().UTC().Add(-time.Second)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.SupplierNextRetryAt = &due })

	st.Tick(context.Background())

	assert.Equal(t, OrderStatusRefunded, o.Status)
	assert.Equal(t, 1, walletSvc.refundCalls)
	assert.Len(t, ntf.notes, 1)
}

func TestSettler_RedispatchSuccessLosesRaceToAdmin(t *testing.T) {
	fake := &fakeProvider{id: 7, res: provider.Result{Reference: "UP_new"}}
	st, repo, walletSvc, ntf, _, _ := newSettlerSUT(fake, 24*time.Hour)
	due := time.Now().UTC().Add(-time.Second)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.SupplierNextRetryAt = &due })

	// The admin completes/refunds between the list query and the settle: the
	// upstream Fulfill still runs, but the guarded transition must lose cleanly.
	require.NoError(t, st.svc.repo.BumpSupplierRetry(context.Background(), o.ID, 1, due)) // keep retryable
	o.Status = OrderStatusRefunded
	require.NoError(t, st.redispatchOne(context.Background(), o))

	assert.Equal(t, OrderStatusRefunded, o.Status) // not resurrected
	assert.Equal(t, 0, walletSvc.refundCalls)
	assert.Empty(t, ntf.notes)
	assert.Equal(t, 1, countTimelineNote(o, "supplier delivered after order left processing — reconcile manually"))
}