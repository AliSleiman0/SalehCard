package product_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

// ---------------------------------------------------------------------------
// mockRepo implements product.Repository for unit tests.
// ---------------------------------------------------------------------------

type mockRepo struct {
	products []product.Product
	err      error
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

func (m *mockRepo) Update(_ context.Context, _ string, _ product.UpdateProductInput) (*product.Product, error) {
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

func TestProductService_List(t *testing.T) {
	twoProducts := []product.Product{
		{Title: product.I18nString{En: "Product A"}},
		{Title: product.I18nString{En: "Product B"}},
	}

	tests := []struct {
		name          string
		repo          *mockRepo
		wantCount     int
		wantTotal     int64
		wantErr       bool
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
