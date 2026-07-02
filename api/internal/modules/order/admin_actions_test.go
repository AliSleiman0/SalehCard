package order

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
)

// fakeRefunder records refund calls and can be forced to fail.
type fakeRefunder struct {
	fail     bool
	refunded []float64
}

func (f *fakeRefunder) Refund(_ context.Context, _ bson.ObjectID, amount float64, _ string) (*wallet.WalletTransaction, error) {
	if f.fail {
		return nil, errors.New("wallet down")
	}
	f.refunded = append(f.refunded, amount)
	return &wallet.WalletTransaction{Amount: amount}, nil
}

// fakeCustomers satisfies customerLookup with no users (respondFresh tolerates it).
type fakeCustomers struct{}

func (fakeCustomers) FindByID(_ context.Context, _ bson.ObjectID) (*user.User, error) {
	return nil, errors.New("not found")
}
func (fakeCustomers) FindByIDs(_ context.Context, _ []bson.ObjectID) ([]*user.User, error) {
	return nil, nil
}
func (fakeCustomers) FindByEmailLike(_ context.Context, _ string) ([]*user.User, error) {
	return nil, nil
}

// fakeAuditRec captures audit entries.
type fakeAuditRec struct {
	entries []audit.Entry
}

func (f *fakeAuditRec) Record(_ context.Context, e audit.Entry) { f.entries = append(f.entries, e) }

func newActionFixture(orders ...*Order) (*adminHandler, *fakeOrderRepo, *fakeRefunder, *fakeAuditRec) {
	repo := newFakeOrderRepo()
	for _, o := range orders {
		repo.byID[o.ID] = o
	}
	w := &fakeRefunder{}
	rec := &fakeAuditRec{}
	return &adminHandler{repo: repo, users: fakeCustomers{}, wallet: w, rec: rec}, repo, w, rec
}

func doAdmin(t *testing.T, method, path, body string, register func(r chi.Router)) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	register(r)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestAdminRefundProcessingWalletOrder(t *testing.T) {
	o := &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Total: 30,
		PaymentMethod: PaymentMethodWallet, Status: OrderStatusProcessing}
	a, repo, w, rec := newActionFixture(o)

	rr := doAdmin(t, http.MethodPost, "/orders/"+o.ID.Hex()+"/refund", `{"reason":"customer request"}`,
		func(r chi.Router) { r.Post("/orders/{id}/refund", a.refund) })

	if rr.Code != http.StatusOK {
		t.Fatalf("refund: got %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}
	if repo.byID[o.ID].Status != OrderStatusRefunded {
		t.Fatalf("status = %q, want refunded", repo.byID[o.ID].Status)
	}
	if len(w.refunded) != 1 || w.refunded[0] != 30 {
		t.Fatalf("wallet refunds = %v, want [30]", w.refunded)
	}
	if len(rec.entries) != 1 || rec.entries[0].Action != audit.ActionOrderRefund {
		t.Fatalf("expected one order.refund audit entry, got %+v", rec.entries)
	}
}

func TestAdminRefundTwiceConflicts(t *testing.T) {
	o := &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Total: 10,
		PaymentMethod: PaymentMethodWallet, Status: OrderStatusCompleted}
	a, _, w, _ := newActionFixture(o)
	reg := func(r chi.Router) { r.Post("/orders/{id}/refund", a.refund) }

	first := doAdmin(t, http.MethodPost, "/orders/"+o.ID.Hex()+"/refund", `{}`, reg)
	second := doAdmin(t, http.MethodPost, "/orders/"+o.ID.Hex()+"/refund", `{}`, reg)

	if first.Code != http.StatusOK {
		t.Fatalf("first refund: got %d, want 200", first.Code)
	}
	if second.Code != http.StatusConflict {
		t.Fatalf("second refund: got %d, want 409", second.Code)
	}
	if len(w.refunded) != 1 {
		t.Fatalf("wallet credited %d times, want exactly once", len(w.refunded))
	}
}

func TestAdminRefundWalletFailureRevertsStatus(t *testing.T) {
	o := &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Total: 20,
		PaymentMethod: PaymentMethodWallet, Status: OrderStatusProcessing}
	a, repo, w, rec := newActionFixture(o)
	w.fail = true

	rr := doAdmin(t, http.MethodPost, "/orders/"+o.ID.Hex()+"/refund", `{}`,
		func(r chi.Router) { r.Post("/orders/{id}/refund", a.refund) })

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("refund with wallet failure: got %d, want 500", rr.Code)
	}
	if repo.byID[o.ID].Status != OrderStatusProcessing {
		t.Fatalf("status after failed refund = %q, want processing (reverted)", repo.byID[o.ID].Status)
	}
	if len(rec.entries) != 0 {
		t.Fatalf("failed refund must not be audited, got %+v", rec.entries)
	}
}

func TestAdminCompleteProcessingOrder(t *testing.T) {
	o := &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Total: 15,
		PaymentMethod: PaymentMethodWallet, Status: OrderStatusProcessing}
	a, repo, _, rec := newActionFixture(o)

	rr := doAdmin(t, http.MethodPut, "/orders/"+o.ID.Hex()+"/status",
		`{"status":"completed","note":"credited in game","transferRef":"TX-99"}`,
		func(r chi.Router) { r.Put("/orders/{id}/status", a.updateStatus) })

	if rr.Code != http.StatusOK {
		t.Fatalf("complete: got %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}
	if repo.byID[o.ID].Status != OrderStatusCompleted {
		t.Fatalf("status = %q, want completed", repo.byID[o.ID].Status)
	}
	if len(rec.entries) != 1 || rec.entries[0].Action != audit.ActionOrderStatus {
		t.Fatalf("expected one order.status audit entry, got %+v", rec.entries)
	}
}

func TestAdminCompleteRejectsNonProcessing(t *testing.T) {
	o := &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Status: OrderStatusCompleted}
	a, _, _, _ := newActionFixture(o)

	rr := doAdmin(t, http.MethodPut, "/orders/"+o.ID.Hex()+"/status", `{"status":"completed"}`,
		func(r chi.Router) { r.Put("/orders/{id}/status", a.updateStatus) })

	if rr.Code != http.StatusConflict {
		t.Fatalf("completing a completed order: got %d, want 409", rr.Code)
	}
}

func TestAdminStatusRejectsNonCompleted(t *testing.T) {
	o := &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Status: OrderStatusProcessing}
	a, _, _, _ := newActionFixture(o)

	rr := doAdmin(t, http.MethodPut, "/orders/"+o.ID.Hex()+"/status", `{"status":"refunded"}`,
		func(r chi.Router) { r.Put("/orders/{id}/status", a.updateStatus) })

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("setting status=refunded via /status: got %d, want 400", rr.Code)
	}
}

func TestAdminMarkFailedPendingOrder(t *testing.T) {
	o := &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Total: 25,
		PaymentMethod: PaymentMethodWallet, Status: OrderStatusPending}
	a, repo, w, rec := newActionFixture(o)

	rr := doAdmin(t, http.MethodPost, "/orders/"+o.ID.Hex()+"/fail", `{"reason":"stuck mid-placement"}`,
		func(r chi.Router) { r.Post("/orders/{id}/fail", a.markFailed) })

	if rr.Code != http.StatusOK {
		t.Fatalf("mark failed: got %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}
	if repo.byID[o.ID].Status != OrderStatusFailed {
		t.Fatalf("status = %q, want failed", repo.byID[o.ID].Status)
	}
	if len(w.refunded) != 0 {
		t.Fatalf("mark failed must not move money, wallet refunds = %v", w.refunded)
	}
	if len(rec.entries) != 1 || rec.entries[0].Action != audit.ActionOrderStatus {
		t.Fatalf("expected one order.status audit entry, got %+v", rec.entries)
	}
}

func TestAdminMarkFailedRejectsNonPending(t *testing.T) {
	o := &Order{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Status: OrderStatusProcessing}
	a, _, _, rec := newActionFixture(o)

	rr := doAdmin(t, http.MethodPost, "/orders/"+o.ID.Hex()+"/fail", `{}`,
		func(r chi.Router) { r.Post("/orders/{id}/fail", a.markFailed) })

	if rr.Code != http.StatusConflict {
		t.Fatalf("failing a processing order: got %d, want 409", rr.Code)
	}
	if len(rec.entries) != 0 {
		t.Fatalf("rejected transition must not be audited, got %+v", rec.entries)
	}
}
