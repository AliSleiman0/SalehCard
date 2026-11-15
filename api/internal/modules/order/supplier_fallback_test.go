package order

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

// refusedFake is a scripted supplier that definitely refuses every order.
func refusedFake(id int) *fakeProvider {
	return &fakeProvider{id: id, err: &refusedTestErr{id: id}}
}

// refusedTestErr matches ErrRefused AND ErrUnavailable, like the adapters'.
type refusedTestErr struct{ id int }

func (e *refusedTestErr) Error() string { return fmt.Sprintf("supplier %d refused (code 100)", e.id) }
func (e *refusedTestErr) Is(t error) bool {
	return t == provider.ErrRefused || t == provider.ErrUnavailable
}

func fallbackActorEntries(rec *fakeRecorder) []audit.Entry {
	var out []audit.Entry
	for _, e := range rec.entries {
		if e.Action == audit.ActionOrderSupplierFallback {
			out = append(out, e)
		}
	}
	return out
}

// --- nextLink decision table --------------------------------------------------

func TestNextLink(t *testing.T) {
	has := func(id int) bool { return id != 99 } // 99 = not configured
	base := func(mut ...func(*Order)) *Order {
		pid := 7
		o := &Order{Items: []OrderItem{{FulfillmentMode: "api", FulfillmentProvider: &pid, UpstreamProductID: "364", Qty: 1}}}
		for _, m := range mut {
			m(o)
		}
		return o
	}
	prod := func(mut func(*product.Product), links ...product.SupplierLink) *product.Product {
		p := &product.Product{SupplierLinks: links}
		if mut != nil {
			mut(p)
		}
		return p
	}
	cur := link("cur", 7, "364", true)

	cases := []struct {
		name   string
		o      *Order
		p      *product.Product
		usable func(product.SupplierLink) bool
		want   string // link id, "" = none
	}{
		{"first eligible in admin order", base(), prod(nil, cur, link("b", 9, "A", true), link("c", 10, "B", true)), nil, "b"},
		{"skips disabled", base(), prod(nil, cur, link("b", 9, "A", false), link("c", 10, "B", true)), nil, "c"},
		{"skips empty upstream", base(), prod(nil, cur, link("b", 9, " ", true), link("c", 10, "B", true)), nil, "c"},
		{"skips unconfigured supplier", base(), prod(nil, cur, link("b", 99, "A", true), link("c", 10, "B", true)), nil, "c"},
		{"skips a link tried earlier", base(func(o *Order) {
			p := 9
			o.Fulfillment.SupplierAttempts = []SupplierAttempt{{Provider: &p, UpstreamProductID: "A"}}
		}), prod(nil, cur, link("b", 9, "A", true), link("c", 10, "B", true)), nil, "c"},
		{"same supplier, other product is not tried", base(), prod(nil, cur, link("b", 7, "365", true)), nil, "b"},
		{"skips a link failing the catalog check", base(), prod(nil, cur, link("b", 9, "A", true), link("c", 10, "B", true)),
			func(l product.SupplierLink) bool { return l.ID != "b" }, "c"},
		{"none left", base(), prod(nil, cur), nil, ""},
		{"fallback off for the product", base(), prod(func(p *product.Product) { p.SupplierFallbackOff = true }, cur, link("b", 9, "A", true)), nil, ""},
		{"admin-routed attempt", base(func(o *Order) { o.Fulfillment.SupplierAttempt = 1 }), prod(nil, cur, link("b", 9, "A", true)), nil, ""},
		{"automatic attempt may hop again", base(func(o *Order) {
			o.Fulfillment.SupplierAttempt = 1
			o.Fulfillment.SupplierAttemptAuto = true
		}), prod(nil, cur, link("b", 9, "A", true)), nil, "b"},
		{"ambiguous attempt", base(func(o *Order) { o.Fulfillment.SupplierAmbiguous = true }), prod(nil, cur, link("b", 9, "A", true)), nil, ""},
		{"live ref", base(func(o *Order) { o.Fulfillment.ProviderRef = "UP_1" }), prod(nil, cur, link("b", 9, "A", true)), nil, ""},
		{"search ref", base(func(o *Order) { o.Fulfillment.ProviderRef = "search:bundle:x" }), prod(nil, cur, link("b", 9, "A", true)), nil, ""},
		{"no product", base(), nil, nil, ""},
		{"no supplier line", &Order{Items: []OrderItem{{FulfillmentMode: "manual_operator"}}}, prod(nil, link("b", 9, "A", true)), nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l, ok := nextLink(tc.o, tc.p, has, tc.usable)
			if tc.want == "" {
				assert.False(t, ok, "got %q", l.ID)
				return
			}
			require.True(t, ok)
			assert.Equal(t, tc.want, l.ID)
		})
	}
}

// --- placement ----------------------------------------------------------------

// fallbackProduct is supplierProduct(…, 7, "364") with links to the given
// backups (provider, upstream) after the active #7 link.
func fallbackProduct(backups ...product.SupplierLink) *product.Product {
	p := supplierProduct(2.5, 7, "364")
	p.SupplierLinks = append([]product.SupplierLink{link("cur", 7, "364", true)}, backups...)
	p.ActiveSupplierLinkID = "cur"
	return p
}

func placeOne(t *testing.T, svc *OrderService, p *product.Product) (*Order, error) {
	t.Helper()
	return svc.PlaceOrder(context.Background(), bson.NewObjectID(), false, "k-"+bson.NewObjectID().Hex(), PlaceOrderInput{
		Items:         []PlaceOrderItemInput{supplierItem(p, 1)},
		PaymentMethod: PaymentMethodWallet,
	})
}

func TestPlacementFallback(t *testing.T) {
	t.Run("refusal completes at the next link", func(t *testing.T) {
		p := fallbackProduct(link("b", 9, "U-9", true))
		first := refusedFake(7)
		backup := &fakeProvider{id: 9, res: provider.Result{Reference: "B_1"}}
		walletSvc := &fakeWalletSvc{balance: 100}
		svc, _ := newSUTWithProviders([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, walletSvc, first, backup)
		rec := &fakeRecorder{}
		svc.SetSupplierFallback(nil, rec)

		o, err := placeOne(t, svc, p)
		require.NoError(t, err)
		assert.Equal(t, OrderStatusCompleted, o.Status)
		assert.Equal(t, "B_1", o.Fulfillment.ProviderRef)
		assert.Equal(t, 9, *o.Items[0].FulfillmentProvider)
		assert.Equal(t, "U-9", o.Items[0].UpstreamProductID)
		assert.Equal(t, 1, o.Fulfillment.SupplierAttempt)
		assert.True(t, o.Fulfillment.SupplierAttemptAuto)
		assert.Nil(t, o.Fulfillment.SupplierNextRetryAt, "no stale retry clock on a completed order")
		assert.Nil(t, o.Fulfillment.SupplierAttemptAt, "the give-up clock keeps counting from placement")
		// Fresh idempotency key for the new attempt.
		require.NotNil(t, backup.lastIn)
		assert.NotEqual(t, o.ID.Hex(), backup.lastIn.OrderUUID)
		assert.Equal(t, o.ID.Hex(), first.lastIn.OrderUUID)
		// Archived attempt + note + audit.
		require.Len(t, o.Fulfillment.SupplierAttempts, 1)
		a := o.Fulfillment.SupplierAttempts[0]
		assert.True(t, a.Auto)
		assert.Equal(t, 7, *a.Provider)
		assert.Equal(t, fallbackActor, a.ReroutedBy)
		assert.Contains(t, a.Reason, "refused")
		assert.True(t, hasTimelineNote(o, "supplier #7 couldn't take the order — trying supplier #9 automatically"))
		entries := fallbackActorEntries(rec)
		require.Len(t, entries, 1)
		assert.Equal(t, fallbackActor, entries[0].ActorID)
		assert.Equal(t, "placement", entries[0].Summary["at"])
		assert.Equal(t, 97.5, walletSvc.balance)
		assert.Equal(t, 0, walletSvc.refundCalls)
	})

	t.Run("refusal then a hard error at the backup compensates", func(t *testing.T) {
		p := fallbackProduct(link("b", 9, "U-9", true))
		backup := &fakeProvider{id: 9, err: errors.New("panel s9: player id blocked (code 107)")}
		walletSvc := &fakeWalletSvc{balance: 100}
		svc, repo := newSUTWithProviders([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, walletSvc, refusedFake(7), backup)

		_, err := placeOne(t, svc, p)
		require.Error(t, err)
		assert.Equal(t, 100.0, walletSvc.balance)
		assert.Equal(t, 1, walletSvc.refundCalls)
		for _, o := range repo.byID {
			assert.Equal(t, OrderStatusFailed, o.Status)
		}
	})

	t.Run("a transport failure never falls back", func(t *testing.T) {
		p := fallbackProduct(link("b", 9, "U-9", true))
		first := &fakeProvider{id: 7, err: fmt.Errorf("panel s7: request failed: %w", provider.ErrUnavailable)}
		backup := &fakeProvider{id: 9, res: provider.Result{Reference: "B_1"}}
		svc, _ := newSUTWithProviders([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{balance: 100}, first, backup)

		o, err := placeOne(t, svc, p)
		require.NoError(t, err)
		assert.Equal(t, OrderStatusProcessing, o.Status)
		assert.Equal(t, 0, backup.calls)
		assert.True(t, o.Fulfillment.SupplierAmbiguous)
		assert.NotNil(t, o.Fulfillment.SupplierNextRetryAt)
	})

	t.Run("toggle off parks without falling back", func(t *testing.T) {
		p := fallbackProduct(link("b", 9, "U-9", true))
		p.SupplierFallbackOff = true
		backup := &fakeProvider{id: 9, res: provider.Result{Reference: "B_1"}}
		svc, _ := newSUTWithProviders([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{balance: 100}, refusedFake(7), backup)

		o, err := placeOne(t, svc, p)
		require.NoError(t, err)
		assert.Equal(t, OrderStatusProcessing, o.Status)
		assert.Equal(t, 0, backup.calls)
		assert.False(t, o.Fulfillment.SupplierAmbiguous, "a refusal is not ambiguous")
		assert.Equal(t, 0, o.Fulfillment.SupplierAttempt)
		assert.NotNil(t, o.Fulfillment.SupplierNextRetryAt)
	})

	t.Run("at most two hops at checkout", func(t *testing.T) {
		p := fallbackProduct(link("b", 8, "U-8", true), link("c", 9, "U-9", true), link("d", 10, "U-10", true))
		s7, s8, s9 := refusedFake(7), refusedFake(8), refusedFake(9)
		s10 := &fakeProvider{id: 10, res: provider.Result{Reference: "D_1"}}
		svc, _ := newSUTWithProviders([]*product.Product{p}, &fakeCodeSvc{available: map[string]int{}}, &fakeWalletSvc{balance: 100}, s7, s8, s9, s10)

		o, err := placeOne(t, svc, p)
		require.NoError(t, err)
		assert.Equal(t, OrderStatusProcessing, o.Status)
		assert.Equal(t, []int{1, 1, 1, 0}, []int{s7.calls, s8.calls, s9.calls, s10.calls})
		assert.Equal(t, 9, *o.Items[0].FulfillmentProvider)
		assert.Equal(t, 2, o.Fulfillment.SupplierAttempt)
		assert.Len(t, o.Fulfillment.SupplierAttempts, 2)
		assert.NotNil(t, o.Fulfillment.SupplierNextRetryAt, "parked for the settler to continue")
	})

	t.Run("customer responses hide the attempts archive", func(t *testing.T) {
		o := &Order{Fulfillment: Fulfillment{SupplierAttempts: []SupplierAttempt{{Reason: "balance"}}}}
		assert.Nil(t, customerOrder(o).Fulfillment.SupplierAttempts)
		assert.Len(t, o.Fulfillment.SupplierAttempts, 1, "the shared order is not mutated")
	})
}

// --- settler ------------------------------------------------------------------

func newFallbackSettler(t *testing.T, providers ...*fakeProvider) (*SupplierSettler, *fakeOrderRepo, *fakeWalletSvc, *fakeRecorder) {
	t.Helper()
	st, repo, walletSvc, rec := newRerouteSUT(providers[0], providers[1])
	if len(providers) > 2 {
		adapters := make([]provider.Provider, len(providers))
		for i, p := range providers {
			adapters[i] = p
		}
		st.svc.providers = provider.NewRegistry(adapters...)
	}
	st.svc.SetSupplierFallback(nil, rec)
	return st, repo, walletSvc, rec
}

// retryDue arms a due retry clock (a ref-less environmental park).
func retryDue(o *Order) {
	due := time.Now().UTC().Add(-time.Minute)
	o.Fulfillment.SupplierNextRetryAt = &due
}

func TestSettlerFallback(t *testing.T) {
	ctx := context.Background()

	t.Run("a refused re-dispatch hops to the next link and completes", func(t *testing.T) {
		backup := &fakeProvider{id: 9, res: provider.Result{Reference: "B_1"}}
		st, repo, walletSvc, rec := newFallbackSettler(t, refusedFake(7), backup)
		pid := linkedProduct(st, link("cur", 7, "364", true), link("b", 9, "U-9", true))
		o := settlerOrder(repo, onProduct(pid, retryDue))

		st.Tick(ctx)
		assert.Equal(t, OrderStatusCompleted, o.Status)
		assert.Equal(t, 1, backup.calls)
		assert.True(t, o.Fulfillment.SupplierAttemptAuto)
		require.Len(t, fallbackActorEntries(rec), 1)
		assert.Equal(t, "settler", fallbackActorEntries(rec)[0].Summary["at"])
		assert.Equal(t, 0, walletSvc.refundCalls)
	})

	blocked := []struct {
		name string
		mut  func(*Order)
	}{
		{"ambiguous attempt", func(o *Order) { o.Fulfillment.SupplierAmbiguous = true }},
		{"admin-routed attempt", func(o *Order) { o.Fulfillment.SupplierAttempt = 1; o.Fulfillment.SupplierOrderUUID = "x" }},
	}
	for _, b := range blocked {
		t.Run(b.name+" never falls back", func(t *testing.T) {
			backup := &fakeProvider{id: 9, res: provider.Result{Reference: "B_1"}}
			st, repo, _, rec := newFallbackSettler(t, refusedFake(7), backup)
			pid := linkedProduct(st, link("cur", 7, "364", true), link("b", 9, "U-9", true))
			o := settlerOrder(repo, onProduct(pid, retryDue, b.mut))

			st.Tick(ctx)
			assert.Equal(t, OrderStatusProcessing, o.Status)
			assert.Equal(t, 0, backup.calls)
			assert.Empty(t, fallbackActorEntries(rec))
			assert.Equal(t, 1, o.Fulfillment.SupplierRetryCount, "just a backoff bump")
		})
	}

	t.Run("a search ref is polled, never re-dispatched or fallen back from", func(t *testing.T) {
		first := refusedFake(7)
		first.checkRes = provider.Status{State: "wait"}
		backup := &fakeProvider{id: 9, res: provider.Result{Reference: "B_1"}}
		st, repo, _, _ := newFallbackSettler(t, first, backup)
		pid := linkedProduct(st, link("cur", 7, "364", true), link("b", 9, "U-9", true))
		o := settlerOrder(repo, onProduct(pid, func(o *Order) { o.Fulfillment.ProviderRef = "search:bundle:" + o.ID.Hex() }))

		st.Tick(ctx)
		assert.Equal(t, 0, first.calls)
		assert.Equal(t, 0, backup.calls)
		assert.Equal(t, OrderStatusProcessing, o.Status)
	})

	t.Run("a transport failure marks the attempt ambiguous", func(t *testing.T) {
		first := &fakeProvider{id: 7, err: fmt.Errorf("request failed: %w", provider.ErrUnavailable)}
		backup := &fakeProvider{id: 9, res: provider.Result{Reference: "B_1"}}
		st, repo, _, _ := newFallbackSettler(t, first, backup)
		pid := linkedProduct(st, link("cur", 7, "364", true), link("b", 9, "U-9", true))
		o := settlerOrder(repo, onProduct(pid, retryDue))

		st.Tick(ctx)
		assert.True(t, o.Fulfillment.SupplierAmbiguous)
		assert.Equal(t, 0, backup.calls)
	})

	t.Run("an automatic attempt rejected by the backup refunds exactly once", func(t *testing.T) {
		backup := &fakeProvider{id: 9, err: errors.New("player id blocked (code 107)")}
		st, repo, walletSvc, _ := newFallbackSettler(t, refusedFake(7), backup)
		pid := linkedProduct(st, link("cur", 7, "364", true), link("b", 9, "U-9", true))
		o := settlerOrder(repo, onProduct(pid, retryDue))

		st.Tick(ctx)
		assert.Equal(t, OrderStatusRefunded, o.Status)
		assert.Equal(t, 1, walletSvc.refundCalls)
		st.Tick(ctx)
		assert.Equal(t, 1, walletSvc.refundCalls)
	})

	t.Run("an automatic attempt rejected on a status check refunds", func(t *testing.T) {
		backup := &fakeProvider{id: 9, checkRes: provider.Status{State: "reject"}}
		st, repo, walletSvc, _ := newFallbackSettler(t, refusedFake(7), backup)
		o := settlerOrder(repo, func(o *Order) {
			p := 9
			o.Items[0].FulfillmentProvider = &p
			o.Fulfillment.SupplierAttempt = 1
			o.Fulfillment.SupplierAttemptAuto = true
			o.Fulfillment.ProviderRef = "B_1"
		})

		st.Tick(ctx)
		assert.Equal(t, OrderStatusRefunded, o.Status)
		assert.Equal(t, 1, walletSvc.refundCalls)
	})

	t.Run("an admin-routed attempt rejected on a status check waits for the admin", func(t *testing.T) {
		backup := &fakeProvider{id: 9, checkRes: provider.Status{State: "reject"}}
		st, repo, walletSvc, _ := newFallbackSettler(t, refusedFake(7), backup)
		o := settlerOrder(repo, func(o *Order) {
			p := 9
			o.Items[0].FulfillmentProvider = &p
			o.Fulfillment.SupplierAttempt = 1
			o.Fulfillment.ProviderRef = "B_1"
		})

		st.Tick(ctx)
		assert.Equal(t, OrderStatusProcessing, o.Status)
		assert.NotNil(t, o.Fulfillment.SupplierStuckAt)
		assert.Equal(t, 0, walletSvc.refundCalls)
	})

	t.Run("all links tried: keeps retrying the last one", func(t *testing.T) {
		st, repo, _, rec := newFallbackSettler(t, refusedFake(7), refusedFake(9))
		pid := linkedProduct(st, link("cur", 7, "364", true), link("b", 9, "U-9", true))
		o := settlerOrder(repo, onProduct(pid, retryDue))

		st.Tick(ctx) // 7 refuses → hop to 9 → 9 refuses → bump
		assert.Equal(t, 9, *o.Items[0].FulfillmentProvider)
		retryDue(o)
		st.Tick(ctx) // 9 refuses again, nothing left → bump
		assert.Equal(t, 9, *o.Items[0].FulfillmentProvider)
		assert.Len(t, fallbackActorEntries(rec), 1)
		assert.Equal(t, OrderStatusProcessing, o.Status)
	})

	t.Run("an admin re-send clears the automatic flag", func(t *testing.T) {
		backup := &fakeProvider{id: 9, err: fmt.Errorf("x: %w", provider.ErrPending), res: provider.Result{Reference: "B_2"}}
		st, repo, _, _ := newFallbackSettler(t, refusedFake(7), backup)
		o := settlerOrder(repo, func(o *Order) {
			o.Fulfillment.SupplierAttempt = 1
			o.Fulfillment.SupplierAttemptAuto = true
			o.Fulfillment.SupplierAmbiguous = true
			stuckAt(o)
		})
		_, err := st.Reroute(ctx, o.ID, RerouteInput{ProviderID: 9, UpstreamProductID: "U-9"}, "admin")
		require.NoError(t, err)
		assert.False(t, o.Fulfillment.SupplierAttemptAuto)
		assert.False(t, o.Fulfillment.SupplierAmbiguous)
		assert.True(t, adminRouted(o))
	})
}
