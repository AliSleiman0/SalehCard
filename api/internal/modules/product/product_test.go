package product_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// ---------------------------------------------------------------------------
// mockRepo implements product.Repository for unit tests.
// ---------------------------------------------------------------------------

type mockRepo struct {
	products   []product.Product
	err        error
	lastUpdate product.UpdateProductInput
}

func (m *mockRepo) FindAll(_ context.Context, _ product.ListFilter, _ pagination.Params) ([]product.Product, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.products, int64(len(m.products)), nil
}

func (m *mockRepo) FindByID(_ context.Context, _ string) (*product.Product, error) {
	if m.err != nil {
		return nil, m.err
	}
	if len(m.products) == 0 {
		return nil, apperrors.ErrNotFound
	}
	return &m.products[0], nil
}

func (m *mockRepo) Create(_ context.Context, in product.CreateProductInput) (*product.Product, error) {
	if m.err != nil {
		return nil, m.err
	}
	p := product.Product{
		Title:    in.Title,
		Category: in.Category,
	}
	return &p, nil
}

func (m *mockRepo) Update(_ context.Context, _ string, in product.UpdateProductInput) (*product.Product, error) {
	m.lastUpdate = in
	if m.err != nil {
		return nil, m.err
	}
	if len(m.products) == 0 {
		return nil, apperrors.ErrNotFound
	}
	return &m.products[0], nil
}

func (m *mockRepo) Delete(_ context.Context, _ string) error {
	return m.err
}

func (m *mockRepo) BulkSetAvailable(_ context.Context, ids []string, _ bool) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return int64(len(ids)), nil
}

func (m *mockRepo) BulkDelete(_ context.Context, ids []string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return int64(len(ids)), nil
}

func (m *mockRepo) FindByLegacyID(_ context.Context, _ int) (*product.Product, error) {
	if m.err != nil {
		return nil, m.err
	}
	if len(m.products) == 0 {
		return nil, apperrors.ErrNotFound
	}
	return &m.products[0], nil
}

func (m *mockRepo) Upsert(_ context.Context, in product.UpsertProductInput) (*product.Product, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &product.Product{Title: in.Title, Category: in.Category}, nil
}

// ---------------------------------------------------------------------------
// Service tests
// ---------------------------------------------------------------------------

func TestProductService_Get_NotFound(t *testing.T) {
	repo := &mockRepo{err: apperrors.ErrNotFound}
	svc := product.NewProductService(repo)

	_, err := svc.FindByID(context.Background(), "000000000000000000000000")

	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestProductService_Update_PassesInputFields(t *testing.T) {
	repo := &mockRepo{products: []product.Product{{Title: product.I18nString{En: "P"}}}}
	svc := product.NewProductService(repo)

	in := product.UpdateProductInput{
		InputFields: []product.InputField{{
			Key:        "field_1",
			Label:      product.I18nLabel{En: "Account ID", Ar: "ايدي"},
			LegacyName: "ايدي الحساب",
			Type:       product.InputFieldText,
		}},
	}

	_, err := svc.Update(context.Background(), "6a3f04c4ea6747f81d0baf8a", in)

	require.NoError(t, err)
	require.Len(t, repo.lastUpdate.InputFields, 1)
	got := repo.lastUpdate.InputFields[0]
	assert.Equal(t, "field_1", got.Key)
	assert.Equal(t, "Account ID", got.Label.En)
	assert.Equal(t, "ايدي الحساب", got.LegacyName)
}

func TestProductService_List(t *testing.T) {
	twoProducts := []product.Product{
		{Title: product.I18nString{En: "Product A"}},
		{Title: product.I18nString{En: "Product B"}},
	}

	tests := []struct {
		name      string
		repo      *mockRepo
		wantCount int
		wantTotal int64
		wantErr   bool
	}{
		{
			name:      "returns two products",
			repo:      &mockRepo{products: twoProducts},
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name:    "repository error is propagated",
			repo:    &mockRepo{err: apperrors.ErrInternal},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := product.NewProductService(tc.repo)
			products, total, err := svc.FindAll(
				context.Background(),
				product.ListFilter{},
				pagination.Params{Page: 1, Limit: 20},
			)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, products, tc.wantCount)
			assert.Equal(t, tc.wantTotal, total)
		})
	}
}

// ---------------------------------------------------------------------------
// Offer-enrichment tests
// ---------------------------------------------------------------------------

// fakeOfferLookup implements product.OfferLookup for enrichment tests.
type fakeOfferLookup struct {
	discounts map[string]product.Discount
	err       error
}

func (f *fakeOfferLookup) LiveDiscountsFor(_ context.Context, _ []string, _ time.Time) (map[string]product.Discount, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.discounts, nil
}

func productWithVariants(id bson.ObjectID, prices ...float64) product.Product {
	vs := make([]product.Variant, len(prices))
	for i, p := range prices {
		vs[i] = product.Variant{ID: bson.NewObjectID(), Price: p}
	}
	return product.Product{ID: id, Variants: vs}
}

func TestProductService_FindByID_AppliesLiveOffer(t *testing.T) {
	id := bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{productWithVariants(id, 10, 20)}}
	offers := &fakeOfferLookup{discounts: map[string]product.Discount{
		id.Hex(): {Type: "percent", Value: 30},
	}}
	svc := product.NewProductService(repo, product.WithOffers(offers))

	p, err := svc.FindByID(context.Background(), id.Hex())
	require.NoError(t, err)

	require.NotNil(t, p.Offer)
	assert.Equal(t, "percent", p.Offer.DiscountType)
	assert.Equal(t, 30.0, p.Offer.DiscountValue)
	assert.Equal(t, 10.0, p.Offer.OriginalFromPrice) // lowest variant price
	assert.Equal(t, 7.0, p.Offer.OfferFromPrice)     // 30% off 10

	require.NotNil(t, p.Variants[0].OfferPrice)
	assert.Equal(t, 7.0, *p.Variants[0].OfferPrice)
	require.NotNil(t, p.Variants[1].OfferPrice)
	assert.Equal(t, 14.0, *p.Variants[1].OfferPrice) // 30% off 20
}

func TestProductService_FindByID_NoOffer(t *testing.T) {
	id := bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{productWithVariants(id, 10)}}
	offers := &fakeOfferLookup{discounts: map[string]product.Discount{}} // no live offer
	svc := product.NewProductService(repo, product.WithOffers(offers))

	p, err := svc.FindByID(context.Background(), id.Hex())
	require.NoError(t, err)
	assert.Nil(t, p.Offer)
	assert.Nil(t, p.Variants[0].OfferPrice)
}

func TestProductService_FindByID_FixedDiscountClampsToZero(t *testing.T) {
	id := bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{productWithVariants(id, 5)}}
	offers := &fakeOfferLookup{discounts: map[string]product.Discount{
		id.Hex(): {Type: "fixed", Value: 8}, // more than the price
	}}
	svc := product.NewProductService(repo, product.WithOffers(offers))

	p, err := svc.FindByID(context.Background(), id.Hex())
	require.NoError(t, err)
	require.NotNil(t, p.Variants[0].OfferPrice)
	assert.Equal(t, 0.0, *p.Variants[0].OfferPrice) // clamped, and still serialized (pointer)
	require.NotNil(t, p.Offer)
	assert.Equal(t, 0.0, p.Offer.OfferFromPrice)
}

func TestProductService_List_EnrichesEach(t *testing.T) {
	id1, id2 := bson.NewObjectID(), bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{
		productWithVariants(id1, 10),
		productWithVariants(id2, 50),
	}}
	offers := &fakeOfferLookup{discounts: map[string]product.Discount{
		id1.Hex(): {Type: "percent", Value: 50}, // only the first has an offer
	}}
	svc := product.NewProductService(repo, product.WithOffers(offers))

	ps, _, err := svc.FindAll(context.Background(), product.ListFilter{}, pagination.Params{Page: 1, Limit: 20})
	require.NoError(t, err)
	require.Len(t, ps, 2)

	require.NotNil(t, ps[0].Offer)
	assert.Equal(t, 5.0, *ps[0].Variants[0].OfferPrice)
	assert.Nil(t, ps[1].Offer) // second product untouched
	assert.Nil(t, ps[1].Variants[0].OfferPrice)
}

func TestProductService_FindByID_OfferLookupErrorDegradesGracefully(t *testing.T) {
	id := bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{productWithVariants(id, 10)}}
	offers := &fakeOfferLookup{err: errors.New("offer store down")}
	svc := product.NewProductService(repo, product.WithOffers(offers))

	p, err := svc.FindByID(context.Background(), id.Hex())
	require.NoError(t, err) // lookup failure does not surface to the caller
	assert.Nil(t, p.Offer)  // just no enrichment
	assert.Nil(t, p.Variants[0].OfferPrice)
}

// ---------------------------------------------------------------------------
// Handler tests
// ---------------------------------------------------------------------------

func TestHandler_GetByID_NotFound(t *testing.T) {
	repo := &mockRepo{err: apperrors.ErrNotFound}
	svc := product.NewProductService(repo)
	h := product.NewHandler(svc)

	// Build a request with a chi URL param set.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/nonexistent", nil)

	// Inject chi route context so chi.URLParam works.
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "000000000000000000000000")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.False(t, body["success"].(bool))
	errObj, ok := body["error"].(map[string]interface{})
	require.True(t, ok, "expected error field in response")
	assert.Equal(t, "NOT_FOUND", errObj["code"])
}

// ---------------------------------------------------------------------------
// Bulk action tests
// ---------------------------------------------------------------------------

func TestProductService_Bulk(t *testing.T) {
	tests := []struct {
		name    string
		in      product.BulkInput
		wantN   int64
		wantErr bool
	}{
		{
			name:  "activate two",
			in:    product.BulkInput{IDs: []string{"a", "b"}, Action: product.BulkActivate},
			wantN: 2,
		},
		{
			name:  "delete three",
			in:    product.BulkInput{IDs: []string{"a", "b", "c"}, Action: product.BulkDelete},
			wantN: 3,
		},
		{
			name:    "empty ids is a bad request",
			in:      product.BulkInput{IDs: nil, Action: product.BulkActivate},
			wantErr: true,
		},
		{
			name:    "unknown action is a bad request",
			in:      product.BulkInput{IDs: []string{"a"}, Action: product.BulkAction("explode")},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := product.NewProductService(&mockRepo{})
			n, err := svc.Bulk(context.Background(), tc.in)
			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, apperrors.ErrBadRequest)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantN, n)
		})
	}
}
