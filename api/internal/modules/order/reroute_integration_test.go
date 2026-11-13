package order

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// TestClaimSupplierReroute_Mongo exercises the real guarded FindOneAndUpdate
// (filter on status + attempt + ref, positional item set, unsets, history
// push) against a throwaway database. Needs MONGO_TEST_URI.
func TestClaimSupplierReroute_Mongo(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	defer func() { _ = client.Disconnect(ctx) }()
	db := client.Database("salehcard_test_reroute_" + bson.NewObjectID().Hex())
	defer func() { _ = db.Drop(ctx) }()
	repo := NewMongoRepository(db.Collection("orders"))

	pid := 7
	stuck := time.Now().UTC()
	retry := time.Now().UTC()
	o := &Order{
		ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Status: OrderStatusProcessing,
		PaymentMethod: PaymentMethodWallet, Currency: "USD", Total: 5, CreatedAt: time.Now().UTC(),
		Items: []OrderItem{
			{ProductID: bson.NewObjectID(), FulfillmentMode: "manual_operator", Qty: 1},
			{ProductID: bson.NewObjectID(), FulfillmentMode: "api", FulfillmentProvider: &pid, UpstreamProductID: "364", Qty: 1},
		},
		Fulfillment: Fulfillment{
			ProviderRef: "search:ALFA:x", SupplierStuckAt: &stuck,
			SupplierRetryCount: 3, SupplierNextRetryAt: &retry,
			StatusTimeline: []TimelineEvent{},
		},
	}
	_, err = db.Collection("orders").InsertOne(ctx, o)
	require.NoError(t, err)

	claim := RerouteClaim{
		OrderID: o.ID, ItemIndex: 1, ExpectedAttempt: 0, ExpectedRef: "search:ALFA:x",
		NewProvider: 9, NewUpstreamID: "U-9", NewOrderUUID: bson.NewObjectID().Hex(),
		Archived: &SupplierAttempt{Provider: &pid, UpstreamProductID: "364", ProviderRef: "search:ALFA:x", ReroutedAt: time.Now().UTC()},
		Event:    TimelineEvent{Status: "processing", Note: "rerouted", At: time.Now().UTC()},
	}
	got, err := repo.ClaimSupplierReroute(ctx, claim)
	require.NoError(t, err)

	assert.Equal(t, 9, *got.Items[1].FulfillmentProvider)
	assert.Equal(t, "U-9", got.Items[1].UpstreamProductID)
	assert.Equal(t, "manual_operator", got.Items[0].FulfillmentMode, "other lines untouched")
	assert.Nil(t, got.Items[0].FulfillmentProvider)
	assert.Equal(t, "", got.Fulfillment.ProviderRef, "old ref no longer polled")
	assert.Nil(t, got.Fulfillment.SupplierStuckAt)
	require.NotNil(t, got.Fulfillment.SupplierNextRetryAt, "fallback retry clock (overwritten by the dispatch outcome)")
	assert.Equal(t, 0, got.Fulfillment.SupplierRetryCount)
	assert.Equal(t, 1, got.Fulfillment.SupplierAttempt)
	assert.Equal(t, claim.NewOrderUUID, got.Fulfillment.SupplierOrderUUID)
	require.NotNil(t, got.Fulfillment.SupplierAttemptAt)
	require.Len(t, got.Fulfillment.SupplierAttempts, 1)
	assert.Equal(t, "search:ALFA:x", got.Fulfillment.SupplierAttempts[0].ProviderRef)
	assert.Len(t, got.Fulfillment.StatusTimeline, 1)

	// The same (now stale) claim loses: attempt and ref both changed.
	_, err = repo.ClaimSupplierReroute(ctx, claim)
	assert.ErrorIs(t, err, apperrors.ErrConflict)

	// Not pollable any more (old ref cleared).
	pollable, err := repo.ListSupplierPollable(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, pollable)

	// A second reroute from attempt 1 with no ref matches the $or branch.
	claim2 := claim
	claim2.ExpectedAttempt, claim2.ExpectedRef, claim2.NewOrderUUID = 1, "", bson.NewObjectID().Hex()
	got, err = repo.ClaimSupplierReroute(ctx, claim2)
	require.NoError(t, err)
	assert.Equal(t, 2, got.Fulfillment.SupplierAttempt)
	assert.Len(t, got.Fulfillment.SupplierAttempts, 2)

	// Missing order → ErrNotFound.
	claim2.OrderID = bson.NewObjectID()
	_, err = repo.ClaimSupplierReroute(ctx, claim2)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

// TestSupplierManualPaths_Mongo covers the real filters added for sending a
// manual order to a supplier: the manual-mode pin + mode flip (including a
// legacy mode-less line), the api pin rejecting a manual line, the
// at-supplier completion guard, and MarkSupplierRejected. Needs MONGO_TEST_URI.
func TestSupplierManualPaths_Mongo(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	defer func() { _ = client.Disconnect(ctx) }()
	db := client.Database("salehcard_test_manual_" + bson.NewObjectID().Hex())
	defer func() { _ = db.Drop(ctx) }()
	repo := NewMongoRepository(db.Collection("orders"))

	insert := func(mode string) *Order {
		amt := 10000.0
		o := &Order{
			ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Status: OrderStatusProcessing,
			PaymentMethod: PaymentMethodWallet, Currency: "USD", Total: 1.23, CreatedAt: time.Now().UTC(),
			Items:       []OrderItem{{ProductID: bson.NewObjectID(), FulfillmentType: "account_credit", FulfillmentMode: mode, Qty: 1, Amount: &amt}},
			Fulfillment: Fulfillment{StatusTimeline: []TimelineEvent{}},
		}
		_, err := db.Collection("orders").InsertOne(ctx, o)
		require.NoError(t, err)
		return o
	}
	claimFor := func(o *Order, fromManual bool) RerouteClaim {
		return RerouteClaim{
			OrderID: o.ID, ItemIndex: 0, FromManual: fromManual,
			NewProvider: 10, NewUpstreamID: "2", NewOrderUUID: bson.NewObjectID().Hex(),
			Event: TimelineEvent{Status: "processing", Note: "sent to supplier", At: time.Now().UTC()},
		}
	}

	// A manual line claimed as an api line loses (mode pin).
	m := insert("manual_operator")
	_, err = repo.ClaimSupplierReroute(ctx, claimFor(m, false))
	assert.ErrorIs(t, err, apperrors.ErrConflict)

	// FromManual: the line flips to api, no attempt archived.
	got, err := repo.ClaimSupplierReroute(ctx, claimFor(m, true))
	require.NoError(t, err)
	assert.Equal(t, "api", got.Items[0].FulfillmentMode)
	assert.Equal(t, 10, *got.Items[0].FulfillmentProvider)
	assert.Empty(t, got.Fulfillment.SupplierAttempts)
	assert.Equal(t, 1, got.Fulfillment.SupplierAttempt)

	// A legacy line with no stored mode also matches the manual pin.
	legacy := insert("")
	got, err = repo.ClaimSupplierReroute(ctx, claimFor(legacy, true))
	require.NoError(t, err)
	assert.Equal(t, "api", got.Items[0].FulfillmentMode)

	// At supplier (retry clock armed by the claim, not stuck): completion refused.
	_, err = repo.CompleteUnlessAtSupplier(ctx, m.ID, TimelineEvent{Status: "completed", At: time.Now().UTC()}, nil)
	assert.ErrorIs(t, err, apperrors.ErrConflict)

	// Rejected → flagged stuck, ref + retry clock dropped; then completion is allowed.
	require.NoError(t, repo.SetSupplierRef(ctx, m.ID, 1, "UP_1", TimelineEvent{Status: "processing", At: time.Now().UTC()}))
	applied, err := repo.MarkSupplierRejected(ctx, m.ID, 1, TimelineEvent{Status: "stuck_upstream", At: time.Now().UTC()})
	require.NoError(t, err)
	assert.True(t, applied)
	cur, err := repo.FindByID(ctx, m.ID)
	require.NoError(t, err)
	assert.NotNil(t, cur.Fulfillment.SupplierStuckAt)
	assert.Equal(t, "", cur.Fulfillment.ProviderRef)
	assert.Nil(t, cur.Fulfillment.SupplierNextRetryAt)
	pollable, err := repo.ListSupplierPollable(ctx, 10)
	require.NoError(t, err)
	for _, p := range pollable {
		assert.NotEqual(t, m.ID, p.ID, "a rejected attempt is not polled again")
	}
	// Wrong attempt → not applied.
	applied, err = repo.MarkSupplierRejected(ctx, m.ID, 5, TimelineEvent{Status: "x", At: time.Now().UTC()})
	require.NoError(t, err)
	assert.False(t, applied)

	before, err := repo.CompleteUnlessAtSupplier(ctx, m.ID, TimelineEvent{Status: "completed", At: time.Now().UTC()}, nil)
	require.NoError(t, err)
	assert.Equal(t, OrderStatusProcessing, before.Status)

	// A plain manual order (never sent) completes through the guard.
	plain := insert("manual_operator")
	_, err = repo.CompleteUnlessAtSupplier(ctx, plain.ID, TimelineEvent{Status: "completed", At: time.Now().UTC()}, nil)
	require.NoError(t, err)
}
