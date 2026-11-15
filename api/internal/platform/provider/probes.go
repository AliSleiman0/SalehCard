package provider

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Default Probes timings. Catalog and balance calls hit billed / rate-limited
// supplier APIs, so every caller (the /suppliers page, the catalog picker, the
// reroute guards, later the sync job) shares one cache instead of probing on
// every request.
const (
	DefaultCatalogTTL = 5 * time.Minute
	// DefaultErrorTTL is how long a FAILED catalog fetch is remembered, so a
	// supplier outage doesn't turn every admin click into another slow call.
	DefaultErrorTTL = 30 * time.Second
	// DefaultStaleMax is how old a good catalog may be and still be served
	// (flagged Stale) when a refresh fails.
	DefaultStaleMax   = 30 * time.Minute
	DefaultBalanceTTL = 60 * time.Second
	// probeCallTimeout caps one upstream catalog/balance call.
	probeCallTimeout = 8 * time.Second
)

// CatalogSnapshot is a supplier's catalog as served by Probes.
type CatalogSnapshot struct {
	Products  []CatalogProduct
	FetchedAt time.Time
	// Stale is true when the latest refresh failed and this is an older copy
	// (at most StaleMax old).
	Stale bool
}

// Find returns the catalog entry for upstreamID.
func (c CatalogSnapshot) Find(upstreamID string) (CatalogProduct, bool) {
	for _, p := range c.Products {
		if p.UpstreamID == upstreamID {
			return p, true
		}
	}
	return CatalogProduct{}, false
}

// ProbesConfig tunes a Probes cache; zero fields take the defaults.
type ProbesConfig struct {
	CatalogTTL time.Duration
	ErrorTTL   time.Duration
	StaleMax   time.Duration
	BalanceTTL time.Duration
	// Now is the clock (tests); nil = time.Now.
	Now func() time.Time
}

// Probes is the shared, read-through cache over each supplier's Cataloger
// (catalog + balance). Concurrent misses for the same supplier collapse into one
// upstream call (singleflight), and that call runs on a context detached from
// the caller's, so an admin closing the tab can't poison the cache with a
// cancellation error. A supplier whose adapter is not a Cataloger reports
// ErrNotImplemented — callers treat that as "can't check", never as a refusal.
type Probes struct {
	reg *Registry
	cfg ProbesConfig

	sf   singleflight.Group
	mu   sync.Mutex
	cats map[int]*catalogEntry
	bals map[int]balanceEntry
}

type catalogEntry struct {
	good    *CatalogSnapshot // last successful fetch
	err     error            // last fetch error (nil when the last fetch succeeded)
	errAt   time.Time
	fetched time.Time // time of the last successful fetch
}

type balanceEntry struct {
	acc Account
	err error
	at  time.Time
}

// NewProbes builds the shared cache over a provider registry.
func NewProbes(reg *Registry, cfg ProbesConfig) *Probes {
	if cfg.CatalogTTL <= 0 {
		cfg.CatalogTTL = DefaultCatalogTTL
	}
	if cfg.ErrorTTL <= 0 {
		cfg.ErrorTTL = DefaultErrorTTL
	}
	if cfg.StaleMax <= 0 {
		cfg.StaleMax = DefaultStaleMax
	}
	if cfg.StaleMax < cfg.CatalogTTL {
		cfg.StaleMax = cfg.CatalogTTL
	}
	if cfg.BalanceTTL <= 0 {
		cfg.BalanceTTL = DefaultBalanceTTL
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Probes{reg: reg, cfg: cfg, cats: map[int]*catalogEntry{}, bals: map[int]balanceEntry{}}
}

// Registry is the provider registry the cache reads through.
func (p *Probes) Registry() *Registry { return p.reg }

// Has reports whether a real adapter is registered for supplier id.
func (p *Probes) Has(id int) bool { return p.reg.Has(id) }

func (p *Probes) cataloger(id int) (Cataloger, bool) {
	if !p.reg.Has(id) {
		return nil, false
	}
	cat, ok := p.reg.Resolve(&id).(Cataloger)
	return cat, ok
}

// Catalog returns a supplier's catalog, from cache when fresh. On a failed
// refresh it serves the last good copy (Stale=true) while that is younger than
// StaleMax, else the error.
func (p *Probes) Catalog(ctx context.Context, id int) (CatalogSnapshot, error) {
	return p.catalog(ctx, id, false)
}

// Refresh fetches a supplier's catalog now (bypassing a fresh cache entry) —
// for the admin "Sync catalog" button and the sync job. The stale fallback
// still applies when the fetch fails.
func (p *Probes) Refresh(ctx context.Context, id int) (CatalogSnapshot, error) {
	return p.catalog(ctx, id, true)
}

// Lookup returns one upstream product from the (cached) catalog. found=false
// with a nil error means the catalog loaded and the id is not in it.
func (p *Probes) Lookup(ctx context.Context, id int, upstreamID string) (CatalogProduct, bool, CatalogSnapshot, error) {
	snap, err := p.Catalog(ctx, id)
	if err != nil {
		return CatalogProduct{}, false, snap, err
	}
	cp, ok := snap.Find(upstreamID)
	return cp, ok, snap, nil
}

func (p *Probes) catalog(ctx context.Context, id int, force bool) (CatalogSnapshot, error) {
	cat, ok := p.cataloger(id)
	if !ok {
		return CatalogSnapshot{}, ErrNotImplemented
	}
	now := p.cfg.Now()
	if !force {
		p.mu.Lock()
		e := p.cats[id]
		if e != nil {
			if e.good != nil && e.err == nil && now.Sub(e.fetched) < p.cfg.CatalogTTL {
				snap := *e.good
				p.mu.Unlock()
				return snap, nil
			}
			if e.err != nil && now.Sub(e.errAt) < p.cfg.ErrorTTL {
				snap, err := p.fallbackLocked(e, now)
				p.mu.Unlock()
				return snap, err
			}
		}
		p.mu.Unlock()
	}

	ch := p.sf.DoChan("cat:"+strconv.Itoa(id), func() (any, error) {
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), probeCallTimeout)
		defer cancel()
		products, err := cat.ListProducts(cctx)
		at := p.cfg.Now()
		p.mu.Lock()
		defer p.mu.Unlock()
		e := p.cats[id]
		if e == nil {
			e = &catalogEntry{}
			p.cats[id] = e
		}
		if err != nil {
			e.err, e.errAt = err, at
			return nil, err
		}
		e.good = &CatalogSnapshot{Products: products, FetchedAt: at}
		e.err, e.fetched = nil, at
		return nil, nil
	})
	select {
	case <-ctx.Done():
		return CatalogSnapshot{}, ctx.Err()
	case <-ch:
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	e := p.cats[id]
	if e == nil {
		return CatalogSnapshot{}, fmt.Errorf("provider %d: catalog unavailable: %w", id, ErrUnavailable)
	}
	if e.err == nil && e.good != nil {
		return *e.good, nil
	}
	return p.fallbackLocked(e, p.cfg.Now())
}

// fallbackLocked serves the last good copy (Stale) when young enough, else the
// remembered error. p.mu must be held.
func (p *Probes) fallbackLocked(e *catalogEntry, now time.Time) (CatalogSnapshot, error) {
	if e.good != nil && now.Sub(e.fetched) < p.cfg.StaleMax {
		snap := *e.good
		snap.Stale = true
		return snap, nil
	}
	return CatalogSnapshot{}, e.err
}

// Balance returns a supplier's account snapshot, cached for BalanceTTL
// (errors too — a failing supplier is not re-probed on every page load).
func (p *Probes) Balance(ctx context.Context, id int) (Account, error) {
	cat, ok := p.cataloger(id)
	if !ok {
		return Account{}, ErrNotImplemented
	}
	p.mu.Lock()
	if b, ok := p.bals[id]; ok && p.cfg.Now().Sub(b.at) < p.cfg.BalanceTTL {
		p.mu.Unlock()
		return b.acc, b.err
	}
	p.mu.Unlock()

	ch := p.sf.DoChan("bal:"+strconv.Itoa(id), func() (any, error) {
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), probeCallTimeout)
		defer cancel()
		acc, err := cat.Profile(cctx)
		p.mu.Lock()
		p.bals[id] = balanceEntry{acc: acc, err: err, at: p.cfg.Now()}
		p.mu.Unlock()
		return nil, nil
	})
	select {
	case <-ctx.Done():
		return Account{}, ctx.Err()
	case <-ch:
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	b := p.bals[id]
	return b.acc, b.err
}

// AcceptsQty reports whether an upstream product accepts qty (true when it
// states no limits) — the server twin of the admin's acceptsQty.
func AcceptsQty(p CatalogProduct, qty int) bool {
	if len(p.QtyValues) > 0 {
		for _, v := range p.QtyValues {
			if n, err := strconv.Atoi(v); err == nil && n == qty {
				return true
			}
			if f, err := strconv.ParseFloat(v, 64); err == nil && f == float64(qty) {
				return true
			}
		}
		return false
	}
	if p.QtyMin != nil && qty < *p.QtyMin {
		return false
	}
	if p.QtyMax != nil && qty > *p.QtyMax {
		return false
	}
	return true
}
