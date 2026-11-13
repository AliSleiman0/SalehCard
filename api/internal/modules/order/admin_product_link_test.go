package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// fakeProducts satisfies productLookup over a fixed set of live product ids.
type fakeProducts struct {
	live  map[bson.ObjectID]bool
	err   error
	calls int
}

func (f *fakeProducts) ExistingIDs(_ context.Context, ids []bson.ObjectID) (map[bson.ObjectID]bool, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	out := map[bson.ObjectID]bool{}
	for _, id := range ids {
		if f.live[id] {
			out[id] = true
		}
	}
	return out, nil
}

// orderWithProducts builds an order with one line per product id.
func orderWithProducts(pids ...bson.ObjectID) *Order {
	o := &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Status: OrderStatusCompleted}
	for _, pid := range pids {
		o.Items = append(o.Items, OrderItem{ProductID: pid, Qty: 1})
	}
	return o
}

func decodeMissing(t *testing.T, body []byte) []string {
	t.Helper()
	var env struct {
		Data struct {
			MissingProductIDs []string `json:"missingProductIds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	return env.Data.MissingProductIDs
}

// The detail view reports a line whose product was deleted, once, and not the
// lines whose product still exists — that drives link vs plain text in admin.
func TestAdminDetail_ReportsDeletedProducts(t *testing.T) {
	live, gone := bson.NewObjectID(), bson.NewObjectID()
	o := orderWithProducts(live, gone, gone)
	h, _, _, _ := newActionFixture(o)
	h.products = &fakeProducts{live: map[bson.ObjectID]bool{live: true}}

	rr := doAdmin(t, http.MethodGet, "/orders/"+o.ID.Hex(), "", func(r chi.Router) { r.Get("/orders/{id}", h.detail) })
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body)
	}
	if got := decodeMissing(t, rr.Body.Bytes()); !slices.Equal(got, []string{gone.Hex()}) {
		t.Fatalf("missingProductIds = %v, want [%s]", got, gone.Hex())
	}
}

// When every product exists the field is omitted entirely.
func TestAdminDetail_AllProductsLive_OmitsField(t *testing.T) {
	live := bson.NewObjectID()
	o := orderWithProducts(live)
	h, _, _, _ := newActionFixture(o)
	h.products = &fakeProducts{live: map[bson.ObjectID]bool{live: true}}

	rr := doAdmin(t, http.MethodGet, "/orders/"+o.ID.Hex(), "", func(r chi.Router) { r.Get("/orders/{id}", h.detail) })
	var raw struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw.Data["missingProductIds"]; ok {
		t.Fatalf("missingProductIds present with every product live: %s", rr.Body)
	}
}

// A failed product lookup must not fail the order read — it reports nothing
// missing (the admin links; the product editor shows its own not-found).
func TestAdminDetail_ProductLookupError_StillServes(t *testing.T) {
	o := orderWithProducts(bson.NewObjectID())
	h, _, _, _ := newActionFixture(o)
	h.products = &fakeProducts{err: errors.New("cosmos down")}

	rr := doAdmin(t, http.MethodGet, "/orders/"+o.ID.Hex(), "", func(r chi.Router) { r.Get("/orders/{id}", h.detail) })
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body)
	}
	if got := decodeMissing(t, rr.Body.Bytes()); len(got) != 0 {
		t.Fatalf("missingProductIds = %v, want none on lookup error", got)
	}
}

// The list resolves product existence for the whole page in ONE query and
// attaches each order's own missing ids.
func TestAdminList_ReportsDeletedProductsPerOrder(t *testing.T) {
	live, gone := bson.NewObjectID(), bson.NewObjectID()
	a, b := orderWithProducts(live), orderWithProducts(gone, live)
	h, _, _, _ := newActionFixture(a, b)
	fp := &fakeProducts{live: map[bson.ObjectID]bool{live: true}}
	h.products = fp

	rr := doAdmin(t, http.MethodGet, "/orders", "", func(r chi.Router) { r.Get("/orders", h.list) })
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body)
	}
	if fp.calls != 1 {
		t.Fatalf("product lookups = %d, want 1 batched query", fp.calls)
	}
	var env struct {
		Data []struct {
			ID                string   `json:"id"`
			MissingProductIDs []string `json:"missingProductIds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Data) != 2 {
		t.Fatalf("got %d orders, want 2", len(env.Data))
	}
	for _, v := range env.Data {
		switch v.ID {
		case a.ID.Hex():
			if len(v.MissingProductIDs) != 0 {
				t.Errorf("order a: missing = %v, want none", v.MissingProductIDs)
			}
		case b.ID.Hex():
			if !slices.Equal(v.MissingProductIDs, []string{gone.Hex()}) {
				t.Errorf("order b: missing = %v, want [%s]", v.MissingProductIDs, gone.Hex())
			}
		}
	}
}

// A line with no product id was never "deleted since" — it must not be
// reported (the field's contract), even though it can't be linked either.
func TestMissingProductIDs_SkipsZeroProductID(t *testing.T) {
	gone := bson.NewObjectID()
	o := orderWithProducts(bson.ObjectID{}, gone)
	got := missingProductIDs(o, map[bson.ObjectID]bool{})
	if !slices.Equal(got, []string{gone.Hex()}) {
		t.Fatalf("missingProductIds = %v, want only [%s]", got, gone.Hex())
	}
}
