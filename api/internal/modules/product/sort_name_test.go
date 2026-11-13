package product

import (
	"strings"
	"testing"
	"time"

	"github.com/AliSleiman0/salehcard/api/pkg/pagination"

	"github.com/stretchr/testify/assert"
)

func TestNaturalLess(t *testing.T) {
	assert.True(t, naturalLess("apple", "Bulala"), "case-insensitive")
	assert.True(t, naturalLess("JAWAKER 150", "JAWAKER 400"))
	assert.True(t, naturalLess("JAWAKER 400", "jawaker 1000"), "numeric, not byte-wise")
	assert.False(t, naturalLess("JAWAKER 1,000", "JAWAKER 1000"), "commas inside numbers ignored")
	assert.False(t, naturalLess("JAWAKER 1000", "JAWAKER 1,000"))
	assert.True(t, naturalLess("2$ ALFA", "10$ ALFA"))
	assert.True(t, naturalLess("abc", "abcd"))
}

func TestSortProductsByName(t *testing.T) {
	ps := []Product{
		{Title: I18nString{En: "Yalla Ludo"}},
		{Title: I18nString{En: "", Ar: "باقة"}},
		{Title: I18nString{En: "bigo live 1000"}},
		{Title: I18nString{En: "Bigo Live 200"}},
		{Title: I18nString{}},
		{Title: I18nString{En: "(Sigma Plus 1 Year"}},
	}
	sortProductsByName(ps)
	got := []string{}
	for i := range ps {
		got = append(got, sortKey(&ps[i]))
	}
	assert.Equal(t, []string{"Bigo Live 200", "bigo live 1000", "Sigma Plus 1 Year", "Yalla Ludo", "باقة", ""}, got)
}

func TestNameCache(t *testing.T) {
	invalidateNameCache()
	t.Cleanup(invalidateNameCache)
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	sorted := []Product{{Title: I18nString{En: "A"}, Variants: []Variant{{Price: 1}}}}

	nameCachePut("k", sorted, now)
	got, ok := nameCacheGet("k", now.Add(10*time.Second))
	assert.True(t, ok, "fresh within TTL")
	assert.Len(t, got, 1)
	_, ok = nameCacheGet("k", now.Add(nameCacheTTL+time.Second))
	assert.False(t, ok, "expired after TTL")

	nameCachePut("k", sorted, now)
	invalidateNameCache()
	_, ok = nameCacheGet("k", now)
	assert.False(t, ok, "cleared by a write")
}

func TestNameCacheEvictsOldest(t *testing.T) {
	invalidateNameCache()
	t.Cleanup(invalidateNameCache)
	base := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	for i := 0; i < nameCacheMax; i++ {
		nameCachePut(string(rune('a'+i%26))+strings.Repeat("x", i), nil, base.Add(time.Duration(i)*time.Millisecond))
	}
	nameCachePut("newest", nil, base.Add(time.Second))
	_, ok := nameCacheGet("a", base.Add(time.Second)) // the i=0 key, oldest
	assert.False(t, ok)
	_, ok = nameCacheGet("newest", base.Add(time.Second))
	assert.True(t, ok)
}

// A page handed to the service is enriched in place; that must never reach
// the cached set shared by other callers.
func TestPageOfClonesRows(t *testing.T) {
	cached := []Product{{Title: I18nString{En: "A"}, Variants: []Variant{{Price: 10}}}}
	page, total, err := pageOf(cached, pagination.Params{Page: 1, Limit: 20})
	assert.NoError(t, err)
	assert.EqualValues(t, 1, total)
	offer := 5.0
	page[0].Variants[0].OfferPrice = &offer
	page[0].Variants[0].Price = 99
	assert.Nil(t, cached[0].Variants[0].OfferPrice)
	assert.Equal(t, 10.0, cached[0].Variants[0].Price)
}
