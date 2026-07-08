package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
)

// asSuperAdmin is chi middleware injecting super-admin claims (wildcard perm)
// into the request context, standing in for the AdminOnly middleware chain.
func asSuperAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.ContextWithClaims(r.Context(), &auth.Claims{
			UserID: bson.NewObjectID().Hex(),
			Email:  "super@test.local",
			Role:   "admin",
			Perms:  []string{auth.PermAll},
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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
		func(r chi.Router) { r.With(asSuperAdmin).Put("/users/{id}/role", a.updateRole) })

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
		func(r chi.Router) { r.With(asSuperAdmin).Put("/users/{id}/role", a.updateRole) })

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

// asLimitedAdmin injects claims for an admin holding a custom role (users.manage
// but no wildcard) — it must not be able to grant or revoke admin access.
func asLimitedAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.ContextWithClaims(r.Context(), &auth.Claims{
			UserID: bson.NewObjectID().Hex(),
			Email:  "limited@test.local",
			Role:   "admin",
			Perms:  []string{"users.view", "users.manage"},
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TestUpdateRoleEscalationBlockedForLimitedAdmin(t *testing.T) {
	customer := &User{ID: bson.NewObjectID(), Role: RoleCustomer, Status: StatusActive}
	super := &User{ID: bson.NewObjectID(), Role: RoleAdmin, Status: StatusActive}
	a, repo, _, rec := newAdminFixture(customer, super)

	// Granting admin access requires a super-admin actor.
	rr := doJSON(t, http.MethodPut, "/users/"+customer.ID.Hex()+"/role", `{"role":"admin"}`,
		func(r chi.Router) { r.With(asLimitedAdmin).Put("/users/{id}/role", a.updateRole) })
	if rr.Code != http.StatusForbidden {
		t.Fatalf("limited admin granting admin: got status %d, want 403", rr.Code)
	}
	if repo.byID[customer.ID].Role != RoleCustomer {
		t.Fatalf("role must not change, got %q", repo.byID[customer.ID].Role)
	}

	// Demoting an existing admin also requires a super-admin actor.
	rr = doJSON(t, http.MethodPut, "/users/"+super.ID.Hex()+"/role", `{"role":"customer"}`,
		func(r chi.Router) { r.With(asLimitedAdmin).Put("/users/{id}/role", a.updateRole) })
	if rr.Code != http.StatusForbidden {
		t.Fatalf("limited admin demoting an admin: got status %d, want 403", rr.Code)
	}

	// customer<->reseller changes stay open to users.manage.
	rr = doJSON(t, http.MethodPut, "/users/"+customer.ID.Hex()+"/role", `{"role":"reseller"}`,
		func(r chi.Router) { r.With(asLimitedAdmin).Put("/users/{id}/role", a.updateRole) })
	if rr.Code != http.StatusOK {
		t.Fatalf("limited admin customer->reseller: got status %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}
	if len(rec.entries) != 1 {
		t.Fatalf("expected exactly the reseller change audited, got %+v", rec.entries)
	}
}

func TestBulkStatusExcludesAdminsOnActivate(t *testing.T) {
	// A suspended admin plus a normal customer. A limited admin bulk-activating
	// both must reactivate only the customer — the admin account is managed
	// solely through the super-admin-gated single endpoint.
	admin := &User{ID: bson.NewObjectID(), Role: RoleAdmin, Status: StatusSuspended}
	customer := &User{ID: bson.NewObjectID(), Role: RoleCustomer, Status: StatusSuspended}
	a, repo, _, _ := newAdminFixture(admin, customer)

	body, _ := json.Marshal(map[string]any{
		"ids":    []string{admin.ID.Hex(), customer.ID.Hex()},
		"action": "activate",
	})
	rr := doJSON(t, http.MethodPost, "/users/bulk", string(body),
		func(r chi.Router) { r.Post("/users/bulk", a.bulkStatus) })

	if rr.Code != http.StatusOK {
		t.Fatalf("bulk activate: got status %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}
	if repo.byID[admin.ID].Status != StatusSuspended {
		t.Fatalf("admin account must NOT be reactivated by bulk, got status %q", repo.byID[admin.ID].Status)
	}
	if repo.byID[customer.ID].Status != StatusActive {
		t.Fatalf("customer should be reactivated, got status %q", repo.byID[customer.ID].Status)
	}
}

func TestUpdateStatusLastAdminGuard(t *testing.T) {
	admin := &User{ID: bson.NewObjectID(), Role: RoleAdmin, Status: StatusActive}
	a, _, _, _ := newAdminFixture(admin)

	rr := doJSON(t, http.MethodPut, "/users/"+admin.ID.Hex()+"/status", `{"status":"suspended"}`,
		func(r chi.Router) { r.With(asSuperAdmin).Put("/users/{id}/status", a.updateStatus) })

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

// fakeSMSSender records every send under a mutex; the bulk-SMS fan-out runs in a
// detached goroutine, so tests wait on recorded() rather than reading immediately.
type fakeSMSSender struct {
	mu   sync.Mutex
	sent []string
}

func (f *fakeSMSSender) Send(_ context.Context, phone, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, phone)
	return nil
}

func (f *fakeSMSSender) recorded() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

// waitForSends polls until the fake has recorded want sends or a short deadline
// elapses (the fan-out goroutine sends serially, off the request path).
func waitForSends(t *testing.T, s *fakeSMSSender, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.recorded() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := s.recorded(); got != want {
		t.Fatalf("bulk-sms sends = %d, want %d", got, want)
	}
}

func strptr(s string) *string { return &s }

func TestBulkSMSSkipsPhonelessAndDeleted(t *testing.T) {
	valid1 := &User{ID: bson.NewObjectID(), Status: StatusActive, Phone: strptr("+96170000001")}
	valid2 := &User{ID: bson.NewObjectID(), Status: StatusActive, Phone: strptr("+96170000002")}
	noPhone := &User{ID: bson.NewObjectID(), Status: StatusActive, Phone: nil}
	deleted := &User{ID: bson.NewObjectID(), Status: StatusDeleted, Phone: strptr("+96170000003")}
	a, _, _, rec := newAdminFixture(valid1, valid2, noPhone, deleted)
	sender := &fakeSMSSender{}
	a.smsSender, a.maxBulkSMS = sender, 200

	ids := []string{valid1.ID.Hex(), valid2.ID.Hex(), noPhone.ID.Hex(), deleted.ID.Hex()}
	body, _ := json.Marshal(map[string]any{"ids": ids, "message": "Hello from SalehCard"})
	rr := doJSON(t, http.MethodPost, "/users/bulk-sms", string(body),
		func(r chi.Router) { r.Post("/users/bulk-sms", a.bulkSMS) })

	if rr.Code != http.StatusOK {
		t.Fatalf("bulk-sms: got status %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Data struct {
			Queued int `json:"queued"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Queued != 2 {
		t.Fatalf("queued = %d, want 2 (phone-less and deleted skipped)", resp.Data.Queued)
	}
	waitForSends(t, sender, 2)
	if len(rec.entries) != 1 || rec.entries[0].Action != audit.ActionUserBulkSMS {
		t.Fatalf("expected one user.bulk_sms audit entry, got %+v", rec.entries)
	}
}

func TestBulkSMSRejectsOverCap(t *testing.T) {
	u1 := &User{ID: bson.NewObjectID(), Status: StatusActive, Phone: strptr("+96170000001")}
	u2 := &User{ID: bson.NewObjectID(), Status: StatusActive, Phone: strptr("+96170000002")}
	a, _, _, rec := newAdminFixture(u1, u2)
	sender := &fakeSMSSender{}
	a.smsSender, a.maxBulkSMS = sender, 1

	body, _ := json.Marshal(map[string]any{"ids": []string{u1.ID.Hex(), u2.ID.Hex()}, "message": "hi"})
	rr := doJSON(t, http.MethodPost, "/users/bulk-sms", string(body),
		func(r chi.Router) { r.Post("/users/bulk-sms", a.bulkSMS) })

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("over-cap: got status %d, want 400", rr.Code)
	}
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error.Code != "BULK_SMS_LIMIT" {
		t.Fatalf("error code = %q, want BULK_SMS_LIMIT", resp.Error.Code)
	}
	if sender.recorded() != 0 {
		t.Fatalf("over-cap send must not dispatch any SMS, got %d", sender.recorded())
	}
	if len(rec.entries) != 0 {
		t.Fatalf("rejected send must not be audited, got %d entries", len(rec.entries))
	}
}

func TestBulkSMSRejectsLongMessage(t *testing.T) {
	u := &User{ID: bson.NewObjectID(), Status: StatusActive, Phone: strptr("+96170000001")}
	a, _, _, _ := newAdminFixture(u)
	a.smsSender, a.maxBulkSMS = &fakeSMSSender{}, 200

	body, _ := json.Marshal(map[string]any{"ids": []string{u.ID.Hex()}, "message": strings.Repeat("x", 161)})
	rr := doJSON(t, http.MethodPost, "/users/bulk-sms", string(body),
		func(r chi.Router) { r.Post("/users/bulk-sms", a.bulkSMS) })

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("161-char message: got status %d, want 400", rr.Code)
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
