package product_test

import (
	"context"
	"errors"
	"testing"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// fakeRepo implements product.Repository for service tests. Only Create/Update
// carry real bodies (they capture the received input so a test can assert what
// the service forwarded); the remaining methods are zero-value stubs.
type fakeRepo struct {
	createIn  *product.CreateProductInput
	updateIn  *product.UpdateProductInput
	createHit bool
	updateHit bool
}

func (f *fakeRepo) Create(_ context.Context, in product.CreateProductInput) (*product.Product, error) {
	f.createHit = true
	f.createIn = &in
	return &product.Product{}, nil
}
func (f *fakeRepo) Update(_ context.Context, _ string, in product.UpdateProductInput) (*product.Product, error) {
	f.updateHit = true
	f.updateIn = &in
	return &product.Product{}, nil
}
func (f *fakeRepo) FindAll(context.Context, product.ListFilter, pagination.Params) ([]product.Product, int64, error) {
	return nil, 0, nil
}
func (f *fakeRepo) FindByID(context.Context, string) (*product.Product, error) { return nil, nil }
func (f *fakeRepo) FindByLegacyID(context.Context, int) (*product.Product, error) {
	return nil, nil
}
func (f *fakeRepo) Upsert(context.Context, product.UpsertProductInput) (*product.Product, error) {
	return nil, nil
}
func (f *fakeRepo) Delete(context.Context, string) error { return nil }
func (f *fakeRepo) BulkSetAvailable(context.Context, []string, bool) (int64, error) {
	return 0, nil
}
func (f *fakeRepo) BulkDelete(context.Context, []string) (int64, error) { return 0, nil }

func ptr(f float64) *float64 { return &f }

func TestCreate_ForwardsCost(t *testing.T) {
	repo := &fakeRepo{}
	svc := product.NewProductService(repo)

	_, err := svc.Create(context.Background(), product.CreateProductInput{
		FulfillmentType: product.FulfillmentCode,
		Cost:            ptr(4.25),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.createHit || repo.createIn.Cost == nil || *repo.createIn.Cost != 4.25 {
		t.Fatalf("cost not forwarded to repo.Create: %+v", repo.createIn)
	}
}

func TestUpdate_ForwardsCost(t *testing.T) {
	repo := &fakeRepo{}
	svc := product.NewProductService(repo)

	_, err := svc.Update(context.Background(), "abc", product.UpdateProductInput{Cost: ptr(0)})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.updateHit || repo.updateIn.Cost == nil || *repo.updateIn.Cost != 0 {
		t.Fatalf("cost (explicit 0) not forwarded to repo.Update: %+v", repo.updateIn)
	}
}

func TestCreate_NegativeCost_Rejected(t *testing.T) {
	repo := &fakeRepo{}
	svc := product.NewProductService(repo)

	_, err := svc.Create(context.Background(), product.CreateProductInput{
		FulfillmentType: product.FulfillmentCode,
		Cost:            ptr(-1),
	})
	if !errors.Is(err, apperrors.ErrBadRequest) {
		t.Fatalf("want ErrBadRequest, got %v", err)
	}
	if repo.createHit {
		t.Fatal("repo.Create must not be called on a negative cost")
	}
}

func TestUpdate_NegativeCost_Rejected(t *testing.T) {
	repo := &fakeRepo{}
	svc := product.NewProductService(repo)

	_, err := svc.Update(context.Background(), "abc", product.UpdateProductInput{Cost: ptr(-0.5)})
	if !errors.Is(err, apperrors.ErrBadRequest) {
		t.Fatalf("want ErrBadRequest, got %v", err)
	}
	if repo.updateHit {
		t.Fatal("repo.Update must not be called on a negative cost")
	}
}
