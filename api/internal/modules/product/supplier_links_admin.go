package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// supplierLinksAPI serves the product editor's supplier-links helpers (the
// products domain, so an editor doesn't need suppliers.view):
//   - POST /products/supplier-links/quote: live price / availability / qty
//     limits for each link row (saved or not);
//   - GET  /products/supplier-search?q=: one product name searched across
//     every connected supplier (the old admin's "Multi-link"), so linking
//     never needs a typed supplier id.
type supplierLinksAPI struct {
	catalog   SupplierCatalog
	providers []FulfillmentProviderOption
}

const (
	maxQuoteRows       = 12
	searchPerSupplier  = 25
	searchMinQueryLen  = 2
	supplierCallBudget = 15 * time.Second
)

// LinkQuote is the live view of one supplier link.
type LinkQuote struct {
	Provider   int    `json:"provider"`
	UpstreamID string `json:"upstreamId"`
	// Status: "ok" (found in the catalog), "unknown" (catalog loaded, id not
	// in it), "unavailable" (catalog couldn't load), "not_checked" (the
	// supplier can't list its catalog or isn't configured).
	Status    string     `json:"status"`
	Name      string     `json:"name,omitempty"`
	Price     float64    `json:"price,omitempty"`
	Currency  string     `json:"currency,omitempty"`
	Available bool       `json:"available"`
	QtyMin    *int       `json:"qtyMin,omitempty"`
	QtyMax    *int       `json:"qtyMax,omitempty"`
	QtyValues []string   `json:"qtyValues,omitempty"`
	Params    []string   `json:"params,omitempty"`
	Stale     bool       `json:"stale,omitempty"`
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
}

type quoteRow struct {
	Provider   int    `json:"provider"`
	UpstreamID string `json:"upstreamId"`
}

type quoteRequest struct {
	Links []quoteRow `json:"links"`
}

func (a *supplierLinksAPI) quote(w http.ResponseWriter, r *http.Request) {
	var in quoteRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if len(in.Links) > maxQuoteRows {
		response.BadRequest(w, "too many links")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), supplierCallBudget)
	defer cancel()
	out := make([]LinkQuote, len(in.Links))
	var wg sync.WaitGroup
	for i, l := range in.Links {
		wg.Add(1)
		go func(i int, l quoteRow) {
			defer wg.Done()
			out[i] = a.quoteOne(ctx, l.Provider, strings.TrimSpace(l.UpstreamID))
		}(i, l)
	}
	wg.Wait()
	response.OK(w, map[string]any{"quotes": out})
}

func (a *supplierLinksAPI) quoteOne(ctx context.Context, prov int, up string) LinkQuote {
	q := LinkQuote{Provider: prov, UpstreamID: up, Status: "not_checked"}
	if a.catalog == nil || !a.catalog.Has(prov) {
		return q
	}
	snap, err := a.catalog.Catalog(ctx, prov)
	switch {
	case errors.Is(err, provider.ErrNotImplemented):
		return q
	case err != nil:
		q.Status = "unavailable"
		return q
	}
	at := snap.FetchedAt
	q.Stale, q.CheckedAt = snap.Stale, &at
	cp, ok := snap.Find(up)
	if !ok {
		q.Status = "unknown"
		return q
	}
	q.Status = "ok"
	q.Name, q.Price, q.Currency, q.Available = cp.Name, cp.Price, cp.Currency, cp.Available
	q.QtyMin, q.QtyMax, q.QtyValues, q.Params = cp.QtyMin, cp.QtyMax, cp.QtyValues, cp.Params
	return q
}

// SupplierSearchHit is one supplier product matching a search.
type SupplierSearchHit struct {
	Provider     int      `json:"provider"`
	ProviderName string   `json:"providerName"`
	UpstreamID   string   `json:"upstreamId"`
	Name         string   `json:"name"`
	Category     string   `json:"category,omitempty"`
	Price        float64  `json:"price"`
	Currency     string   `json:"currency"`
	Available    bool     `json:"available"`
	QtyMin       *int     `json:"qtyMin,omitempty"`
	QtyMax       *int     `json:"qtyMax,omitempty"`
	QtyValues    []string `json:"qtyValues,omitempty"`
	Params       []string `json:"params,omitempty"`
}

// SupplierSearchMiss reports a supplier whose catalog couldn't be searched.
type SupplierSearchMiss struct {
	Provider     int    `json:"provider"`
	ProviderName string `json:"providerName"`
	Reason       string `json:"reason"` // "unavailable" | "not_supported"
}

func (a *supplierLinksAPI) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < searchMinQueryLen {
		response.BadRequest(w, "type at least 2 characters")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), supplierCallBudget)
	defer cancel()

	type result struct {
		hits []SupplierSearchHit
		miss *SupplierSearchMiss
	}
	results := make([]result, len(a.providers))
	var wg sync.WaitGroup
	for i, p := range a.providers {
		if a.catalog == nil || !a.catalog.Has(p.ID) {
			continue
		}
		wg.Add(1)
		go func(i int, p FulfillmentProviderOption) {
			defer wg.Done()
			snap, err := a.catalog.Catalog(ctx, p.ID)
			switch {
			case errors.Is(err, provider.ErrNotImplemented):
				results[i].miss = &SupplierSearchMiss{Provider: p.ID, ProviderName: p.Name, Reason: "not_supported"}
				return
			case err != nil:
				results[i].miss = &SupplierSearchMiss{Provider: p.ID, ProviderName: p.Name, Reason: "unavailable"}
				return
			}
			results[i].hits = matchCatalog(p, snap.Products, q, searchPerSupplier)
		}(i, p)
	}
	wg.Wait()

	hits := []SupplierSearchHit{}
	misses := []SupplierSearchMiss{}
	for _, res := range results {
		hits = append(hits, res.hits...)
		if res.miss != nil {
			misses = append(misses, *res.miss)
		}
	}
	response.OK(w, map[string]any{"results": hits, "unavailable": misses})
}

// matchCatalog returns a supplier's products matching q: an exact id match, or
// every word of q found in the name or category (case-insensitive). The exact
// id, then exact name, then name-prefix matches sort first, available before
// unavailable, then by name.
func matchCatalog(p FulfillmentProviderOption, products []provider.CatalogProduct, q string, limit int) []SupplierSearchHit {
	ql := strings.ToLower(q)
	words := strings.Fields(ql)
	type scored struct {
		hit   SupplierSearchHit
		score int
	}
	var found []scored
	for _, cp := range products {
		name := strings.ToLower(cp.Name)
		hay := name + " " + strings.ToLower(cp.Category)
		score := 0
		switch {
		case cp.UpstreamID == q:
			score = 4
		case name == ql:
			score = 3
		case strings.HasPrefix(name, ql):
			score = 2
		default:
			for _, w := range words {
				if !strings.Contains(hay, w) {
					score = -1
					break
				}
			}
			if score < 0 {
				continue
			}
			score = 1
		}
		found = append(found, scored{score: score, hit: SupplierSearchHit{
			Provider: p.ID, ProviderName: p.Name, UpstreamID: cp.UpstreamID, Name: cp.Name, Category: cp.Category,
			Price: cp.Price, Currency: cp.Currency, Available: cp.Available,
			QtyMin: cp.QtyMin, QtyMax: cp.QtyMax, QtyValues: cp.QtyValues, Params: cp.Params,
		}})
	}
	sort.SliceStable(found, func(i, j int) bool {
		a, b := found[i], found[j]
		if a.score != b.score {
			return a.score > b.score
		}
		if a.hit.Available != b.hit.Available {
			return a.hit.Available
		}
		return a.hit.Name < b.hit.Name
	})
	if len(found) > limit {
		found = found[:limit]
	}
	out := make([]SupplierSearchHit, len(found))
	for i, f := range found {
		out[i] = f.hit
	}
	return out
}
