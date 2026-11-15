package main

import (
	"testing"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

func TestPlan(t *testing.T) {
	ip := func(n int) *int { return &n }
	now := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	created := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	link, reason := plan(candidate{FulfillmentProvider: ip(10), UpstreamProductID: " 3258 ", CreatedAt: created}, now)
	if reason != "" {
		t.Fatalf("mapped product skipped: %s", reason)
	}
	if link.Provider != 10 || link.UpstreamID != "3258" || !link.Enabled || link.ID == "" || !link.AddedAt.Equal(created) {
		t.Fatalf("link = %+v", link)
	}

	cases := []struct {
		name string
		c    candidate
	}{
		{"already linked", candidate{FulfillmentProvider: ip(10), UpstreamProductID: "1", SupplierLinks: []product.SupplierLink{{ID: "x"}}}},
		{"no supplier", candidate{}},
		{"zero supplier", candidate{FulfillmentProvider: ip(0), UpstreamProductID: "1"}},
		{"no upstream id", candidate{FulfillmentProvider: ip(10)}},
	}
	for _, c := range cases {
		if _, reason := plan(c.c, now); reason == "" {
			t.Errorf("%s: planned a link, want skip", c.name)
		}
	}
}
