package provider

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeCataloger is a Provider + Cataloger with scripted catalog/balance results.
type fakeCataloger struct {
	StubProvider
	listCalls atomic.Int32
	profCalls atomic.Int32
	gate      chan struct{} // when non-nil, ListProducts blocks until closed

	mu       sync.Mutex
	products []CatalogProduct
	listErr  error
	balance  float64
	profErr  error
}

func (f *fakeCataloger) ListProducts(_ context.Context) ([]CatalogProduct, error) {
	f.listCalls.Add(1)
	if f.gate != nil {
		<-f.gate
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.products, f.listErr
}

func (f *fakeCataloger) Profile(_ context.Context) (Account, error) {
	f.profCalls.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	return Account{Balance: f.balance, Currency: "USD"}, f.profErr
}

func (f *fakeCataloger) set(products []CatalogProduct, err error) {
	f.mu.Lock()
	f.products, f.listErr = products, err
	f.mu.Unlock()
}

type testClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func newTestProbes(f *fakeCataloger) (*Probes, *testClock) {
	f.id = 10
	clk := &testClock{t: time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)}
	p := NewProbes(NewRegistry(f), ProbesConfig{
		CatalogTTL: 5 * time.Minute, ErrorTTL: 30 * time.Second,
		StaleMax: 30 * time.Minute, BalanceTTL: time.Minute, Now: clk.Now,
	})
	return p, clk
}

var boom = errors.New("boom")

func TestProbesCatalog_CachesWithinTTL(t *testing.T) {
	f := &fakeCataloger{}
	f.set([]CatalogProduct{{UpstreamID: "3258", Price: 0.0001}}, nil)
	p, clk := newTestProbes(f)
	ctx := context.Background()

	for i := range 3 {
		snap, err := p.Catalog(ctx, 10)
		if err != nil || len(snap.Products) != 1 || snap.Stale {
			t.Fatalf("call %d: snap=%+v err=%v", i, snap, err)
		}
	}
	if n := f.listCalls.Load(); n != 1 {
		t.Fatalf("ListProducts calls = %d, want 1 (cached)", n)
	}
	clk.Advance(5*time.Minute + time.Second)
	if _, err := p.Catalog(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if n := f.listCalls.Load(); n != 2 {
		t.Fatalf("after TTL ListProducts calls = %d, want 2", n)
	}
}

func TestProbesCatalog_ServesStaleOnError(t *testing.T) {
	f := &fakeCataloger{}
	f.set([]CatalogProduct{{UpstreamID: "3258"}}, nil)
	p, clk := newTestProbes(f)
	ctx := context.Background()
	if _, err := p.Catalog(ctx, 10); err != nil {
		t.Fatal(err)
	}

	f.set(nil, boom)
	clk.Advance(6 * time.Minute)
	snap, err := p.Catalog(ctx, 10)
	if err != nil || !snap.Stale || len(snap.Products) != 1 {
		t.Fatalf("want stale copy, got snap=%+v err=%v", snap, err)
	}
	// The error is remembered for ErrorTTL: no second call yet.
	_, _ = p.Catalog(ctx, 10)
	if n := f.listCalls.Load(); n != 2 {
		t.Fatalf("ListProducts calls = %d, want 2 (error cached)", n)
	}

	// Past StaleMax the error surfaces.
	clk.Advance(30 * time.Minute)
	if _, err := p.Catalog(ctx, 10); !errors.Is(err, boom) {
		t.Fatalf("past StaleMax err = %v, want boom", err)
	}
}

func TestProbesCatalog_ErrorWithoutGoodCopy(t *testing.T) {
	f := &fakeCataloger{}
	f.set(nil, boom)
	p, clk := newTestProbes(f)
	ctx := context.Background()
	if _, err := p.Catalog(ctx, 10); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	if _, err := p.Catalog(ctx, 10); !errors.Is(err, boom) {
		t.Fatalf("cached err = %v, want boom", err)
	}
	if n := f.listCalls.Load(); n != 1 {
		t.Fatalf("calls = %d, want 1", n)
	}
	f.set([]CatalogProduct{{UpstreamID: "1"}}, nil)
	clk.Advance(31 * time.Second)
	if snap, err := p.Catalog(ctx, 10); err != nil || len(snap.Products) != 1 {
		t.Fatalf("after ErrorTTL snap=%+v err=%v", snap, err)
	}
}

func TestProbesRefresh_BypassesCache(t *testing.T) {
	f := &fakeCataloger{}
	f.set([]CatalogProduct{{UpstreamID: "1"}}, nil)
	p, _ := newTestProbes(f)
	ctx := context.Background()
	_, _ = p.Catalog(ctx, 10)
	f.set([]CatalogProduct{{UpstreamID: "1"}, {UpstreamID: "2"}}, nil)
	snap, err := p.Refresh(ctx, 10)
	if err != nil || len(snap.Products) != 2 {
		t.Fatalf("refresh snap=%+v err=%v", snap, err)
	}
	if snap2, _ := p.Catalog(ctx, 10); len(snap2.Products) != 2 {
		t.Fatalf("cache not updated by refresh: %+v", snap2)
	}
}

func TestProbesCatalog_Singleflight(t *testing.T) {
	f := &fakeCataloger{gate: make(chan struct{})}
	f.set([]CatalogProduct{{UpstreamID: "1"}}, nil)
	p, _ := newTestProbes(f)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := p.Catalog(context.Background(), 10); err != nil {
				t.Error(err)
			}
		}()
	}
	// Let the goroutines pile up on the in-flight call, then release it.
	time.Sleep(50 * time.Millisecond)
	close(f.gate)
	wg.Wait()
	if n := f.listCalls.Load(); n != 1 {
		t.Fatalf("ListProducts calls = %d, want 1 (singleflight)", n)
	}
}

func TestProbesCatalog_CallerCancelDoesNotPoisonCache(t *testing.T) {
	f := &fakeCataloger{gate: make(chan struct{})}
	f.set([]CatalogProduct{{UpstreamID: "1"}}, nil)
	p, _ := newTestProbes(f)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := p.Catalog(ctx, 10)
		done <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("caller err = %v, want canceled", err)
	}
	close(f.gate)
	// The detached fetch still completes and fills the cache.
	deadline := time.Now().Add(2 * time.Second)
	for {
		snap, err := p.Catalog(context.Background(), 10)
		if err == nil && len(snap.Products) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("cache never filled: %+v %v", snap, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if n := f.listCalls.Load(); n != 1 {
		t.Fatalf("ListProducts calls = %d, want 1", n)
	}
}

func TestProbes_NotCataloger(t *testing.T) {
	p := NewProbes(NewRegistry(NewStub(10)), ProbesConfig{})
	if _, err := p.Catalog(context.Background(), 10); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("stub catalog err = %v", err)
	}
	if _, err := p.Catalog(context.Background(), 99); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("unregistered catalog err = %v", err)
	}
	if _, err := p.Balance(context.Background(), 10); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("stub balance err = %v", err)
	}
}

func TestProbesBalance_Cached(t *testing.T) {
	f := &fakeCataloger{balance: 12.5}
	p, clk := newTestProbes(f)
	ctx := context.Background()
	for range 3 {
		acc, err := p.Balance(ctx, 10)
		if err != nil || acc.Balance != 12.5 {
			t.Fatalf("acc=%+v err=%v", acc, err)
		}
	}
	if n := f.profCalls.Load(); n != 1 {
		t.Fatalf("Profile calls = %d, want 1", n)
	}
	clk.Advance(61 * time.Second)
	_, _ = p.Balance(ctx, 10)
	if n := f.profCalls.Load(); n != 2 {
		t.Fatalf("Profile calls = %d, want 2", n)
	}
}

func TestLookup(t *testing.T) {
	f := &fakeCataloger{}
	f.set([]CatalogProduct{{UpstreamID: "3258", Name: "SOUL STAR"}}, nil)
	p, _ := newTestProbes(f)
	cp, ok, _, err := p.Lookup(context.Background(), 10, "3258")
	if err != nil || !ok || cp.Name != "SOUL STAR" {
		t.Fatalf("lookup 3258 = %+v %v %v", cp, ok, err)
	}
	if _, ok, _, err := p.Lookup(context.Background(), 10, "1045"); err != nil || ok {
		t.Fatalf("lookup 1045 ok=%v err=%v, want not found", ok, err)
	}
}

func TestAcceptsQty(t *testing.T) {
	ip := func(n int) *int { return &n }
	cases := []struct {
		name string
		p    CatalogProduct
		qty  int
		want bool
	}{
		{"no limits", CatalogProduct{}, 5, true},
		{"range ok", CatalogProduct{QtyMin: ip(10000), QtyMax: ip(50000000)}, 10000, true},
		{"below min", CatalogProduct{QtyMin: ip(10000)}, 1, false},
		{"above max", CatalogProduct{QtyMax: ip(100)}, 101, false},
		{"values hit", CatalogProduct{QtyValues: []string{"110", "150"}}, 150, true},
		{"values miss", CatalogProduct{QtyValues: []string{"110", "150"}}, 120, false},
		{"values decimal", CatalogProduct{QtyValues: []string{"110.0"}}, 110, true},
	}
	for _, c := range cases {
		if got := AcceptsQty(c.p, c.qty); got != c.want {
			t.Errorf("%s: AcceptsQty = %v, want %v", c.name, got, c.want)
		}
	}
}
