package product

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// --- filter -----------------------------------------------------------------

// Every listing excludes the trash unless it asks for the trash explicitly.
func TestBuildFilter_ExcludesTrashByDefault(t *testing.T) {
	for _, status := range []string{"", "active", "draft", "out"} {
		f := buildFilter(ListFilter{Status: status})
		if v, ok := lookup(f, "deletedAt"); !ok || v != nil {
			t.Fatalf("status %q: deletedAt = %#v (present=%v), want {deletedAt: null}", status, v, ok)
		}
	}
	// The customer list path (available=true) also excludes the trash.
	yes := true
	if v, ok := lookup(buildFilter(ListFilter{Available: &yes}), "deletedAt"); !ok || v != nil {
		t.Fatalf("customer list: deletedAt = %#v, want null", v)
	}
}

func TestBuildFilter_DeletedListsOnlyTrash(t *testing.T) {
	f := buildFilter(ListFilter{Status: StatusDeleted})
	v, ok := lookup(f, "deletedAt")
	d, isDoc := v.(bson.D)
	if !ok || !isDoc || len(d) != 1 || d[0].Key != "$ne" || d[0].Value != nil {
		t.Fatalf("deleted view deletedAt = %#v, want {$ne: null}", v)
	}
	// The trash view must not also filter on availability: drafts and live
	// products are both in it.
	if _, ok := lookup(f, "available"); ok {
		t.Fatal("deleted view must not filter on available")
	}
}

func TestSelectorFilter(t *testing.T) {
	if _, ok := selectorFilter(Selector{IDs: []string{"not-hex"}}); ok {
		t.Fatal("a selector with no valid ids must match nothing")
	}
	oid := bson.NewObjectID()
	f, ok := selectorFilter(Selector{IDs: []string{oid.Hex()}})
	if !ok || len(f) != 1 || f[0].Key != "_id" {
		t.Fatalf("id selector = %#v", f)
	}
	f, ok = selectorFilter(Selector{Filter: &ListFilter{Status: StatusDeleted}})
	if !ok {
		t.Fatal("a filter selector always applies")
	}
	if _, has := lookup(f, "deletedAt"); !has {
		t.Fatalf("filter selector must reuse buildFilter: %#v", f)
	}
}

// andWith keeps the selector's own deletedAt clause and the write's guard in
// separate $and branches (one document can't hold the same key twice).
func TestAndTrash(t *testing.T) {
	got := andWith(bson.D{notDeleted}, inTrash)
	arr, ok := got[0].Value.(bson.A)
	if got[0].Key != "$and" || !ok || len(arr) != 2 {
		t.Fatalf("andWith = %#v", got)
	}
}

func TestPurchasable(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		p    Product
		want bool
	}{
		{"live", Product{Available: true}, true},
		{"draft", Product{}, false},
		{"trashed live", Product{Available: true, DeletedAt: &now}, false},
		{"trashed draft", Product{DeletedAt: &now}, false},
	}
	for _, c := range cases {
		if got := c.p.Purchasable(); got != c.want {
			t.Errorf("%s: Purchasable = %v, want %v", c.name, got, c.want)
		}
	}
}

// --- handlers ---------------------------------------------------------------

// trashService is a Service stub for the trash handlers: it records the bulk
// input and returns canned results/errors.
type trashService struct {
	stubService
	opErr    error
	bulkN    int64
	bulkErr  error
	bulkSeen *BulkInput
}

func (s *trashService) Delete(context.Context, string) error  { return s.opErr }
func (s *trashService) Restore(context.Context, string) error { return s.opErr }
func (s *trashService) Purge(context.Context, string) error   { return s.opErr }
func (s *trashService) Bulk(_ context.Context, in BulkInput) (int64, error) {
	s.bulkSeen = &in
	return s.bulkN, s.bulkErr
}

func withID(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func asAdmin(r *http.Request, super bool) *http.Request {
	c := &auth.Claims{UserID: bson.NewObjectID().Hex(), Role: "admin", Perms: []string{"products.manage"}}
	if super {
		c.Perms = []string{auth.PermAll}
	}
	return r.WithContext(auth.ContextWithClaims(r.Context(), c))
}

func TestGetByIDPublic_TrashedProductIsNotFound(t *testing.T) {
	now := time.Now()
	h := NewHandler(stubService{prod: &Product{Available: true, DeletedAt: &now}})
	rec := httptest.NewRecorder()
	h.GetByIDPublic(rec, withID(httptest.NewRequest(http.MethodGet, "/api/v1/products/x", nil), "x"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestTrashOps_AuditAndStatus(t *testing.T) {
	cases := []struct {
		name       string
		call       func(h *Handler) http.HandlerFunc
		wantAction string
	}{
		{"delete soft-deletes", func(h *Handler) http.HandlerFunc { return h.Delete }, audit.ActionProductSoftDelete},
		{"restore", func(h *Handler) http.HandlerFunc { return h.Restore }, audit.ActionProductRestore},
		{"purge", func(h *Handler) http.HandlerFunc { return h.Purge }, audit.ActionProductDelete},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := &fakeRecorder{}
			h := NewHandler(&trashService{})
			h.rec = rec
			w := httptest.NewRecorder()
			c.call(h)(w, withID(httptest.NewRequest(http.MethodDelete, "/p", nil), "pid"))
			if w.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want 204", w.Code)
			}
			if len(rec.entries) != 1 || rec.entries[0].Action != c.wantAction || rec.entries[0].TargetID != "pid" {
				t.Fatalf("audit = %+v, want one %s for pid", rec.entries, c.wantAction)
			}
		})
	}
}

func TestPurge_LiveProductIsConflict(t *testing.T) {
	rec := &fakeRecorder{}
	h := NewHandler(&trashService{opErr: apperrors.ErrConflict})
	h.rec = rec
	w := httptest.NewRecorder()
	h.Purge(w, withID(httptest.NewRequest(http.MethodDelete, "/p", nil), "pid"))
	if w.Code != http.StatusConflict || !bytes.Contains(w.Body.Bytes(), []byte("PRODUCT_NOT_DELETED")) {
		t.Fatalf("status = %d body = %s, want 409 PRODUCT_NOT_DELETED", w.Code, w.Body.String())
	}
	if len(rec.entries) != 0 {
		t.Fatal("a refused purge must not be audited")
	}
}

func postBulk(h *Handler, body any, super bool) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	h.Bulk(w, asAdmin(httptest.NewRequest(http.MethodPost, "/api/admin/products/bulk", bytes.NewReader(b)), super))
	return w
}

func TestBulkPurge_SuperAdminOnly(t *testing.T) {
	svc := &trashService{bulkN: 2}
	h := NewHandler(svc)
	body := map[string]any{"ids": []string{"a", "b"}, "action": "purge"}

	if w := postBulk(h, body, false); w.Code != http.StatusForbidden {
		t.Fatalf("non-super purge status = %d, want 403", w.Code)
	}
	if svc.bulkSeen != nil {
		t.Fatal("service reached for a non-super purge")
	}
	if w := postBulk(h, body, true); w.Code != http.StatusOK {
		t.Fatalf("super purge status = %d, want 200 (%s)", w.Code, w.Body.String())
	}
}

// A bulk trash call writes ONE audit entry carrying the count and target.
func TestBulk_AuditsTrashActions(t *testing.T) {
	rec := &fakeRecorder{}
	h := NewHandler(&trashService{bulkN: 40})
	h.rec = rec

	w := postBulk(h, map[string]any{"action": "delete", "filter": map[string]any{"search": "pubg"}}, false)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", w.Code, w.Body.String())
	}
	if len(rec.entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(rec.entries))
	}
	e := rec.entries[0]
	if e.Action != audit.ActionProductSoftDelete || e.TargetID != "bulk" || e.Summary["count"] != int64(40) {
		t.Fatalf("audit entry = %+v", e)
	}
	if f, ok := e.Summary["filter"].(*BulkFilter); !ok || f.Search != "pubg" {
		t.Fatalf("audit filter = %#v", e.Summary["filter"])
	}

	// Non-trash bulk actions (activate) are not audited here.
	rec.entries = nil
	postBulk(h, map[string]any{"action": "activate", "ids": []string{"a"}}, false)
	if len(rec.entries) != 0 {
		t.Fatalf("activate audited: %+v", rec.entries)
	}
}

// lookup returns the value for key in a filter document and whether it exists.
func lookup(d bson.D, key string) (any, bool) {
	for _, e := range d {
		if e.Key == key {
			return e.Value, true
		}
	}
	return nil, false
}
