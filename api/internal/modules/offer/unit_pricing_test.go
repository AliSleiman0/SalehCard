package offer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// fakeProducts is a product.Service serving one product by id.
type fakeProducts struct{ p *product.Product }

func (f fakeProducts) FindAll(context.Context, product.ListFilter, pagination.Params) ([]product.Product, int64, error) {
	return nil, 0, nil
}
func (f fakeProducts) FindByID(_ context.Context, id string) (*product.Product, error) {
	if f.p == nil || f.p.ID.Hex() != id {
		return nil, apperrors.ErrNotFound
	}
	return f.p, nil
}
func (f fakeProducts) Create(context.Context, product.CreateProductInput) (*product.Product, error) {
	return nil, nil
}
func (f fakeProducts) Update(context.Context, string, product.UpdateProductInput) (*product.Product, error) {
	return nil, nil
}
func (f fakeProducts) Delete(context.Context, string) error                   { return nil }
func (f fakeProducts) Bulk(context.Context, product.BulkInput) (int64, error) { return 0, nil }

func fp(v float64) *float64 { return &v }

// coinProduct: $10 per 90,000 coins, amount 9,000–900,000.
func coinProduct(unit bool) *product.Product {
	p := &product.Product{
		ID:        bson.NewObjectID(),
		Available: true,
		Variants:  []product.Variant{{ID: bson.NewObjectID(), Price: 10}},
		InputFields: []product.InputField{{Key: "amount", Type: product.InputFieldAmount,
			Constraints: &product.InputFieldConstraints{Min: fp(9000), Max: fp(900000)}}},
	}
	if unit {
		p.UnitPricing = &product.UnitPricing{FieldKey: "amount", BlockUnits: 90000}
	}
	return p
}

func postOffer(t *testing.T, a *adminHandler, body offerBody) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	a.create(rec, httptest.NewRequest(http.MethodPost, "/offers", bytes.NewReader(b)))
	return rec
}

func TestAdminCreate_RejectsFixedOfferOnUnitPricedProduct(t *testing.T) {
	p := coinProduct(true)
	a := &adminHandler{repo: &stubRepo{}, products: fakeProducts{p: p}}

	rec := postOffer(t, a, offerBody{ProductID: p.ID.Hex(), DiscountType: DiscountFixed, DiscountValue: 1, Active: true})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}
	var env struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != ErrCodeFixedUnitPriced || env.Error.Message == "" {
		t.Fatalf("error = %+v, want %s with a message", env.Error, ErrCodeFixedUnitPriced)
	}

	// Percent offers on the same product, and fixed offers on a fixed product, pass.
	if rec := postOffer(t, a, offerBody{ProductID: p.ID.Hex(), DiscountType: DiscountPercent, DiscountValue: 10}); rec.Code != http.StatusOK {
		t.Fatalf("percent offer code = %d (%s)", rec.Code, rec.Body.String())
	}
	fixed := coinProduct(false)
	a.products = fakeProducts{p: fixed}
	if rec := postOffer(t, a, offerBody{ProductID: fixed.ID.Hex(), DiscountType: DiscountFixed, DiscountValue: 1}); rec.Code != http.StatusOK {
		t.Fatalf("fixed offer on fixed product code = %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestOfferFromPrice_UnitPriced(t *testing.T) {
	p := coinProduct(true)
	pct := &Offer{DiscountType: DiscountPercent, DiscountValue: 20}
	if got := fromPrice(p); got != 1 { // 10 × 9,000 ÷ 90,000
		t.Fatalf("fromPrice = %v, want 1", got)
	}
	if got := offerFromPrice(pct, p); got != 0.8 {
		t.Fatalf("percent offerFromPrice = %v, want 0.8", got)
	}
	fixed := &Offer{DiscountType: DiscountFixed, DiscountValue: 3}
	if AppliesTo(fixed, p) {
		t.Fatal("fixed offer must not apply to a per-unit product")
	}
	if got := offerFromPrice(fixed, p); got != 1 {
		t.Fatalf("fixed offerFromPrice = %v, want unchanged 1", got)
	}
	if got := offerFromPrice(fixed, coinProduct(false)); got != 7 {
		t.Fatalf("fixed offer on fixed product = %v, want 7", got)
	}
}
