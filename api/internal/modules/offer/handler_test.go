package offer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// stubRepo implements Repository; it records whether ListLive was consulted.
type stubRepo struct {
	listLiveCalled bool
}

func (s *stubRepo) FindByID(context.Context, bson.ObjectID) (*Offer, error) { return nil, nil }
func (s *stubRepo) List(context.Context, OfferFilter, pagination.Params) ([]*Offer, int64, error) {
	return nil, 0, nil
}
func (s *stubRepo) ListLive(context.Context, time.Time) ([]*Offer, error) {
	s.listLiveCalled = true
	return []*Offer{}, nil
}
func (s *stubRepo) FindLiveByProduct(context.Context, bson.ObjectID, time.Time) (*Offer, error) {
	return nil, nil
}
func (s *stubRepo) Create(context.Context, *Offer) error { return nil }
func (s *stubRepo) Update(context.Context, bson.ObjectID, OfferUpdate) (*Offer, error) {
	return nil, nil
}
func (s *stubRepo) Delete(context.Context, bson.ObjectID) error { return nil }

// listBody decodes the response envelope's data array.
func listBody(t *testing.T, rec *httptest.ResponseRecorder) []any {
	t.Helper()
	var envelope struct {
		Data []any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v (%s)", err, rec.Body.String())
	}
	return envelope.Data
}

func TestListSuppressedForResellers(t *testing.T) {
	repo := &stubRepo{}
	h := NewHandler(repo, nil) // products never consulted on the reseller path

	req := httptest.NewRequest(http.MethodGet, "/api/v1/offers", nil)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{
		UserID: bson.NewObjectID().Hex(),
		Role:   "reseller",
	}))
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	if data := listBody(t, rec); len(data) != 0 {
		t.Fatalf("reseller offers = %v, want empty list", data)
	}
	if repo.listLiveCalled {
		t.Fatal("ListLive consulted for a reseller — offers must be suppressed before the query")
	}
}

func TestListQueriesLiveOffersForCustomers(t *testing.T) {
	repo := &stubRepo{}
	h := NewHandler(repo, nil) // empty live list → products never consulted

	req := httptest.NewRequest(http.MethodGet, "/api/v1/offers", nil)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{
		UserID: bson.NewObjectID().Hex(),
		Role:   "customer",
	}))
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	if !repo.listLiveCalled {
		t.Fatal("ListLive not consulted for a customer")
	}
}
