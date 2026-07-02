package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
)

// fakeRecorder captures audit entries in memory.
type fakeRecorder struct {
	entries []audit.Entry
}

func (f *fakeRecorder) Record(_ context.Context, e audit.Entry) {
	f.entries = append(f.entries, e)
}

// fakeWalletRepo is an in-memory wallet.Repository whose ledger insert can be
// forced to fail (the mandatory-ledger compensation path).
type fakeWalletRepo struct {
	balances   map[bson.ObjectID]float64
	failCreate bool
	created    []*wallet.WalletTransaction
}

func (f *fakeWalletRepo) Create(_ context.Context, tx *wallet.WalletTransaction) error {
	if f.failCreate {
		return errors.New("ledger down")
	}
	f.created = append(f.created, tx)
	return nil
}

func (f *fakeWalletRepo) FindByUserID(_ context.Context, _ bson.ObjectID) ([]*wallet.WalletTransaction, error) {
	return nil, nil
}

func (f *fakeWalletRepo) GetBalance(_ context.Context, id bson.ObjectID) (float64, error) {
	return f.balances[id], nil
}

func (f *fakeWalletRepo) Credit(_ context.Context, id bson.ObjectID, amount float64) (float64, error) {
	f.balances[id] += amount
	return f.balances[id], nil
}

func (f *fakeWalletRepo) Debit(_ context.Context, id bson.ObjectID, amount float64) (float64, error) {
	if f.balances[id] < amount {
		return 0, errors.New("insufficient funds")
	}
	f.balances[id] -= amount
	return f.balances[id], nil
}

func (f *fakeWalletRepo) TopUpsForDay(_ context.Context, _ time.Time) (float64, error) {
	return 0, nil
}

// newAdminFixture builds an adminHandler over in-memory fakes seeded with the
// given users.
func newAdminFixture(users ...*User) (*adminHandler, *fakeUserRepo, *fakeWalletRepo, *fakeRecorder) {
	repo := newFakeUserRepo()
	for _, u := range users {
		repo.byID[u.ID] = u
	}
	w := &fakeWalletRepo{balances: map[bson.ObjectID]float64{}}
	rec := &fakeRecorder{}
	return &adminHandler{repo: repo, wallet: w, rec: rec}, repo, w, rec
}

// doJSON performs a chi-routed request against handler with the {id} param set.
func doJSON(t *testing.T, method, path string, body string, register func(r chi.Router)) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	register(r)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestUpdateRoleLastAdminGuard(t *testing.T) {
	admin := &User{ID: bson.NewObjectID(), Role: RoleAdmin, Status: StatusActive}
	a, _, _, rec := newAdminFixture(admin)

	rr := doJSON(t, http.MethodPut, "/users/"+admin.ID.Hex()+"/role", `{"role":"customer"}`,
		func(r chi.Router) { r.Put("/users/{id}/role", a.updateRole) })

	if rr.Code != http.StatusConflict {
		t.Fatalf("demoting the only admin: got status %d, want 409", rr.Code)
	}
	if len(rec.entries) != 0 {
		t.Fatalf("blocked demotion must not be audited, got %d entries", len(rec.entries))
	}
}

func TestUpdateRoleWithAnotherAdminSucceeds(t *testing.T) {
	admin1 := &User{ID: bson.NewObjectID(), Role: RoleAdmin, Status: StatusActive}
	admin2 := &User{ID: bson.NewObjectID(), Role: RoleAdmin, Status: StatusActive}
	a, repo, _, rec := newAdminFixture(admin1, admin2)

	rr := doJSON(t, http.MethodPut, "/users/"+admin1.ID.Hex()+"/role", `{"role":"customer"}`,
		func(r chi.Router) { r.Put("/users/{id}/role", a.updateRole) })

	if rr.Code != http.StatusOK {
		t.Fatalf("demoting one of two admins: got status %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}
	if repo.byID[admin1.ID].Role != RoleCustomer {
		t.Fatalf("role not updated, got %q", repo.byID[admin1.ID].Role)
	}
	if len(rec.entries) != 1 || rec.entries[0].Action != audit.ActionRoleChange {
		t.Fatalf("expected one role_change audit entry, got %+v", rec.entries)
	}
}

func TestUpdateStatusLastAdminGuard(t *testing.T) {
	admin := &User{ID: bson.NewObjectID(), Role: RoleAdmin, Status: StatusActive}
	a, _, _, _ := newAdminFixture(admin)

	rr := doJSON(t, http.MethodPut, "/users/"+admin.ID.Hex()+"/status", `{"status":"suspended"}`,
		func(r chi.Router) { r.Put("/users/{id}/status", a.updateStatus) })

	if rr.Code != http.StatusConflict {
		t.Fatalf("suspending the only admin: got status %d, want 409", rr.Code)
	}
}

func TestWalletAdjustLedgerFailureReverts(t *testing.T) {
	customer := &User{ID: bson.NewObjectID(), Role: RoleCustomer, Status: StatusActive}
	a, _, w, rec := newAdminFixture(customer)
	w.balances[customer.ID] = 40
	w.failCreate = true

	rr := doJSON(t, http.MethodPost, "/users/"+customer.ID.Hex()+"/wallet-adjust",
		`{"direction":"credit","amount":25,"reason":"promo"}`,
		func(r chi.Router) { r.Post("/users/{id}/wallet-adjust", a.walletAdjust) })

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("ledger failure: got status %d, want 500", rr.Code)
	}
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error.Code != "LEDGER_WRITE_FAILED" {
		t.Fatalf("error code = %q, want LEDGER_WRITE_FAILED", resp.Error.Code)
	}
	if got := w.balances[customer.ID]; got != 40 {
		t.Fatalf("balance after reverted credit = %v, want 40", got)
	}
	if len(rec.entries) != 0 {
		t.Fatalf("failed adjustment must not be audited, got %d entries", len(rec.entries))
	}
}

func TestWalletAdjustSuccessAuditsAndWritesLedger(t *testing.T) {
	customer := &User{ID: bson.NewObjectID(), Role: RoleCustomer, Status: StatusActive}
	a, _, w, rec := newAdminFixture(customer)
	w.balances[customer.ID] = 10

	rr := doJSON(t, http.MethodPost, "/users/"+customer.ID.Hex()+"/wallet-adjust",
		`{"direction":"credit","amount":15,"reason":"manual top-up"}`,
		func(r chi.Router) { r.Post("/users/{id}/wallet-adjust", a.walletAdjust) })

	if rr.Code != http.StatusOK {
		t.Fatalf("adjust: got status %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}
	if got := w.balances[customer.ID]; got != 25 {
		t.Fatalf("balance = %v, want 25", got)
	}
	if len(w.created) != 1 || w.created[0].Type != wallet.TxTypeAdjustment {
		t.Fatalf("expected one adjustment ledger row, got %+v", w.created)
	}
	if len(rec.entries) != 1 || rec.entries[0].Action != audit.ActionWalletAdjust {
		t.Fatalf("expected one wallet.adjust audit entry, got %+v", rec.entries)
	}
}
