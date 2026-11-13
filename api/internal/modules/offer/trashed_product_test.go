package offer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

// liveRepo serves one live offer from ListLive.
type liveRepo struct {
	stubRepo
	offer *Offer
}

func (l *liveRepo) ListLive(context.Context, time.Time) ([]*Offer, error) {
	return []*Offer{l.offer}, nil
}

// The public offers list never advertises a deal on a trashed (or hidden)
// product — the storefront could not sell it.
func TestList_SkipsUnpurchasableProducts(t *testing.T) {
	now := time.Now()
	cases := map[string]struct {
		mutate func(p *product.Product)
		want   int
	}{
		"live":    {func(*product.Product) {}, 1},
		"hidden":  {func(p *product.Product) { p.Available = false }, 0},
		"trashed": {func(p *product.Product) { p.DeletedAt = &now }, 0},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p := coinProduct(false)
			c.mutate(p)
			repo := &liveRepo{offer: &Offer{ID: bson.NewObjectID(), ProductID: p.ID, DiscountType: DiscountPercent, DiscountValue: 10, Active: true}}
			h := NewHandler(repo, fakeProducts{p: p})

			rec := httptest.NewRecorder()
			h.List(rec, httptest.NewRequest(http.MethodGet, "/api/v1/offers", nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("code = %d", rec.Code)
			}
			if got := len(listBody(t, rec)); got != c.want {
				t.Fatalf("offers listed = %d, want %d", got, c.want)
			}
		})
	}
}
