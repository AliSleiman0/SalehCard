package product_test

import (
	"bytes"
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
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
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
	// Records of the last BulkSetCategory call, for assign-category assertions.
	catIDs   []string
	catID    string
	catSlug  string
	catRoot  string
	catCalls int
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

func (m *mockRepo) BulkSetCategory(_ context.Context, ids []string, categoryID, slug, rootDomain string) (int64, error) {
	m.catCalls++
	m.catIDs = ids
	m.catID = categoryID
	m.catSlug = slug
	m.catRoot = rootDomain
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

func TestProductService_Update_RejectsDuplicateFieldKeys(t *testing.T) {
	repo := &mockRepo{products: []product.Product{{Title: product.I18nString{En: "P"}}}}
	svc := product.NewProductService(repo)

	in := product.UpdateProductInput{
		InputFields: []product.InputField{
			{Key: "accountId", Type: product.InputFieldText},
			{Key: "accountId", Type: product.InputFieldText},
		},
	}

	_, err := svc.Update(context.Background(), "6a3f04c4ea6747f81d0baf8a", in)

	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrBadRequest)
}

// A legacy product carrying the corrupt {min:0,max:0} constraint must stay
// editable: the save succeeds and the sanitizer strips the bogus bound instead
// of rejecting the whole update (regression guard for imported catalog data).
func TestProductService_Update_RepairsLegacyZeroBounds(t *testing.T) {
	zero := 0.0
	repo := &mockRepo{products: []product.Product{{Title: product.I18nString{En: "P"}}}}
	svc := product.NewProductService(repo)

	in := product.UpdateProductInput{
		Title: &product.I18nString{En: "Renamed"},
		InputFields: []product.InputField{{
			Key:         "quantity",
			Type:        product.InputFieldQuantity,
			Constraints: &product.InputFieldConstraints{Min: &zero, Max: &zero},
		}},
	}

	_, err := svc.Update(context.Background(), "6a3f04c4ea6747f81d0baf8a", in)

	require.NoError(t, err)
	require.Len(t, repo.lastUpdate.InputFields, 1)
	assert.Nil(t, repo.lastUpdate.InputFields[0].Constraints, "corrupt {0,0} bound should be stripped")
}

// A partial update that doesn't touch inputFields (nil slice) must never be
// re-validated — stored legacy data stays untouched behind the repository
// nil-guard, whatever shape it is in.
func TestProductService_Update_NilInputFieldsSkipsValidation(t *testing.T) {
	repo := &mockRepo{products: []product.Product{{Title: product.I18nString{En: "P"}}}}
	svc := product.NewProductService(repo)

	in := product.UpdateProductInput{Title: &product.I18nString{En: "Renamed"}}

	_, err := svc.Update(context.Background(), "6a3f04c4ea6747f81d0baf8a", in)

	require.NoError(t, err)
	assert.Nil(t, repo.lastUpdate.InputFields)
}

func TestProductService_Create_RejectsInvalidBounds(t *testing.T) {
	lo, hi := 10.0, 2.0
	repo := &mockRepo{}
	svc := product.NewProductService(repo)

	in := product.CreateProductInput{
		Title: product.I18nString{En: "P"},
		InputFields: []product.InputField{{
			Key:         "amount",
			Type:        product.InputFieldAmount,
			Constraints: &product.InputFieldConstraints{Min: &lo, Max: &hi},
		}},
	}

	_, err := svc.Create(context.Background(), in)

	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrBadRequest)
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
// Reseller-enrichment tests
// ---------------------------------------------------------------------------

// fakeResellerPricing implements product.ResellerPricing for enrichment tests.
type fakeResellerPricing struct {
	margin    float64
	custom    map[string]float64
	marginErr error
	customErr error
}

func (f *fakeResellerPricing) MarginForUser(_ context.Context, _ bson.ObjectID) (float64, error) {
	return f.margin, f.marginErr
}

func (f *fakeResellerPricing) PricesForUser(_ context.Context, _ bson.ObjectID) (map[string]float64, error) {
	if f.customErr != nil {
		return nil, f.customErr
	}
	return f.custom, nil
}

// resellerCtx returns a context carrying reseller claims, as auth.Optional
// attaches on the catalog routes.
func resellerCtx() context.Context {
	return auth.ContextWithClaims(context.Background(), &auth.Claims{
		UserID: bson.NewObjectID().Hex(),
		Role:   "reseller",
	})
}

func TestProductService_FindByID_ResellerTierMargin(t *testing.T) {
	id := bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{productWithVariants(id, 100, 50)}}
	svc := product.NewProductService(repo,
		product.WithResellerPricing(&fakeResellerPricing{margin: 12}))

	p, err := svc.FindByID(resellerCtx(), id.Hex())
	require.NoError(t, err)

	assert.Nil(t, p.Offer) // no sale badge for reseller pricing
	require.NotNil(t, p.Variants[0].OfferPrice)
	assert.InDelta(t, 88.0, *p.Variants[0].OfferPrice, 1e-9)
	require.NotNil(t, p.Variants[1].OfferPrice)
	assert.InDelta(t, 44.0, *p.Variants[1].OfferPrice, 1e-9)
}

func TestProductService_FindByID_ResellerCustomPriceWins(t *testing.T) {
	id := bson.NewObjectID()
	prod := productWithVariants(id, 100)
	repo := &mockRepo{products: []product.Product{prod}}
	svc := product.NewProductService(repo,
		product.WithResellerPricing(&fakeResellerPricing{
			margin: 12,
			custom: map[string]float64{prod.Variants[0].ID.Hex(): 75},
		}))

	p, err := svc.FindByID(resellerCtx(), id.Hex())
	require.NoError(t, err)
	require.NotNil(t, p.Variants[0].OfferPrice)
	assert.InDelta(t, 75.0, *p.Variants[0].OfferPrice, 1e-9) // custom beats the 88 margin price
}

func TestProductService_FindByID_ResellerAtRetailNoOverlay(t *testing.T) {
	id := bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{productWithVariants(id, 100)}}
	svc := product.NewProductService(repo,
		product.WithResellerPricing(&fakeResellerPricing{margin: 0})) // no tier → retail

	p, err := svc.FindByID(resellerCtx(), id.Hex())
	require.NoError(t, err)
	assert.Nil(t, p.Variants[0].OfferPrice) // not strictly below retail → no struck price
}

func TestProductService_FindByID_ResellerSuppressesOffers(t *testing.T) {
	id := bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{productWithVariants(id, 100)}}
	offers := &fakeOfferLookup{discounts: map[string]product.Discount{
		id.Hex(): {Type: "percent", Value: 50}, // a live retail sale
	}}
	svc := product.NewProductService(repo,
		product.WithOffers(offers),
		product.WithResellerPricing(&fakeResellerPricing{margin: 12}))

	// Reseller: sees only reseller pricing — the offer never applies (it doesn't
	// stack at checkout either).
	p, err := svc.FindByID(resellerCtx(), id.Hex())
	require.NoError(t, err)
	assert.Nil(t, p.Offer)
	require.NotNil(t, p.Variants[0].OfferPrice)
	assert.InDelta(t, 88.0, *p.Variants[0].OfferPrice, 1e-9)

	// Anonymous: the offer path is untouched.
	p, err = svc.FindByID(context.Background(), id.Hex())
	require.NoError(t, err)
	require.NotNil(t, p.Offer)
	assert.InDelta(t, 50.0, *p.Variants[0].OfferPrice, 1e-9)
}

func TestProductService_FindByID_CustomerGetsOfferPathNotResellerPath(t *testing.T) {
	id := bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{productWithVariants(id, 100)}}
	svc := product.NewProductService(repo,
		product.WithOffers(&fakeOfferLookup{discounts: map[string]product.Discount{}}),
		product.WithResellerPricing(&fakeResellerPricing{margin: 50}))

	ctx := auth.ContextWithClaims(context.Background(), &auth.Claims{
		UserID: bson.NewObjectID().Hex(),
		Role:   "customer",
	})
	p, err := svc.FindByID(ctx, id.Hex())
	require.NoError(t, err)
	assert.Nil(t, p.Offer)
	assert.Nil(t, p.Variants[0].OfferPrice) // reseller margin never leaks to customers
}

func TestProductService_FindByID_ResellerLookupErrorDegrades(t *testing.T) {
	id := bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{productWithVariants(id, 100)}}
	svc := product.NewProductService(repo,
		product.WithResellerPricing(&fakeResellerPricing{
			marginErr: errors.New("tier store down"),
			customErr: errors.New("price store down"),
		}))

	p, err := svc.FindByID(resellerCtx(), id.Hex())
	require.NoError(t, err) // degraded, never failed
	assert.Nil(t, p.Variants[0].OfferPrice)
}

func TestProductService_FindByID_ResellerNilPortNoop(t *testing.T) {
	id := bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{productWithVariants(id, 100)}}
	svc := product.NewProductService(repo) // no reseller port wired

	p, err := svc.FindByID(resellerCtx(), id.Hex())
	require.NoError(t, err)
	assert.Nil(t, p.Variants[0].OfferPrice)
}

func TestProductService_List_ResellerEnrichesEach(t *testing.T) {
	id1, id2 := bson.NewObjectID(), bson.NewObjectID()
	repo := &mockRepo{products: []product.Product{
		productWithVariants(id1, 10),
		productWithVariants(id2, 50),
	}}
	svc := product.NewProductService(repo,
		product.WithResellerPricing(&fakeResellerPricing{margin: 10}))

	ps, _, err := svc.FindAll(resellerCtx(), product.ListFilter{}, pagination.Params{Page: 1, Limit: 20})
	require.NoError(t, err)
	require.Len(t, ps, 2)
	require.NotNil(t, ps[0].Variants[0].OfferPrice)
	assert.InDelta(t, 9.0, *ps[0].Variants[0].OfferPrice, 1e-9)
	require.NotNil(t, ps[1].Variants[0].OfferPrice)
	assert.InDelta(t, 45.0, *ps[1].Variants[0].OfferPrice, 1e-9)
	assert.Nil(t, ps[0].Offer)
	assert.Nil(t, ps[1].Offer)
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

// A bridge (mobile-recharge) product with transfer_credit but a variant missing
// its face value is a validation (bad-request) error, not a server fault. The
// Create/Update handlers must surface it as HTTP 400 with the real message — a
// regression guard against the old behavior of dumping it into a 500.

func TestHandler_Update_BridgeValidation_Returns400(t *testing.T) {
	repo := &mockRepo{products: []product.Product{{Title: product.I18nString{En: "1$ ALFA"}}}}
	h := product.NewHandler(product.NewProductService(repo))

	body, _ := json.Marshal(map[string]any{
		"fulfillmentMode": "bridge_device",
		"bridge":          map[string]string{"provider": "alfa", "method": "transfer_credit"},
		"variants":        []map[string]any{{"denomination": "Default", "price": 1.235}}, // no faceValue
	})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/products/x", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "6a3f0455ea6747f81d0baeb6")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.False(t, resp["success"].(bool))
	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Contains(t, errObj["message"], "face value")
}

func TestHandler_Create_BridgeValidation_Returns400(t *testing.T) {
	h := product.NewHandler(product.NewProductService(&mockRepo{}))

	body, _ := json.Marshal(map[string]any{
		"title":           map[string]any{"en": "1$ ALFA"},
		"fulfillmentType": "account_credit",
		"fulfillmentMode": "bridge_device",
		"bridge":          map[string]string{"provider": "alfa", "method": "transfer_credit"},
		"variants":        []map[string]any{{"denomination": "Default", "price": 1.0}}, // no faceValue
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/products", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
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

// fakeCategoryResolver satisfies product.CategoryResolver for assign-category
// tests: Resolve returns a fixed slug/root (or an error), DescendantIDs is unused.
type fakeCategoryResolver struct {
	slug string
	root string
	err  error
}

func (f fakeCategoryResolver) DescendantIDs(context.Context, string) ([]bson.ObjectID, error) {
	return nil, nil
}
func (f fakeCategoryResolver) Resolve(context.Context, string) (string, string, error) {
	return f.slug, f.root, f.err
}

func TestProductService_Bulk_AssignCategory(t *testing.T) {
	const nodeID = "6a3f04c4ea6747f81d0baf8a"

	t.Run("resolves node and sets slug + rootDomain", func(t *testing.T) {
		repo := &mockRepo{}
		svc := product.NewProductService(repo, product.WithCategoryResolver(
			fakeCategoryResolver{slug: "pubg-mobile-id-uc", root: "games"}))

		n, err := svc.Bulk(context.Background(), product.BulkInput{
			IDs: []string{"a", "b"}, Action: product.BulkAssignCategory, CategoryID: nodeID,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(2), n)
		assert.Equal(t, []string{"a", "b"}, repo.catIDs)
		assert.Equal(t, nodeID, repo.catID)
		assert.Equal(t, "pubg-mobile-id-uc", repo.catSlug)
		assert.Equal(t, "games", repo.catRoot)
	})

	t.Run("empty categoryId unassigns without resolving", func(t *testing.T) {
		repo := &mockRepo{}
		// No resolver wired: an unassign must not need one.
		svc := product.NewProductService(repo)

		n, err := svc.Bulk(context.Background(), product.BulkInput{
			IDs: []string{"a"}, Action: product.BulkAssignCategory, CategoryID: "",
		})

		require.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.Equal(t, 1, repo.catCalls)
		assert.Empty(t, repo.catID)
		assert.Empty(t, repo.catSlug)
		assert.Empty(t, repo.catRoot)
	})

	t.Run("assigning without a resolver is a bad request", func(t *testing.T) {
		repo := &mockRepo{}
		svc := product.NewProductService(repo)

		_, err := svc.Bulk(context.Background(), product.BulkInput{
			IDs: []string{"a"}, Action: product.BulkAssignCategory, CategoryID: nodeID,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrBadRequest)
		assert.Zero(t, repo.catCalls)
	})

	t.Run("unresolvable node is a bad request", func(t *testing.T) {
		repo := &mockRepo{}
		svc := product.NewProductService(repo, product.WithCategoryResolver(
			fakeCategoryResolver{err: apperrors.ErrNotFound}))

		_, err := svc.Bulk(context.Background(), product.BulkInput{
			IDs: []string{"a"}, Action: product.BulkAssignCategory, CategoryID: nodeID,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrBadRequest)
		assert.Zero(t, repo.catCalls)
	})
}
