package code

import (
	"context"
	"time"

	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// Service is the business-logic contract for the code/inventory domain.
type Service interface {
	Upload(ctx context.Context, productID string, in UploadInput) (*UploadResult, error)
	ListCodes(ctx context.Context, productID string, status Status, p pagination.Params) ([]Code, int64, error)
	Lookup(ctx context.Context, value string) (*CodeAudit, error)
	SetThreshold(ctx context.Context, productID string, threshold int) (*InventoryStats, error)
	Inventory(ctx context.Context) ([]InventoryStats, error)
	LowStock(ctx context.Context) ([]InventoryStats, error)
	// ClaimForOrder claims qty available codes for a product against an order,
	// re-mirroring product stock. On any shortfall it releases what it claimed
	// and returns ErrOutOfStock (the order claims nothing).
	ClaimForOrder(ctx context.Context, productID, orderID, deliveredTo string, qty int) ([]Code, error)
	// ReleaseForOrder returns an order's claimed codes to the pool and
	// re-mirrors stock. Used to compensate a failed order.
	ReleaseForOrder(ctx context.Context, orderID string, productIDs []string) error
	// CountAvailable returns the number of available codes for a product, for a
	// pre-charge stock check.
	CountAvailable(ctx context.Context, productID string) (int, error)
}

// CodeService is the concrete Service implementation.
type CodeService struct {
	repo Repository
}

// NewCodeService constructs a CodeService.
func NewCodeService(repo Repository) *CodeService {
	return &CodeService{repo: repo}
}

// Upload commits a batch of codes, dedupes them, and mirrors the resulting
// available count onto the product's stock field.
func (s *CodeService) Upload(ctx context.Context, productID string, in UploadInput) (*UploadResult, error) {
	invalid := 0
	for _, it := range in.Codes {
		if it.Code == "" {
			invalid++
		}
	}

	batch := in.Batch
	if batch == "" {
		batch = "upload-" + time.Now().UTC().Format("2006-01-02")
	}

	inserted, duplicates, err := s.repo.BulkInsert(ctx, productID, in.Codes, batch)
	if err != nil {
		return nil, err
	}

	// Mirror available count → product.stock (best effort).
	if counts, err := s.repo.CountsByProduct(ctx, productID); err == nil {
		_ = s.repo.SetProductStock(ctx, productID, counts[StatusAvailable])
	}

	return &UploadResult{Inserted: inserted, Duplicates: duplicates, Invalid: invalid}, nil
}

// ListCodes returns a product's codes, paginated and optionally status-filtered.
func (s *CodeService) ListCodes(ctx context.Context, productID string, status Status, p pagination.Params) ([]Code, int64, error) {
	return s.repo.ListByProduct(ctx, productID, status, p)
}

// Lookup finds a code (exact or by suffix) and attaches its product summary.
func (s *CodeService) Lookup(ctx context.Context, value string) (*CodeAudit, error) {
	c, err := s.repo.FindByCodeOrSuffix(ctx, value)
	if err != nil {
		return nil, err
	}
	audit := &CodeAudit{Code: *c}
	if meta, err := s.repo.ProductMeta(ctx, c.ProductID); err == nil {
		audit.Product = &ProductSummary{ID: meta.ID, Title: meta.Title}
	}
	return audit, nil
}

// SetThreshold persists a per-product threshold and returns the refreshed stats.
func (s *CodeService) SetThreshold(ctx context.Context, productID string, threshold int) (*InventoryStats, error) {
	if err := s.repo.SetThreshold(ctx, productID, threshold); err != nil {
		return nil, err
	}
	meta, err := s.repo.ProductMeta(ctx, productID)
	if err != nil {
		meta = &ProductMeta{ID: productID}
	}
	stats, err := s.statsFor(ctx, *meta, threshold)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// Inventory returns code stats for every code-type product.
func (s *CodeService) Inventory(ctx context.Context) ([]InventoryStats, error) {
	products, err := s.repo.CodeProducts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]InventoryStats, 0, len(products))
	for _, p := range products {
		threshold, _ := s.repo.GetThreshold(ctx, p.ID)
		stats, err := s.statsFor(ctx, p, threshold)
		if err != nil {
			return nil, err
		}
		out = append(out, stats)
	}
	return out, nil
}

// LowStock returns the subset of inventory whose available count is below the
// configured threshold (most depleted first).
func (s *CodeService) LowStock(ctx context.Context) ([]InventoryStats, error) {
	all, err := s.Inventory(ctx)
	if err != nil {
		return nil, err
	}
	low := make([]InventoryStats, 0)
	for _, st := range all {
		if st.Available < st.Threshold {
			low = append(low, st)
		}
	}
	// Sort most-depleted first (ratio of available/threshold).
	for i := 1; i < len(low); i++ {
		for j := i; j > 0; j-- {
			a := float64(low[j].Available) / float64(maxInt(low[j].Threshold, 1))
			b := float64(low[j-1].Available) / float64(maxInt(low[j-1].Threshold, 1))
			if a < b {
				low[j], low[j-1] = low[j-1], low[j]
			} else {
				break
			}
		}
	}
	return low, nil
}

// ClaimForOrder claims qty codes for a product atomically, one at a time. If any
// claim fails (e.g. the pool runs dry mid-loop), it releases everything already
// claimed for this order so the order ends up claiming nothing, then returns the
// error. On success it re-mirrors the product's available count onto stock.
func (s *CodeService) ClaimForOrder(ctx context.Context, productID, orderID, deliveredTo string, qty int) ([]Code, error) {
	now := time.Now().UTC()
	claimed := make([]Code, 0, qty)
	for i := 0; i < qty; i++ {
		c, err := s.repo.ClaimOne(ctx, productID, orderID, deliveredTo, now)
		if err != nil {
			_ = s.repo.ReleaseByOrder(ctx, orderID)
			s.remirror(ctx, productID)
			return nil, err
		}
		claimed = append(claimed, *c)
	}
	s.remirror(ctx, productID)
	return claimed, nil
}

// ReleaseForOrder returns an order's claimed codes to the pool and re-mirrors
// stock for each affected product (best effort).
func (s *CodeService) ReleaseForOrder(ctx context.Context, orderID string, productIDs []string) error {
	if err := s.repo.ReleaseByOrder(ctx, orderID); err != nil {
		return err
	}
	for _, pid := range productIDs {
		s.remirror(ctx, pid)
	}
	return nil
}

// CountAvailable returns the number of available codes for a product.
func (s *CodeService) CountAvailable(ctx context.Context, productID string) (int, error) {
	counts, err := s.repo.CountsByProduct(ctx, productID)
	if err != nil {
		return 0, err
	}
	return counts[StatusAvailable], nil
}

// remirror recomputes the product's available-code count onto its stock field
// (best effort; matches the mirror Upload performs).
func (s *CodeService) remirror(ctx context.Context, productID string) {
	if counts, err := s.repo.CountsByProduct(ctx, productID); err == nil {
		_ = s.repo.SetProductStock(ctx, productID, counts[StatusAvailable])
	}
}

// statsFor builds the InventoryStats for one product given its threshold.
func (s *CodeService) statsFor(ctx context.Context, meta ProductMeta, threshold int) (InventoryStats, error) {
	counts, err := s.repo.CountsByProduct(ctx, meta.ID)
	if err != nil {
		return InventoryStats{}, err
	}
	available := counts[StatusAvailable]
	delivered := counts[StatusDelivered]
	expired := counts[StatusExpired]
	uploaded := available + delivered + expired
	if threshold <= 0 {
		threshold = DefaultThreshold
	}
	return InventoryStats{
		ProductID: meta.ID,
		Title:     meta.Title,
		Category:  meta.Category,
		Uploaded:  uploaded,
		Available: available,
		Delivered: delivered,
		Expired:   expired,
		Threshold: threshold,
		Level:     computeLevel(available, threshold),
	}, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
