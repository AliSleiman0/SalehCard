package product

import (
	"context"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// Service defines the business-logic contract for the product domain.
type Service interface {
	FindAll(ctx context.Context, f ListFilter, p pagination.Params) ([]Product, int64, error)
	FindByID(ctx context.Context, id string) (*Product, error)
	Create(ctx context.Context, in CreateProductInput) (*Product, error)
	Update(ctx context.Context, id string, in UpdateProductInput) (*Product, error)
	Delete(ctx context.Context, id string) error
	Bulk(ctx context.Context, in BulkInput) (int64, error)
}

// ProductService is the concrete implementation of Service.
type ProductService struct {
	repo Repository
}

// NewProductService constructs a ProductService backed by the given repository.
func NewProductService(repo Repository) *ProductService {
	return &ProductService{repo: repo}
}

// FindAll returns a paginated slice of products filtered by f.
func (s *ProductService) FindAll(ctx context.Context, f ListFilter, p pagination.Params) ([]Product, int64, error) {
	// TODO: enforce visibility rules (e.g. hide unavailable products for non-admin callers).
	return s.repo.FindAll(ctx, f, p)
}

// FindByID retrieves a single product by its ID string.
func (s *ProductService) FindByID(ctx context.Context, id string) (*Product, error) {
	return s.repo.FindByID(ctx, id)
}

// Create persists a new product built from in.
func (s *ProductService) Create(ctx context.Context, in CreateProductInput) (*Product, error) {
	// TODO: validate that at least one variant is supplied.
	// TODO: validate that image URLs are reachable (async background job).
	return s.repo.Create(ctx, in)
}

// Update applies a partial update to the product identified by id.
func (s *ProductService) Update(ctx context.Context, id string, in UpdateProductInput) (*Product, error) {
	// TODO: emit a product-updated event so dependent read-models can refresh.
	return s.repo.Update(ctx, id, in)
}

// Delete removes the product identified by id.
func (s *ProductService) Delete(ctx context.Context, id string) error {
	// TODO: check whether any open orders reference this product before deleting.
	return s.repo.Delete(ctx, id)
}

// Bulk applies a bulk action (activate / deactivate / delete) to the given IDs,
// returning the number of affected products.
func (s *ProductService) Bulk(ctx context.Context, in BulkInput) (int64, error) {
	if len(in.IDs) == 0 {
		return 0, apperrors.ErrBadRequest
	}
	switch in.Action {
	case BulkActivate:
		return s.repo.BulkSetAvailable(ctx, in.IDs, true)
	case BulkDeactivate:
		return s.repo.BulkSetAvailable(ctx, in.IDs, false)
	case BulkDelete:
		return s.repo.BulkDelete(ctx, in.IDs)
	default:
		return 0, apperrors.ErrBadRequest
	}
}
