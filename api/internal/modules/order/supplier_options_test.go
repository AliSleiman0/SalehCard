package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

// linkedProduct registers a product with the given supplier links (the first
// one active) in the SUT's fake catalog and returns its id.
func linkedProduct(st *SupplierSettler, links ...product.SupplierLink) bson.ObjectID {
	id := bson.NewObjectID()
	p := &product.Product{ID: id, SupplierLinks: links}
	if len(links) > 0 {
		p.ActiveSupplierLinkID = links[0].ID
	}
	st.svc.products.(*fakeProductSvc).byID[id.Hex()] = p
	return id
}

// lastReroute is the summary of the newest supplier-reroute audit entry.
func lastReroute(rec *fakeRecorder) map[string]any {
	for i := len(rec.entries) - 1; i >= 0; i-- {
		if rec.entries[i].Action == audit.ActionOrderSupplierReroute {
			return rec.entries[i].Summary
		}
	}
	return nil
}

func link(id string, prov int, upstream string, enabled bool) product.SupplierLink {
	return product.SupplierLink{ID: id, Provider: prov, UpstreamID: upstream, Enabled: enabled}
}

func onProduct(pid bson.ObjectID, more ...func(*Order)) func(*Order) {
	return func(o *Order) {
		o.Items[0].ProductID = pid
		for _, m := range more {
			m(o)
		}
	}
}

func TestReroute_ByLink(t *testing.T) {
	ctx := context.Background()
	newP := &fakeProvider{id: 9, res: provider.Result{Reference: "N", Codes: []string{"C"}}}
	st, repo, _, rec := newRerouteSUT(&fakeProvider{id: 7}, newP)
	pid := linkedProduct(st, link("cur", 7, "364", true), link("alt", 9, "U-900", true), link("off", 9, "U-901", false))

	t.Run("a disabled link is refused", func(t *testing.T) {
		o := settlerOrder(repo, onProduct(pid, stuckAt))
		_, err := st.Reroute(ctx, o.ID, RerouteInput{LinkID: "off"}, "a")
		assert.Equal(t, "REROUTE_LINK_DISABLED", appCode(err))
	})

	t.Run("a link that is gone is refused", func(t *testing.T) {
		o := settlerOrder(repo, onProduct(pid, stuckAt))
		_, err := st.Reroute(ctx, o.ID, RerouteInput{LinkID: "nope"}, "a")
		assert.Equal(t, "REROUTE_LINK_UNKNOWN", appCode(err))
	})

	t.Run("the link's supplier product is used — nothing typed", func(t *testing.T) {
		o := settlerOrder(repo, onProduct(pid, stuckAt))
		got, err := st.Reroute(ctx, o.ID, RerouteInput{LinkID: "alt", ProviderID: 7, UpstreamProductID: "ignored"}, "a")
		require.NoError(t, err)
		assert.Equal(t, OrderStatusCompleted, got.Status)
		assert.Equal(t, "U-900", newP.lastIn.UpstreamID)
		assert.Equal(t, 9, *got.Items[0].FulfillmentProvider)
		assert.Equal(t, "alt", lastReroute(rec)["linkId"])
	})

	t.Run("sending to the same supplier again is allowed (new attempt, new key)", func(t *testing.T) {
		oldP := &fakeProvider{id: 7, res: provider.Result{Reference: "R2"}}
		st2, repo2, _, _ := newRerouteSUT(oldP, &fakeProvider{id: 9})
		pid2 := linkedProduct(st2, link("cur", 7, "364", true))
		o := settlerOrder(repo2, onProduct(pid2, stuckAt))
		got, err := st2.Reroute(ctx, o.ID, RerouteInput{LinkID: "cur"}, "a")
		require.NoError(t, err)
		assert.Equal(t, 1, oldP.calls)
		assert.NotEqual(t, o.ID.Hex(), oldP.lastIn.OrderUUID)
		assert.Contains(t, got.Fulfillment.StatusTimeline[len(got.Fulfillment.StatusTimeline)-2].Note, "sent again")
	})
}

func TestSupplierOptions(t *testing.T) {
	ctx := context.Background()
	ip := func(n int) *int { return &n }
	target := &catalogProvider{balance: 0.5, products: []provider.CatalogProduct{
		{UpstreamID: "3258", Name: "SOUL STAR", Available: true, Price: 0.0001, Currency: "USD", QtyMin: ip(10000)},
		{UpstreamID: "77", Name: "Gone soon", Available: false, Price: 0.0001, Currency: "USD"},
	}}
	target.id = 9
	oldP := &fakeProvider{id: 7}
	st, repo, _, _ := newRerouteSUT(oldP, &fakeProvider{id: 99})
	reg := provider.NewRegistry(oldP, target)
	st.svc.providers = reg
	st.WithProbes(provider.NewProbes(reg, provider.ProbesConfig{}))

	pid := linkedProduct(st,
		link("cur", 7, "364", true), link("ok", 9, "3258", true), link("bad", 9, "1045", true),
		link("off", 9, "77", false), link("none", 12, "5", true))
	amount := 10000.0
	o := settlerOrder(repo, onProduct(pid, stuckAt, func(o *Order) {
		o.Items[0].Amount = &amount
		o.Items[0].Price = 1.23
		o.Fulfillment.SupplierAttempts = []SupplierAttempt{{Provider: ip(9), UpstreamProductID: "1045"}}
	}))

	got, err := st.SupplierOptions(ctx, o.ID, false)
	require.NoError(t, err)
	assert.True(t, got.CanSend)
	assert.False(t, got.NeedsAck, "stuck and ref-less: nothing in flight")
	assert.Equal(t, 10000, got.Qty)
	assert.InDelta(t, 1.23, got.LinePrice, 1e-9)
	require.Len(t, got.Options, 5)

	byID := map[string]SupplierOption{}
	for _, op := range got.Options {
		byID[op.LinkID] = op
	}
	assert.True(t, byID["cur"].Current)
	assert.True(t, byID["cur"].Active)
	assert.Equal(t, "not_checked", byID["cur"].CatalogStatus, "a supplier that can't list products isn't checked")
	assert.Empty(t, byID["cur"].Blocked)

	okOpt := byID["ok"]
	assert.Equal(t, "ok", okOpt.CatalogStatus)
	assert.Equal(t, "SOUL STAR", okOpt.Name)
	require.NotNil(t, okOpt.Cost)
	assert.InDelta(t, 1.0, *okOpt.Cost, 1e-9, "unit price × the units this order sends")
	assert.True(t, okOpt.QtyOK)
	assert.True(t, okOpt.LowBalance, "0.50 balance < 1.00 cost")
	assert.Nil(t, okOpt.Balance, "balance hidden without suppliers.view")
	assert.Empty(t, okOpt.Blocked, "low balance is a warning, not a block")

	assert.Equal(t, "unknown", byID["bad"].CatalogStatus)
	assert.NotEmpty(t, byID["bad"].Blocked)
	assert.True(t, byID["bad"].Tried)
	assert.Contains(t, byID["off"].Blocked, "unavailable")
	assert.Contains(t, byID["none"].Blocked, "isn't connected")

	withBal, err := st.SupplierOptions(ctx, o.ID, true)
	require.NoError(t, err)
	for _, op := range withBal.Options {
		if op.LinkID == "ok" {
			require.NotNil(t, op.Balance)
			assert.InDelta(t, 0.5, *op.Balance, 1e-9)
		}
	}

	t.Run("not eligible orders explain why", func(t *testing.T) {
		done := settlerOrder(repo, onProduct(pid, func(o *Order) { o.Status = OrderStatusCompleted }))
		got, err := st.SupplierOptions(ctx, done.ID, false)
		require.NoError(t, err)
		assert.False(t, got.CanSend)
		assert.Equal(t, "REROUTE_NOT_PROCESSING", got.Reason)
	})

	t.Run("an order the supplier may still deliver needs the OK", func(t *testing.T) {
		live := settlerOrder(repo, onProduct(pid, func(o *Order) { o.Fulfillment.ProviderRef = "UP" }))
		got, err := st.SupplierOptions(ctx, live.ID, false)
		require.NoError(t, err)
		assert.True(t, got.CanSend)
		assert.True(t, got.NeedsAck)
	})
}

func TestBulkReroute(t *testing.T) {
	ctx := context.Background()

	t.Run("limits", func(t *testing.T) {
		st, _, _, _ := newRerouteSUT(&fakeProvider{id: 7}, &fakeProvider{id: 9})
		ids := make([]string, MaxBulkReroute+1)
		for i := range ids {
			ids[i] = bson.NewObjectID().Hex()
		}
		_, err := st.BulkReroute(ctx, BulkRerouteInput{IDs: ids}, "a")
		assert.Equal(t, "REROUTE_BULK_LIMIT", appCode(err))
		_, err = st.BulkReroute(ctx, BulkRerouteInput{}, "a")
		assert.Equal(t, "REROUTE_BULK_EMPTY", appCode(err))
		_, err = st.BulkReroute(ctx, BulkRerouteInput{IDs: ids[:1], ProviderID: 42}, "a")
		assert.Equal(t, "REROUTE_UNKNOWN_SUPPLIER", appCode(err))
	})

	t.Run("each order goes to its first usable link; per-order outcomes", func(t *testing.T) {
		newP := &fakeProvider{id: 9, res: provider.Result{Reference: "N", Codes: []string{"C"}}}
		st, repo, _, rec := newRerouteSUT(&fakeProvider{id: 7}, newP)
		// Active = the current supplier product → skipped; next = 9/U-900.
		pid := linkedProduct(st, link("cur", 7, "364", true), link("off", 9, "U-899", false), link("alt", 9, "U-900", true))
		onlyCur := linkedProduct(st, link("cur", 7, "364", true))

		moved := settlerOrder(repo, onProduct(pid, stuckAt))
		atSupplier := settlerOrder(repo, onProduct(pid, func(o *Order) { o.Fulfillment.ProviderRef = "UP" }))
		noLink := settlerOrder(repo, onProduct(onlyCur, stuckAt))
		done := settlerOrder(repo, onProduct(pid, func(o *Order) { o.Status = OrderStatusRefunded }))

		_, err := st.BulkReroute(ctx, BulkRerouteInput{IDs: []string{
			moved.ID.Hex(), atSupplier.ID.Hex(), noLink.ID.Hex(), done.ID.Hex(), "zz", moved.ID.Hex(),
		}}, "a")
		require.Error(t, err, "6 ids is over the limit even with a duplicate")

		res, err := st.BulkReroute(ctx, BulkRerouteInput{IDs: []string{
			moved.ID.Hex(), atSupplier.ID.Hex(), noLink.ID.Hex(), done.ID.Hex(), "zz",
		}}, "a")
		require.NoError(t, err)
		status := []string{}
		for _, r := range res {
			status = append(status, r.Status)
		}
		assert.Equal(t, []string{"completed", "needs_ack", "no_link", "not_eligible", "error"}, status)
		assert.Equal(t, 9, res[0].Provider)
		assert.Equal(t, "U-900", res[0].UpstreamID)
		assert.Equal(t, 1, newP.calls, "only the eligible order is sent")
		assert.Equal(t, true, lastReroute(rec)["bulk"])

		// With the OK, the at-supplier order goes too.
		res, err = st.BulkReroute(ctx, BulkRerouteInput{IDs: []string{atSupplier.ID.Hex()}, AcknowledgeAtSupplier: true}, "a")
		require.NoError(t, err)
		assert.Equal(t, "completed", res[0].Status)
	})

	t.Run("a chosen supplier may re-send to the current product; a refusal is reported", func(t *testing.T) {
		oldP := &fakeProvider{id: 7, err: errors.New("invalid player id")}
		st, repo, _, _ := newRerouteSUT(oldP, &fakeProvider{id: 9})
		pid := linkedProduct(st, link("cur", 7, "364", true))
		o := settlerOrder(repo, onProduct(pid, stuckAt))
		res, err := st.BulkReroute(ctx, BulkRerouteInput{IDs: []string{o.ID.Hex()}, ProviderID: 7}, "a")
		require.NoError(t, err)
		assert.Equal(t, "refused", res[0].Status)
		assert.Equal(t, 1, oldP.calls)
		assert.NotNil(t, repo.byID[o.ID].Fulfillment.SupplierStuckAt)
		assert.WithinDuration(t, time.Now(), *repo.byID[o.ID].Fulfillment.SupplierAttemptAt, time.Minute)
	})
}
