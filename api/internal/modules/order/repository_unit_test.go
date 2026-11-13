package order

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// get returns the value for key in a bson.D, and whether it was present.
func get(d bson.D, key string) (any, bool) {
	for _, e := range d {
		if e.Key == key {
			return e.Value, true
		}
	}
	return nil, false
}

func TestOrderFilter_build(t *testing.T) {
	t.Run("empty filter matches everything", func(t *testing.T) {
		assert.Empty(t, OrderFilter{}.build())
	})

	t.Run("status / paymentMethod / fulfillmentType map to equality", func(t *testing.T) {
		d := OrderFilter{
			Status:          OrderStatusCompleted,
			PaymentMethod:   PaymentMethodCard,
			FulfillmentType: "code",
		}.build()
		v, ok := get(d, "status")
		require.True(t, ok)
		assert.Equal(t, OrderStatusCompleted, v)
		v, ok = get(d, "paymentMethod")
		require.True(t, ok)
		assert.Equal(t, PaymentMethodCard, v)
		v, ok = get(d, "items.fulfillmentType")
		require.True(t, ok)
		assert.Equal(t, "code", v)
	})

	t.Run("hex search sets _id and skips $or", func(t *testing.T) {
		id := bson.NewObjectID()
		d := OrderFilter{OrderID: &id}.build()
		v, ok := get(d, "_id")
		require.True(t, ok)
		assert.Equal(t, id, v)
		_, ok = get(d, "$or")
		assert.False(t, ok)
	})

	t.Run("non-hex search builds $or over deliveredCode, item codes and userId", func(t *testing.T) {
		uid := bson.NewObjectID()
		d := OrderFilter{SearchRaw: "DEMO-0001", UserIDs: []bson.ObjectID{uid}}.build()
		v, ok := get(d, "$or")
		require.True(t, ok)
		or, ok := v.(bson.A)
		require.True(t, ok)
		assert.Len(t, or, 5) // deliveredCode + items.codes + playerId + field values + userId
	})

	t.Run("search with no matching users still matches by code / player ID / field", func(t *testing.T) {
		d := OrderFilter{SearchRaw: "DEMO-0001", UserIDs: []bson.ObjectID{}}.build()
		v, _ := get(d, "$or")
		or, _ := v.(bson.A)
		assert.Len(t, or, 4) // deliveredCode + items.codes + playerId + field values
	})

	t.Run("created window maps to a half-open createdAt range", func(t *testing.T) {
		from := time.Date(2026, 11, 13, 22, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 0, 1)
		d := OrderFilter{CreatedFrom: &from, CreatedTo: &to}.build()
		v, ok := get(d, "createdAt")
		require.True(t, ok)
		assert.Equal(t, bson.D{{Key: "$gte", Value: from}, {Key: "$lt", Value: to}}, v)
	})

	t.Run("supplierStuck matches the dashboard stuckUpstream predicate and overrides status", func(t *testing.T) {
		d := OrderFilter{Status: OrderStatusFailed, SupplierStuck: true}.build()
		n := 0
		for _, e := range d {
			if e.Key == "status" {
				n++
				assert.Equal(t, OrderStatusProcessing, e.Value)
			}
		}
		assert.Equal(t, 1, n, "exactly one status key")
		v, ok := get(d, "fulfillment.supplierStuckAt")
		require.True(t, ok)
		assert.Equal(t, bson.D{{Key: "$exists", Value: true}, {Key: "$ne", Value: nil}}, v)
	})

	t.Run("short hex id also matches the order ref", func(t *testing.T) {
		d := OrderFilter{SearchRaw: "2F1E2818"}.build()
		v, _ := get(d, "$or")
		or, _ := v.(bson.A)
		require.Len(t, or, 5)
		assert.Equal(t, bson.D{{Key: "ref", Value: bson.D{{Key: "$regex", Value: "2f1e2818"}}}}, or[4])
	})
}

func TestIsShortRef(t *testing.T) {
	assert.True(t, isShortRef("2f1e2818"))
	assert.True(t, isShortRef("2F1E28"))
	assert.False(t, isShortRef("abcd"))                     // 4 hex: too loose
	assert.False(t, isShortRef("abc12"))                    // too short
	assert.False(t, isShortRef("6af5b2f431f548082f1e2818")) // full id → exact path
	assert.False(t, isShortRef("1234-567"))
	assert.False(t, isShortRef("player9"))
}

func TestRevenueBuckets(t *testing.T) {
	now := time.Date(2026, time.June, 27, 12, 0, 0, 0, time.UTC)
	loc := time.UTC

	t.Run("daily: 14 contiguous day buckets ending today", func(t *testing.T) {
		fmtStr, buckets := revenueBuckets(now, "daily", loc)
		assert.Equal(t, "%Y-%m-%d", fmtStr)
		require.Len(t, buckets, 14)
		assert.Equal(t, "Jun 27", buckets[13].label)
		assert.Equal(t, time.Date(2026, time.June, 27, 0, 0, 0, 0, loc), buckets[13].start)
		assert.Equal(t, time.Date(2026, time.June, 28, 0, 0, 0, 0, loc), buckets[13].end)
		assert.Equal(t, "Jun 14", buckets[0].label)
		assert.Equal(t, time.Date(2026, time.June, 14, 0, 0, 0, 0, loc), buckets[0].start)
	})

	t.Run("monthly: 12 month buckets ending this month", func(t *testing.T) {
		fmtStr, buckets := revenueBuckets(now, "monthly", loc)
		assert.Equal(t, "%Y-%m", fmtStr)
		require.Len(t, buckets, 12)
		assert.Equal(t, "Jun", buckets[11].label)
		assert.Equal(t, time.Date(2026, time.June, 1, 0, 0, 0, 0, loc), buckets[11].start)
		assert.Equal(t, time.Date(2026, time.July, 1, 0, 0, 0, 0, loc), buckets[11].end)
	})

	t.Run("weekly: 12 portable day-grouped week buckets, W-prefixed labels", func(t *testing.T) {
		fmtStr, buckets := revenueBuckets(now, "weekly", loc)
		assert.Equal(t, "%Y-%m-%d", fmtStr) // portable: no ISO-week format codes
		require.Len(t, buckets, 12)
		assert.Equal(t, byte('W'), buckets[11].label[0])
		// Each bucket is a 7-day window starting on a Monday.
		assert.Equal(t, time.Monday, buckets[11].start.Weekday())
		assert.Equal(t, 7*24*time.Hour, buckets[11].end.Sub(buckets[11].start))
	})

	t.Run("unknown range falls back to daily", func(t *testing.T) {
		_, buckets := revenueBuckets(now, "nonsense", loc)
		assert.Len(t, buckets, 14)
	})
}

func TestBusinessDay(t *testing.T) {
	saved := businessLocation
	t.Cleanup(func() { businessLocation = saved })
	beirut, err := time.LoadLocation("Asia/Beirut")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	businessLocation = beirut
	// 23:30 UTC on Nov 13 is already Nov 14 in Beirut (UTC+2).
	start, end := businessDay(time.Date(2026, 11, 13, 23, 30, 0, 0, time.UTC))
	assert.Equal(t, time.Date(2026, 11, 13, 22, 0, 0, 0, time.UTC), start.UTC())
	assert.Equal(t, time.Date(2026, 11, 14, 22, 0, 0, 0, time.UTC), end.UTC())
}
