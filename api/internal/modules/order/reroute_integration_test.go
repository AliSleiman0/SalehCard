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
		Archived: SupplierAttempt{Provider: &pid, UpstreamProductID: "364", ProviderRef: "search:ALFA:x", ReroutedAt: time.Now().UTC()},
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
