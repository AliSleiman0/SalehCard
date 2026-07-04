package product

import (
	"context"
	"log/slog"
	"time"

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

// Discount describes a live sale on a product. It is a product-local view of an
// offer, kept free of any offer-package type so the product module never imports
// offer (offer imports product — the reverse would cycle). Type is "percent" | "fixed".
type Discount struct {
	Type   string
	Value  float64
	EndsAt *time.Time
}

// OfferLookup is the narrow port the catalog uses to enrich products with live
// sale prices. The bulk map return (keyed by product-id hex; absent = no live
// offer) lets the list endpoint enrich a whole page with one query. The concrete
// adapter that wraps the offer module is wired in the server package.
type OfferLookup interface {
	LiveDiscountsFor(ctx context.Context, ids []string, now time.Time) (map[string]Discount, error)
}

// ProductService is the concrete implementation of Service.
type ProductService struct {
	repo   Repository
	offers OfferLookup // optional; nil disables offer enrichment
}

// ServiceOption configures a ProductService.
type ServiceOption func(*ProductService)

// WithOffers enables live-offer price enrichment on catalog reads.
func WithOffers(o OfferLookup) ServiceOption {
	return func(s *ProductService) { s.offers = o }
}

// NewProductService constructs a ProductService backed by the given repository.
func NewProductService(repo Repository, opts ...ServiceOption) *ProductService {
	s := &ProductService{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// FindAll returns a paginated slice of products filtered by f.
func (s *ProductService) FindAll(ctx context.Context, f ListFilter, p pagination.Params) ([]Product, int64, error) {
	// TODO: enforce visibility rules (e.g. hide unavailable products for non-admin callers).
	products, total, err := s.repo.FindAll(ctx, f, p)
	if err != nil {
		return nil, 0, err
	}
	ptrs := make([]*Product, len(products))
	for i := range products {
		ptrs[i] = &products[i]
	}
	s.enrich(ctx, ptrs)
	return products, total, nil
}

// FindByID retrieves a single product by its ID string.
func (s *ProductService) FindByID(ctx context.Context, id string) (*Product, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enrich(ctx, []*Product{p})
	return p, nil
}

// enrich overlays live-offer sale prices onto the given products in place: a
// product-level Offer block plus a per-variant OfferPrice. It is best-effort —
// a lookup failure logs and leaves the products at their base price (mirrors how
// the order engine tolerates an offer-lookup miss). Order pricing is unaffected;
// these are transient response-only fields.
func (s *ProductService) enrich(ctx context.Context, ps []*Product) {
	if s.offers == nil || len(ps) == 0 {
		return
	}
	ids := make([]string, len(ps))
	for i, p := range ps {
		ids[i] = p.ID.Hex()
	}
	disc, err := s.offers.LiveDiscountsFor(ctx, ids, time.Now().UTC())
	if err != nil {
		slog.Warn("product: offer enrichment failed", "error", err)
		return
	}
	for _, p := range ps {
		d, ok := disc[p.ID.Hex()]
		if !ok {
			continue
		}
		from := fromPrice(p)
		p.Offer = &OfferInfo{
			DiscountType:      d.Type,
			DiscountValue:     d.Value,
			OriginalFromPrice: from,
			OfferFromPrice:    applyDiscount(d, from),
			EndsAt:            d.EndsAt,
		}
		for i := range p.Variants {
			op := applyDiscount(d, p.Variants[i].Price)
			p.Variants[i].OfferPrice = &op
		}
	}
}

// applyDiscount returns the discounted unit price, clamped to [0, original].
// It mirrors offer.OfferPriceFor but stays local so product need not import
// offer; offer.OfferPriceFor remains the single source of truth for order pricing.
func applyDiscount(d Discount, original float64) float64 {
	if original <= 0 {
		return original
	}
	var off float64
	switch d.Type {
	case "percent":
		off = original * d.Value / 100
	case "fixed":
		off = d.Value
	default:
		return original
	}
	if off < 0 {
		off = 0
	}
	if off > original {
		off = original
	}
	return original - off
}

// fromPrice returns the lowest variant price (0 when there are no variants).
func fromPrice(p *Product) float64 {
	from := 0.0
	for i, v := range p.Variants {
		if i == 0 || v.Price < from {
			from = v.Price
		}
	}
	return from
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
