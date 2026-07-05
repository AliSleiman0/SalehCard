package product

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/AliSleiman0/salehcard/api/internal/platform/idcheck"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// stubService implements Service; only FindByID is exercised by VerifyAccount.
type stubService struct {
	prod *Product
	err  error
}

func (s stubService) FindByID(context.Context, string) (*Product, error) { return s.prod, s.err }
func (stubService) FindAll(context.Context, ListFilter, pagination.Params) ([]Product, int64, error) {
	return nil, 0, nil
}
func (stubService) Create(context.Context, CreateProductInput) (*Product, error)      { return nil, nil }
func (stubService) Update(context.Context, string, UpdateProductInput) (*Product, error) {
	return nil, nil
}
func (stubService) Delete(context.Context, string) error       { return nil }
func (stubService) Bulk(context.Context, BulkInput) (int64, error) { return 0, nil }

// fakeVerifier returns a canned account/error.
type fakeVerifier struct {
	acct idcheck.Account
	err  error
}

func (f fakeVerifier) Verify(context.Context, string, string) (idcheck.Account, error) {
	return f.acct, f.err
}

// callVerify drives VerifyAccount with the given handler and JSON body, returning
// the recorder for status/body assertions.
func callVerify(h *Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/pid/verify-account", strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "pid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h.VerifyAccount(rec, req)
	return rec
}

func decodeVerify(t *testing.T, rec *httptest.ResponseRecorder) verifyAccountResponse {
	t.Helper()
	var env struct {
		Data verifyAccountResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return env.Data
}

func verifiableProduct() *Product {
	return &Product{Verification: &Verification{Provider: 1, App: "pubgm-global"}}
}

func TestVerifyAccount_Found(t *testing.T) {
	h := NewHandler(stubService{prod: verifiableProduct()})
	h.verifier = fakeVerifier{acct: idcheck.Account{Username: "Agus", Banned: true}}

	rec := callVerify(h, `{"playerId":"5204837417"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := decodeVerify(t, rec)
	if !got.Found || got.Username != "Agus" || !got.Banned {
		t.Errorf("got %+v, want found=true username=Agus banned=true", got)
	}
}

func TestVerifyAccount_NotFoundBlocks(t *testing.T) {
	h := NewHandler(stubService{prod: verifiableProduct()})
	h.verifier = fakeVerifier{err: idcheck.ErrIDNotFound}

	got := decodeVerify(t, callVerify(h, `{"playerId":"0000000000"}`))
	if got.Found || got.Reason != "id_not_found" {
		t.Errorf("got %+v, want found=false reason=id_not_found", got)
	}
}

func TestVerifyAccount_UpstreamErrorFailsOpen(t *testing.T) {
	// A capturing logger so we can also assert the sensitive playerId is not logged.
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	h := NewHandler(stubService{prod: verifiableProduct()})
	h.verifier = fakeVerifier{err: context.DeadlineExceeded}

	got := decodeVerify(t, callVerify(h, `{"playerId":"secret-player-42"}`))
	if got.Found || got.Reason != "unavailable" {
		t.Errorf("got %+v, want found=false reason=unavailable (fail-open)", got)
	}
	if strings.Contains(buf.String(), "secret-player-42") {
		t.Errorf("sensitive playerId leaked into logs: %s", buf.String())
	}
}

func TestVerifyAccount_NoVerificationConfig(t *testing.T) {
	h := NewHandler(stubService{prod: &Product{}}) // Verification == nil
	h.verifier = fakeVerifier{}
	rec := callVerify(h, `{"playerId":"123"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestVerifyAccount_EmptyPlayerID(t *testing.T) {
	h := NewHandler(stubService{prod: verifiableProduct()})
	h.verifier = fakeVerifier{}
	rec := callVerify(h, `{"playerId":"   "}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestVerifyAccount_ProductNotFound(t *testing.T) {
	h := NewHandler(stubService{err: apperrors.ErrNotFound})
	h.verifier = fakeVerifier{}
	rec := callVerify(h, `{"playerId":"123"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
