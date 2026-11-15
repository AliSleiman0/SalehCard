package product

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The supplier sync job's per-link data (snapshot, loss pause, watch toggles)
// must survive an admin save of the links table.
func TestPlanSupplierLinks_KeepsSyncData(t *testing.T) {
	at := time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC)
	usd := 0.00013
	cur := &Product{
		SupplierLinksRev:     4,
		ActiveSupplierLinkID: "b",
		SupplierLinks: []SupplierLink{
			{ID: "a", Provider: 10, UpstreamID: "3258", Enabled: true, PriceSyncOff: true,
				PausedReason: PausedLoss, PausedAt: &at,
				Sync: &LinkSync{CheckedAt: at, Found: true, Available: true, Cost: 0.00013, CostUSD: &usd}},
			{ID: "b", Provider: 11, UpstreamID: "4410", Enabled: true, Sync: &LinkSync{CheckedAt: at, Found: true}},
		},
	}
	s := &ProductService{catalog: newLinkCatalog()}

	t.Run("same rows keep snapshot, pause and toggles", func(t *testing.T) {
		plan, err := s.planSupplierLinks(context.Background(), cur, &SupplierLinksInput{Rev: 4, Links: []SupplierLinkInput{
			{ID: "a", Provider: 10, UpstreamID: "3258", Enabled: true},
			{ID: "b", Provider: 11, UpstreamID: "4410", Enabled: true, Active: true},
		}})
		require.NoError(t, err)
		a := plan.Links[0]
		assert.Equal(t, PausedLoss, a.PausedReason)
		require.NotNil(t, a.Sync)
		assert.Equal(t, 0.00013, a.Sync.Cost)
		assert.True(t, a.PriceSyncOff, "an omitted toggle keeps the stored value")
		assert.NotNil(t, plan.Links[1].Sync)
	})

	t.Run("toggles are set from the editor", func(t *testing.T) {
		on, off := true, false
		plan, err := s.planSupplierLinks(context.Background(), cur, &SupplierLinksInput{Rev: 4, Links: []SupplierLinkInput{
			{ID: "a", Provider: 10, UpstreamID: "3258", Enabled: true, PriceSync: &on},
			{ID: "b", Provider: 11, UpstreamID: "4410", Enabled: true, Active: true, StatusSync: &off},
		}})
		require.NoError(t, err)
		assert.False(t, plan.Links[0].PriceSyncOff)
		assert.True(t, plan.Links[1].StatusSyncOff)
	})

	t.Run("an admin activating a paused link clears the pause", func(t *testing.T) {
		plan, err := s.planSupplierLinks(context.Background(), cur, &SupplierLinksInput{Rev: 4, Links: []SupplierLinkInput{
			{ID: "a", Provider: 10, UpstreamID: "3258", Enabled: true, Active: true},
			{ID: "b", Provider: 11, UpstreamID: "4410", Enabled: true},
		}})
		require.NoError(t, err)
		assert.Equal(t, "a", plan.ActiveID)
		assert.Empty(t, plan.Links[0].PausedReason)
		assert.Nil(t, plan.Links[0].PausedAt)
	})

	t.Run("a changed supplier product drops the old snapshot", func(t *testing.T) {
		plan, err := s.planSupplierLinks(context.Background(), cur, &SupplierLinksInput{Rev: 4, Links: []SupplierLinkInput{
			{ID: "a", Provider: 10, UpstreamID: "195", Enabled: true},
			{ID: "b", Provider: 11, UpstreamID: "4410", Enabled: true, Active: true},
		}})
		require.NoError(t, err)
		assert.Nil(t, plan.Links[0].Sync)
		assert.Empty(t, plan.Links[0].PausedReason)
	})
}

func TestSupplierLinkJSON_PositiveToggles(t *testing.T) {
	b, err := json.Marshal(SupplierLink{ID: "a", Provider: 10, UpstreamID: "1", Enabled: true, StatusSyncOff: true})
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Equal(t, true, m["priceSync"])
	assert.Equal(t, false, m["statusSync"])
	assert.Equal(t, "a", m["id"])
	_, leaked := m["priceSyncOff"]
	assert.False(t, leaked)
}

// A product's public JSON never carries the links (and so no costs).
func TestPublicProductJSON_NoSyncData(t *testing.T) {
	at := time.Now()
	p := Product{HiddenBySyncAt: &at, SupplierLinks: []SupplierLink{{ID: "a", Sync: &LinkSync{Cost: 1}}}}
	b, err := json.Marshal(p)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "supplierLinks")
	assert.NotContains(t, string(b), "hiddenBySync")
	assert.NotContains(t, string(b), "cost\":1")
}
