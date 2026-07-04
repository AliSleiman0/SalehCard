package server

import (
	"context"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/offer"
	product "github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

// offerCatalog adapts the offer module to the product package's OfferLookup port.
// It lives here (not in product) because it must import both packages, and product
// must not import offer (offer already imports product — the reverse would cycle).
type offerCatalog struct {
	repo offer.Repository
}

// LiveDiscountsFor returns the winning live discount per requested product id.
// It reads every live offer once (ListLive, sorted sortOrder ASC, createdAt DESC)
// and keeps the FIRST offer seen per product — i.e. the lowest sortOrder, the same
// winner FindLiveByProduct and GET /api/v1/offers resolve to.
func (c offerCatalog) LiveDiscountsFor(ctx context.Context, ids []string, now time.Time) (map[string]product.Discount, error) {
	live, err := c.repo.ListLive(ctx, now)
	if err != nil {
		return nil, err
	}
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	out := make(map[string]product.Discount)
	for _, o := range live {
		pid := o.ProductID.Hex()
		if _, ok := want[pid]; !ok {
			continue
		}
		if _, seen := out[pid]; seen {
			continue // first (lowest sortOrder) wins
		}
		out[pid] = product.Discount{
			Type:   string(o.DiscountType),
			Value:  o.DiscountValue,
			EndsAt: o.EndsAt,
		}
	}
	return out, nil
}
