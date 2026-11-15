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

// TestAutoFallbackClaim_Mongo checks the automatic-fallback claim against a
// real Mongo: a pending order can be claimed (placement), the give-up clock is
// kept, the attempt is flagged auto, an ambiguous attempt can't be claimed
// automatically, a BumpSupplierRetry(ambiguous) sticks, and an admin claim
// clears both flags. Needs MONGO_TEST_URI.
func TestAutoFallbackClaim_Mongo(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	defer func() { _ = client.Disconnect(ctx) }()
	db := client.Database("salehcard_test_fallback_" + bson.NewObjectID().Hex())
	defer func() { _ = db.Drop(ctx) }()
	repo := NewMongoRepository(db.Collection("orders"))

	pid := 7
	seed := func(status OrderStatus) *Order {
		o := &Order{
			ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Status: status,
			PaymentMethod: PaymentMethodWallet, Currency: "USD", Total: 5, CreatedAt: time.Now().UTC(),
			Items:       []OrderItem{{ProductID: bson.NewObjectID(), FulfillmentMode: "api", FulfillmentProvider: &pid, UpstreamProductID: "364", Qty: 1}},
			Fulfillment: Fulfillment{StatusTimeline: []TimelineEvent{}},
		}
		_, err := db.Collection("orders").InsertOne(ctx, o)
		require.NoError(t, err)
		return o
	}
	auto := func(o *Order, attempt int, pending bool) RerouteClaim {
		return RerouteClaim{
			OrderID: o.ID, ItemIndex: 0, ExpectedAttempt: attempt,
			NewProvider: 9, NewUpstreamID: "U-9", NewOrderUUID: bson.NewObjectID().Hex(),
			Archived: &SupplierAttempt{Provider: &pid, UpstreamProductID: "364", ReroutedAt: time.Now().UTC(), Auto: true, Reason: "balance"},
			Event:    TimelineEvent{Status: "processing", Note: "auto", At: time.Now().UTC()},
			Auto:     true, AllowPending: pending, KeepClock: true,
		}
	}

	t.Run("placement claim on a pending order", func(t *testing.T) {
		o := seed(OrderStatusPending)
		// Without AllowPending the pending order doesn't match.
		_, err := repo.ClaimSupplierReroute(ctx, auto(o, 0, false))
		assert.ErrorIs(t, err, apperrors.ErrConflict)

		got, err := repo.ClaimSupplierReroute(ctx, auto(o, 0, true))
		require.NoError(t, err)
		assert.Equal(t, OrderStatusPending, got.Status, "a claim never changes status")
		assert.True(t, got.Fulfillment.SupplierAttemptAuto)
		assert.Nil(t, got.Fulfillment.SupplierAttemptAt, "KeepClock: no give-up clock restart")
		assert.Equal(t, 1, got.Fulfillment.SupplierAttempt)
		require.Len(t, got.Fulfillment.SupplierAttempts, 1)
		assert.True(t, got.Fulfillment.SupplierAttempts[0].Auto)
		assert.Equal(t, "balance", got.Fulfillment.SupplierAttempts[0].Reason)
	})

	t.Run("an ambiguous attempt is never claimed automatically", func(t *testing.T) {
		o := seed(OrderStatusProcessing)
		require.NoError(t, repo.BumpSupplierRetry(ctx, o.ID, 0, 1, time.Now().UTC(), true))
		// A later non-ambiguous bump doesn't clear it.
		require.NoError(t, repo.BumpSupplierRetry(ctx, o.ID, 0, 2, time.Now().UTC(), false))
		cur, err := repo.FindByID(ctx, o.ID)
		require.NoError(t, err)
		assert.True(t, cur.Fulfillment.SupplierAmbiguous)

		_, err = repo.ClaimSupplierReroute(ctx, auto(o, 0, false))
		assert.ErrorIs(t, err, apperrors.ErrConflict)

		// An admin claim (the acknowledged re-send) still works and starts a
		// clean, admin-routed attempt.
		admin := auto(o, 0, false)
		admin.Auto, admin.KeepClock = false, false
		got, err := repo.ClaimSupplierReroute(ctx, admin)
		require.NoError(t, err)
		assert.False(t, got.Fulfillment.SupplierAmbiguous)
		assert.False(t, got.Fulfillment.SupplierAttemptAuto)
		assert.NotNil(t, got.Fulfillment.SupplierAttemptAt)
		assert.True(t, adminRouted(got))
	})

	t.Run("an admin claim clears the auto flag of an automatic attempt", func(t *testing.T) {
		o := seed(OrderStatusProcessing)
		got, err := repo.ClaimSupplierReroute(ctx, auto(o, 0, false))
		require.NoError(t, err)
		require.True(t, got.Fulfillment.SupplierAttemptAuto)
		admin := auto(got, 1, false)
		admin.Auto, admin.KeepClock = false, false
		got, err = repo.ClaimSupplierReroute(ctx, admin)
		require.NoError(t, err)
		assert.False(t, got.Fulfillment.SupplierAttemptAuto)
		assert.Equal(t, 2, got.Fulfillment.SupplierAttempt)
	})
}
