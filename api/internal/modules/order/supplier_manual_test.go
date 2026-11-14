package order

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// manualOrder seeds a processing, never-sent manual account-credit order (the
// client's SOUL STAR shape: per-unit, 10,000 coins typed into "amount").
func manualOrder(repo *fakeOrderRepo, mut func(*Order)) *Order {
	o := &Order{
		ID:     bson.NewObjectID(),
		UserID: bson.NewObjectID(),
		Items: []OrderItem{{
			ProductID:       bson.NewObjectID(),
			VariantID:       bson.NewObjectID(),
			Qty:             1,
			Amount:          f64p(10000),
			Price:           1.23,
			FulfillmentType: "account_credit",
			FulfillmentMode: "manual_operator",
			PlayerID:        "64579619",
			Fields:          []OrderField{{Key: "id", Value: "64579619"}, {Key: "amount", Value: "10000"}},
		}},
		Subtotal:      1.23,
		Total:         1.23,
		Currency:      "USD",
		PaymentMethod: PaymentMethodWallet,
		Status:        OrderStatusProcessing,
		CreatedAt:     time.Now().UTC().Add(-time.Hour),
	}
	if mut != nil {
		mut(o)
	}
	repo.byID[o.ID] = o
	return o
}

func TestSendManualOrderToSupplier_Completes(t *testing.T) {
	newP := &fakeProvider{id: 9, res: provider.Result{Reference: "UP_M1"}}
	st, repo, walletSvc, rec := newRerouteSUT(&fakeProvider{id: 7}, newP)
	o := manualOrder(repo, nil)

	got, err := st.Reroute(context.Background(), o.ID, RerouteInput{ProviderID: 9, UpstreamProductID: "2"}, "admin@x")
	require.NoError(t, err)

	assert.Equal(t, OrderStatusCompleted, got.Status)
	assert.Equal(t, "api", got.Items[0].FulfillmentMode, "the line becomes a supplier line")
	assert.Equal(t, 9, *got.Items[0].FulfillmentProvider)
	assert.Equal(t, "2", got.Items[0].UpstreamProductID)
	assert.Empty(t, got.Fulfillment.SupplierAttempts, "a never-sent manual line has no earlier attempt to archive")
	assert.Equal(t, 1, got.Fulfillment.SupplierAttempt)

	require.Equal(t, 1, newP.calls)
	assert.Equal(t, 10000, newP.lastIn.Qty, "the typed units, not the cart qty")
	assert.Equal(t, "64579619", newP.lastIn.PlayerID)
	assert.NotEqual(t, o.ID.Hex(), newP.lastIn.OrderUUID, "fresh per-attempt key")
	assert.Equal(t, 0, walletSvc.refundCalls)

	var reroute *audit.Entry
	for i := range rec.entries {
		if rec.entries[i].Action == audit.ActionOrderSupplierReroute {
			reroute = &rec.entries[i]
		}
	}
	require.NotNil(t, reroute)
	assert.Equal(t, "manual_operator", reroute.Summary["fromMode"])
	assert.Equal(t, "sent to supplier (#9) by admin", got.Fulfillment.StatusTimeline[0].Note)
}

// A legacy line with no stored mode derives manual for account credit.
func TestSendManualOrderToSupplier_LegacyModeLessLine(t *testing.T) {
	newP := &fakeProvider{id: 9, res: provider.Result{Reference: "UP_M2"}}
	st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, newP)
	o := manualOrder(repo, func(o *Order) { o.Items[0].FulfillmentMode = "" })

	got, err := st.Reroute(context.Background(), o.ID, RerouteInput{ProviderID: 9, UpstreamProductID: "2"}, "a")
	require.NoError(t, err)
	assert.Equal(t, OrderStatusCompleted, got.Status)
}

func TestSendManualOrderToSupplier_Ineligible(t *testing.T) {
	in := RerouteInput{ProviderID: 9, UpstreamProductID: "2"}
	cases := map[string]func(*Order){
		"money transfer": func(o *Order) { o.Items[0].FulfillmentType = "transfer" },
		"bridge recharge": func(o *Order) {
			o.Items[0].FulfillmentMode = "bridge_device"
		},
		"two lines": func(o *Order) { o.Items = append(o.Items, o.Items[0]) },
		"code (inventory) line": func(o *Order) {
			o.Items[0].FulfillmentType = "code"
			o.Items[0].FulfillmentMode = "inventory"
		},
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			newP := &fakeProvider{id: 9}
			st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, newP)
			o := manualOrder(repo, mut)
			_, err := st.Reroute(context.Background(), o.ID, in, "a")
			assert.Equal(t, "REROUTE_NOT_SUPPLIER_ORDER", appCode(err))
			assert.Equal(t, 0, newP.calls)
		})
	}

	t.Run("already completed by hand", func(t *testing.T) {
		st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, &fakeProvider{id: 9})
		o := manualOrder(repo, func(o *Order) { o.Status = OrderStatusCompleted })
		_, err := st.Reroute(context.Background(), o.ID, in, "a")
		assert.Equal(t, "REROUTE_NOT_PROCESSING", appCode(err))
	})

	t.Run("corrupt amount is never sent", func(t *testing.T) {
		newP := &fakeProvider{id: 9}
		st, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, newP)
		o := manualOrder(repo, func(o *Order) { o.Items[0].Amount = f64p(0.5) })
		_, err := st.Reroute(context.Background(), o.ID, in, "a")
		assert.Equal(t, "REROUTE_BAD_QUANTITY", appCode(err))
		assert.Equal(t, 0, newP.calls)
	})
}

// The guarded claim pins the line's mode: if the line stopped being manual
// between the read and the claim, the claim loses.
func TestSendManualOrderToSupplier_ModeChangedConflicts(t *testing.T) {
	_, repo, _, _ := newRerouteSUT(&fakeProvider{id: 7}, &fakeProvider{id: 9})
	o := manualOrder(repo, nil)
	_, err := repo.ClaimSupplierReroute(context.Background(), RerouteClaim{
		OrderID: o.ID, ItemIndex: 0, FromManual: false, NewProvider: 9, NewUpstreamID: "2", NewOrderUUID: "k",
	})
	assert.True(t, errors.Is(err, apperrors.ErrConflict))
}

// A later rejection (found by the settler's poll) of an admin-chosen route is
// flagged for the admin — never auto-refunded — and leaves the pollable class.
func TestSettler_PollRejectAfterAdminRouteFlagsWithoutRefund(t *testing.T) {
	fake := &fakeProvider{id: 7, checkRes: provider.Status{State: "reject"}}
	st, repo, walletSvc, _, rec, _ := newSettlerSUT(fake, 24*time.Hour)
	o := settlerOrder(repo, func(o *Order) {
		o.Fulfillment.ProviderRef = "UP_9"
		o.Fulfillment.SupplierAttempt = 1
	})

	st.Tick(context.Background())

	assert.Equal(t, OrderStatusProcessing, o.Status)
	assert.Equal(t, 0, walletSvc.refundCalls)
	assert.NotNil(t, o.Fulfillment.SupplierStuckAt)
	assert.Empty(t, o.Fulfillment.ProviderRef, "the dead ref is dropped so it is not re-polled")
	require.NotEmpty(t, rec.entries)
	assert.Equal(t, "rejected_after_reroute", rec.entries[len(rec.entries)-1].Summary["outcome"])

	// Next tick: nothing left to poll, no second audit entry.
	n := len(rec.entries)
	st.Tick(context.Background())
	assert.Equal(t, 1, fake.checkCalls)
	assert.Len(t, rec.entries, n)

	// And the admin can send it to another supplier straight away.
	_, _, err := rerouteEligible(o)
	assert.NoError(t, err)
}

// A rejection on re-dispatch of an admin-chosen route: same rule.
func TestSettler_RedispatchRejectAfterAdminRouteFlagsWithoutRefund(t *testing.T) {
	fake := &fakeProvider{id: 7, err: errors.New("panel: invalid quantity (code 112)")}
	st, repo, walletSvc, _, _, _ := newSettlerSUT(fake, 24*time.Hour)
	due := time.Now().UTC().Add(-time.Second)
	o := settlerOrder(repo, func(o *Order) {
		o.Fulfillment.SupplierAttempt = 2
		o.Fulfillment.SupplierNextRetryAt = &due
	})

	st.Tick(context.Background())

	assert.Equal(t, OrderStatusProcessing, o.Status)
	assert.Equal(t, 0, walletSvc.refundCalls)
	assert.NotNil(t, o.Fulfillment.SupplierStuckAt)
	assert.Nil(t, o.Fulfillment.SupplierNextRetryAt)
}

// Placement-time supplier orders (attempt 0) keep the automatic refund.
func TestSettler_PollRejectAtPlacementStillRefunds(t *testing.T) {
	fake := &fakeProvider{id: 7, checkRes: provider.Status{State: "reject"}}
	st, repo, walletSvc, _, _, _ := newSettlerSUT(fake, 24*time.Hour)
	o := settlerOrder(repo, func(o *Order) { o.Fulfillment.ProviderRef = "UP_0" })

	st.Tick(context.Background())

	assert.Equal(t, OrderStatusRefunded, o.Status)
	assert.Equal(t, 1, walletSvc.refundCalls)
}

func TestOrderAtSupplier(t *testing.T) {
	now := time.Now().UTC()
	repo := newFakeOrderRepo()
	assert.False(t, orderAtSupplier(manualOrder(repo, nil)), "manual order")
	assert.False(t, orderAtSupplier(settlerOrder(repo, nil)), "stub park: no ref, no retry clock")
	assert.True(t, orderAtSupplier(settlerOrder(repo, func(o *Order) { o.Fulfillment.ProviderRef = "R" })), "accepted, polled")
	assert.True(t, orderAtSupplier(settlerOrder(repo, func(o *Order) { o.Fulfillment.SupplierNextRetryAt = &now })), "retry pending")
	assert.True(t, orderAtSupplier(settlerOrder(repo, func(o *Order) {
		o.Fulfillment.ProviderRef = "R"
		o.Fulfillment.SupplierStuckAt = &now
	})), "stuck for slowness with a live ref: the supplier can still deliver")
	assert.False(t, orderAtSupplier(settlerOrder(repo, stuckAt)), "stuck, ref-less: nothing in flight")
	assert.False(t, orderAtSupplier(settlerOrder(repo, func(o *Order) {
		o.Fulfillment.ProviderRef = "R"
		o.Status = OrderStatusCompleted
	})), "a finished order is never at the supplier")
}

// Completing by hand an order a supplier may still deliver needs an explicit
// acknowledgement (double-delivery guard).
func TestAdminCompleteAtSupplierNeedsAcknowledgement(t *testing.T) {
	pid := 7
	o := &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Total: 5, PaymentMethod: PaymentMethodWallet,
		Status:      OrderStatusProcessing,
		Items:       []OrderItem{{FulfillmentMode: "api", FulfillmentProvider: &pid, Qty: 1}},
		Fulfillment: Fulfillment{ProviderRef: "UP_7"},
	}
	a, repo, _, rec := newActionFixture(o)
	route := func(r chi.Router) { r.Put("/orders/{id}/status", a.updateStatus) }

	rr := doAdmin(t, http.MethodPut, "/orders/"+o.ID.Hex()+"/status", `{"status":"completed"}`, route)
	require.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "ORDER_AT_SUPPLIER")
	assert.Equal(t, OrderStatusProcessing, repo.byID[o.ID].Status)

	rr = doAdmin(t, http.MethodPut, "/orders/"+o.ID.Hex()+"/status", `{"status":"completed","acknowledgeSupplier":true}`, route)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	assert.Equal(t, OrderStatusCompleted, repo.byID[o.ID].Status)
	require.Len(t, rec.entries, 1)
	assert.Equal(t, true, rec.entries[0].Summary["acknowledgedSupplier"])
}

// Refunding an order a supplier may still deliver needs the same explicit
// acknowledgement (the customer could keep both the money and the goods);
// bulk refunds skip such orders instead of refunding them.
func TestAdminRefundAtSupplierNeedsAcknowledgement(t *testing.T) {
	pid := 7
	atSupplier := func() *Order {
		return &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Total: 5, PaymentMethod: PaymentMethodWallet,
			Status:      OrderStatusProcessing,
			Items:       []OrderItem{{FulfillmentMode: "api", FulfillmentProvider: &pid, Qty: 1}},
			Fulfillment: Fulfillment{ProviderRef: "UP_7"},
		}
	}
	o := atSupplier()
	a, repo, wal, _ := newActionFixture(o)
	route := func(r chi.Router) { r.Post("/orders/{id}/refund", a.refund) }

	rr := doAdmin(t, http.MethodPost, "/orders/"+o.ID.Hex()+"/refund", `{}`, route)
	require.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "ORDER_AT_SUPPLIER")
	assert.Empty(t, wal.refunded)

	rr = doAdmin(t, http.MethodPost, "/orders/"+o.ID.Hex()+"/refund", `{"acknowledgeSupplier":true}`, route)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	assert.Equal(t, OrderStatusRefunded, repo.byID[o.ID].Status)
	assert.Equal(t, []float64{5}, wal.refunded)

	// A completed supplier order (ref kept) refunds without any confirmation.
	done := atSupplier()
	done.Status = OrderStatusCompleted
	b, _, _, _ := newActionFixture(done)
	rr = doAdmin(t, http.MethodPost, "/orders/"+done.ID.Hex()+"/refund", `{}`,
		func(r chi.Router) { r.Post("/orders/{id}/refund", b.refund) })
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	// Bulk: the at-supplier order is skipped, the plain one refunded.
	busy, plain := atSupplier(), &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Total: 3,
		PaymentMethod: PaymentMethodWallet, Status: OrderStatusProcessing}
	c, crepo, _, _ := newActionFixture(busy, plain)
	rr = doAdmin(t, http.MethodPost, "/orders/refund-bulk",
		`{"ids":["`+busy.ID.Hex()+`","`+plain.ID.Hex()+`"]}`,
		func(r chi.Router) { r.Post("/orders/refund-bulk", c.refundBulk) })
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"at_supplier"`)
	assert.Equal(t, OrderStatusProcessing, crepo.byID[busy.ID].Status)
	assert.Equal(t, OrderStatusRefunded, crepo.byID[plain.ID].Status)
}

// A rejection of an admin-chosen route is recorded once even when two ticks
// see it, and the rejected attempt's supplier ref survives into the history.
func TestRejectAdminRoute_OnceAndRefKept(t *testing.T) {
	fake := &fakeProvider{id: 7, checkRes: provider.Status{State: "reject"}}
	st, repo, _, _, rec, _ := newSettlerSUT(fake, 24*time.Hour)
	o := settlerOrder(repo, func(o *Order) {
		o.Fulfillment.ProviderRef = "UP_9"
		o.Fulfillment.SupplierAttempt = 1
	})
	stale := *o // a second replica's snapshot, read before the first write

	st.Tick(context.Background())
	require.NoError(t, st.pollOne(context.Background(), &stale))

	settles := 0
	for _, e := range rec.entries {
		if e.Summary["outcome"] == "rejected_after_reroute" {
			settles++
			assert.Equal(t, "UP_9", e.Summary["providerRef"])
		}
	}
	assert.Equal(t, 1, settles, "recorded once")
	assert.Equal(t, 1, countTimelineNote(o, "stuck upstream — needs attention: rejected by the supplier"))
	assert.Equal(t, "UP_9", o.Fulfillment.RejectedProviderRef)

	// The next reroute archives the rejected ref, not an empty one.
	newP := &fakeProvider{id: 9, res: provider.Result{Reference: "OK"}}
	st2, repo2, _, _ := newRerouteSUT(&fakeProvider{id: 7}, newP)
	repo2.byID[o.ID] = o
	got, err := st2.Reroute(context.Background(), o.ID, RerouteInput{ProviderID: 9, UpstreamProductID: "U"}, "a")
	require.NoError(t, err)
	require.Len(t, got.Fulfillment.SupplierAttempts, 1)
	assert.Equal(t, "UP_9", got.Fulfillment.SupplierAttempts[0].ProviderRef)
	assert.Empty(t, got.Fulfillment.RejectedProviderRef)
}

// An order whose quantity can't be sent is flagged stuck at placement (so it
// shows in the stuck list/dashboard), never dispatched.
func TestFulfillAPI_UnsendableQuantityFlagsStuck(t *testing.T) {
	fake := &fakeProvider{id: 7, res: provider.Result{Reference: "X"}}
	st, repo, _, _, _, _ := newSettlerSUT(fake, 24*time.Hour)
	o := settlerOrder(repo, func(o *Order) { o.Items[0].Amount = f64p(0.5) })

	got, err := st.svc.fulfillAPI(context.Background(), o.UserID, o, true)
	require.NoError(t, err)
	assert.Equal(t, 0, fake.calls)
	assert.Equal(t, OrderStatusProcessing, got.Status)
	assert.NotNil(t, got.Fulfillment.SupplierStuckAt)
	assert.Nil(t, got.Fulfillment.SupplierNextRetryAt)
}
