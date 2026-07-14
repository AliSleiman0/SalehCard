package product

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
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

// ResellerPricing is the narrow port the catalog uses to price variants for a
// reseller caller — the buyer's tier margin plus their per-variant custom
// prices, preloaded once per request. It is satisfied directly by
// reseller.MongoRepository (same shape as the order module's resellerPricing
// port) and wired in routes.go, keeping product decoupled from the reseller
// module.
type ResellerPricing interface {
	MarginForUser(ctx context.Context, userID bson.ObjectID) (float64, error)
	PricesForUser(ctx context.Context, userID bson.ObjectID) (map[string]float64, error)
}

// CategoryResolver is the narrow port the catalog uses to make a category filter
// tree-aware (a node lists its own + descendants' products) and to denormalize a
// product's flat category/rootDomain from an assigned taxonomy node. Satisfied by
// category.MongoRepository and wired in the server package (product must not
// import category — category imports product for its product-counter port).
type CategoryResolver interface {
	DescendantIDs(ctx context.Context, id string) ([]bson.ObjectID, error)
	Resolve(ctx context.Context, id string) (slug, rootDomain string, err error)
}

// ProductService is the concrete implementation of Service.
type ProductService struct {
	repo       Repository
	offers     OfferLookup      // optional; nil disables offer enrichment
	resellers  ResellerPricing  // optional; nil disables reseller catalog pricing
	categories CategoryResolver // optional; nil disables tree-aware category filtering
}

// ServiceOption configures a ProductService.
type ServiceOption func(*ProductService)

// WithOffers enables live-offer price enrichment on catalog reads.
func WithOffers(o OfferLookup) ServiceOption {
	return func(s *ProductService) { s.offers = o }
}

// WithResellerPricing enables reseller price enrichment on catalog reads for
// callers whose context claims carry role "reseller".
func WithResellerPricing(rp ResellerPricing) ServiceOption {
	return func(s *ProductService) { s.resellers = rp }
}

// WithCategoryResolver enables tree-aware category filtering (a node lists its
// subtree's products) and taxonomy denormalization on create/update.
func WithCategoryResolver(cr CategoryResolver) ServiceOption {
	return func(s *ProductService) { s.categories = cr }
}

// NewProductService constructs a ProductService backed by the given repository.
func NewProductService(repo Repository, opts ...ServiceOption) *ProductService {
	s := &ProductService{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// FindAll returns a paginated slice of products filtered by f. A categoryId
// filter is expanded to the node + all its descendants (tree-aware) when a
// category resolver is wired, so listing a Collection or Category returns every
// product beneath it.
func (s *ProductService) FindAll(ctx context.Context, f ListFilter, p pagination.Params) ([]Product, int64, error) {
	// TODO: enforce visibility rules (e.g. hide unavailable products for non-admin callers).
	if f.CategoryID != "" && s.categories != nil {
		ids, err := s.categories.DescendantIDs(ctx, f.CategoryID)
		if err != nil {
			return nil, 0, err
		}
		if len(ids) == 0 {
			// Unknown/empty node → no products (avoid a fall-through to unfiltered).
			return []Product{}, 0, nil
		}
		f.CategoryIDs = ids
	}
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

// enrich overlays caller-appropriate pricing onto the given products in place.
// A reseller caller (role "reseller" in the context claims, attached by
// auth.Optional on the catalog routes) gets reseller pricing; everyone else
// gets live-offer enrichment. The two never combine — sale offers don't stack
// on reseller pricing, exactly as at checkout (order.PlaceOrder). Order
// pricing is unaffected; these are transient response-only fields.
func (s *ProductService) enrich(ctx context.Context, ps []*Product) {
	if len(ps) == 0 {
		return
	}
	if claims, ok := auth.ClaimsFromContext(ctx); ok && claims.Role == "reseller" {
		s.enrichReseller(ctx, ps)
		return
	}
	s.enrichOffers(ctx, ps)
}

// enrichReseller overlays the exact checkout price for the calling reseller
// onto Variant.OfferPrice (only when strictly below retail, so clients render
// a struck retail price) and leaves Product.Offer nil (no sale badge). It is
// best-effort — a lookup failure logs and degrades to the remaining pricing
// layers, mirroring order.PlaceOrder's tolerance — and shares
// ResellerUnitPrice with checkout so the displayed price always equals the
// charged price.
func (s *ProductService) enrichReseller(ctx context.Context, ps []*Product) {
	if s.resellers == nil {
		return
	}
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return
	}
	var margin float64
	var custom map[string]float64
	if pct, err := s.resellers.MarginForUser(ctx, userID); err != nil {
		slog.Warn("product: reseller margin lookup failed", "user", userID.Hex(), "error", err)
	} else {
		margin = pct
	}
	if m, err := s.resellers.PricesForUser(ctx, userID); err != nil {
		slog.Warn("product: reseller custom-price lookup failed", "user", userID.Hex(), "error", err)
	} else {
		custom = m
	}
	for _, p := range ps {
		for i := range p.Variants {
			v := &p.Variants[i]
			if rp := ResellerUnitPrice(*v, margin, custom); rp < v.Price {
				price := rp
				v.OfferPrice = &price
			}
		}
	}
}

// enrichOffers overlays live-offer sale prices onto the given products in
// place: a product-level Offer block plus a per-variant OfferPrice. It is
// best-effort — a lookup failure logs and leaves the products at their base
// price (mirrors how the order engine tolerates an offer-lookup miss).
func (s *ProductService) enrichOffers(ctx context.Context, ps []*Product) {
	if s.offers == nil {
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
	mode := in.FulfillmentMode
	if mode == "" {
		mode = DeriveMode(in.FulfillmentType)
	}
	if err := validateBridge(mode, in.Bridge, in.Variants); err != nil {
		return nil, err
	}
	if in.Cost != nil && *in.Cost < 0 {
		return nil, badRequest("cost must be non-negative")
	}
	fields, err := sanitizeInputFields(in.InputFields)
	if err != nil {
		return nil, err
	}
	in.InputFields = fields
	// Assigned to a taxonomy node → denormalize the flat category slug + rootDomain
	// so the legacy flat filters keep finding it.
	if in.CategoryID != nil && *in.CategoryID != "" && s.categories != nil {
		slug, root, rerr := s.categories.Resolve(ctx, *in.CategoryID)
		if rerr != nil {
			return nil, badRequest("category not found")
		}
		in.Category = slug
		in.RootDomain = root
	}
	return s.repo.Create(ctx, in)
}

// Update applies a partial update to the product identified by id.
func (s *ProductService) Update(ctx context.Context, id string, in UpdateProductInput) (*Product, error) {
	// TODO: emit a product-updated event so dependent read-models can refresh.
	// Validate bridge config when it (or a switch to bridge mode) is part of this
	// update. A partial update that touches neither the mode nor the spec skips
	// this — the admin editor always sends mode+spec+variants together on save.
	if in.FulfillmentMode != nil && *in.FulfillmentMode == FulfillmentModeBridgeDevice {
		if err := validateBridge(FulfillmentModeBridgeDevice, in.Bridge, in.Variants); err != nil {
			return nil, err
		}
	} else if in.Bridge != nil && in.Bridge.Provider != "" {
		if !in.Bridge.Valid() {
			return nil, badRequest("bridge fulfillment requires a valid operator (touch|alfa) and method (transfer_credit|recharge_line)")
		}
		if in.Bridge.Method == BridgeMethodTransferCredit && in.Variants != nil {
			if err := validateFaceValues(in.Variants); err != nil {
				return nil, err
			}
		}
	}
	if in.Cost != nil && *in.Cost < 0 {
		return nil, badRequest("cost must be non-negative")
	}
	// Sanitize input-field specs only when the update carries them — nil means
	// "leave unchanged" (repository nil-guard), so a partial update never
	// re-validates a legacy product's stored fields. An empty non-nil slice is
	// a legal clear and passes through.
	if in.InputFields != nil {
		fields, err := sanitizeInputFields(in.InputFields)
		if err != nil {
			return nil, err
		}
		in.InputFields = fields
	}
	// Re-assigned to a taxonomy node → denormalize the flat category slug +
	// rootDomain from it (repository persists categoryId + rootDomain together).
	if in.CategoryID != nil && *in.CategoryID != "" && s.categories != nil {
		slug, root, rerr := s.categories.Resolve(ctx, *in.CategoryID)
		if rerr != nil {
			return nil, badRequest("category not found")
		}
		in.Category = &slug
		in.RootDomain = &root
	}
	return s.repo.Update(ctx, id, in)
}

// badRequest builds a 400-class validation error with a machine code.
func badRequest(msg string) error {
	return &apperrors.AppError{Code: "BAD_REQUEST", Message: msg, Err: apperrors.ErrBadRequest}
}

// validateBridge enforces the bridge_device invariants on a create/full update:
// a bridge product must carry a valid BridgeSpec, and a transfer_credit product's
// every variant must have a positive face value (the amount transferred). A
// non-bridge product carrying a stray spec is rejected so the two never drift.
func validateBridge(mode FulfillmentMode, spec *BridgeSpec, variants []Variant) error {
	if mode != FulfillmentModeBridgeDevice {
		if spec != nil && spec.Provider != "" {
			return badRequest("only bridge_device products may carry a bridge spec")
		}
		return nil
	}
	if spec == nil || !spec.Valid() {
		return badRequest("bridge fulfillment requires a valid operator (touch|alfa) and method (transfer_credit|recharge_line)")
	}
	if spec.Method == BridgeMethodTransferCredit {
		return validateFaceValues(variants)
	}
	return nil
}

// validateFaceValues requires every variant to carry a positive face value.
func validateFaceValues(variants []Variant) error {
	for _, v := range variants {
		if v.FaceValue == nil || *v.FaceValue <= 0 {
			return badRequest("each recharge denomination needs a positive face value")
		}
	}
	return nil
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
