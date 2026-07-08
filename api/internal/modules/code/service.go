package code

import (
	"context"
	"sort"
	"strings"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// Service is the business-logic contract for the code/inventory domain.
type Service interface {
	Upload(ctx context.Context, productID string, in UploadInput) (*UploadResult, error)
	ListCodes(ctx context.Context, productID string, status Status, p pagination.Params) ([]Code, int64, error)
	Lookup(ctx context.Context, value string) (*CodeAudit, error)
	SetThreshold(ctx context.Context, productID string, threshold int) (*InventoryStats, error)
	Inventory(ctx context.Context) ([]InventoryStats, error)
	// InventoryPaged returns one page of the inventory listing (sorted by title),
	// the global totals across every code-type product (for the KPI cards), and
	// the filtered total count for pagination. lowOnly restricts the returned rows
	// to low-stock products; search restricts them to a case-insensitive title
	// substring. Neither filter affects the global totals.
	InventoryPaged(ctx context.Context, p pagination.Params, lowOnly bool, search string) (rows []InventoryStats, totals InventoryTotals, total int64, err error)
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

	// Record the upload for the inventory history view (best effort — a history
	// write must not fail the upload itself).
	_ = s.repo.RecordBatch(ctx, UploadBatch{
		ProductID:  productID,
		Batch:      batch,
		Inserted:   inserted,
		Duplicates: duplicates,
		Invalid:    invalid,
		UploadedBy: in.UploadedBy,
		CreatedAt:  time.Now().UTC(),
	})

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

// Inventory returns code stats for every code-type product. It reads in bulk —
// the product list, all per-(product,status) counts, and all thresholds in three
// queries total — then merges in memory, so its cost is independent of the
// product count (previously a 1+2N sequential fan-out that dominated the admin
// dashboard latency).
func (s *CodeService) Inventory(ctx context.Context) ([]InventoryStats, error) {
	products, err := s.repo.CodeProducts(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := s.repo.CountsByAllProducts(ctx)
	if err != nil {
		return nil, err
	}
	thresholds, err := s.repo.AllThresholds(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]InventoryStats, 0, len(products))
	for _, p := range products {
		out = append(out, buildStats(p, counts[p.ID], thresholds[p.ID]))
	}
	return out, nil
}

// InventoryPaged returns one page of the inventory listing. It builds the full
// stats set once (via Inventory), computes the global KPI totals over it, then
// sorts by title (so skip/limit paging is deterministic), optionally filters to
// low-stock rows and/or a title search, and slices the requested page.
func (s *CodeService) InventoryPaged(ctx context.Context, p pagination.Params, lowOnly bool, search string) ([]InventoryStats, InventoryTotals, int64, error) {
	all, err := s.Inventory(ctx)
	if err != nil {
		return nil, InventoryTotals{}, 0, err
	}

	// Global totals across every product — unaffected by the page, low-only, or
	// search view.
	var totals InventoryTotals
	for _, st := range all {
		totals.Uploaded += st.Uploaded
		totals.Available += st.Available
		totals.Delivered += st.Delivered
		if st.Level == LevelLo {
			totals.LowStock++
		}
	}

	// Stable, deterministic order for paging (natural product order is not).
	sort.SliceStable(all, func(i, j int) bool { return all[i].Title < all[j].Title })

	q := strings.ToLower(strings.TrimSpace(search))
	rows := all
	if lowOnly || q != "" {
		rows = make([]InventoryStats, 0, len(all))
		for _, st := range all {
			if lowOnly && st.Level != LevelLo {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(st.Title), q) {
				continue
			}
			rows = append(rows, st)
		}
	}

	total := int64(len(rows))
	skip := int(pagination.Skip(p))
	if skip >= len(rows) {
		return []InventoryStats{}, totals, total, nil
	}
	end := skip + p.Limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[skip:end], totals, total, nil
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

// Expire marks one available code (looked up by its value) as expired and
// re-mirrors the product's stock. Only available codes can be expired — a code
// that is already delivered or expired returns CODE_NOT_AVAILABLE.
func (s *CodeService) Expire(ctx context.Context, code string) (*Code, error) {
	existing, err := s.repo.FindByCodeOrSuffix(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing.Status != StatusAvailable {
		return nil, &apperrors.AppError{
			Code:    "CODE_NOT_AVAILABLE",
			Message: "only available (undelivered) codes can be expired",
			Err:     apperrors.ErrConflict,
		}
	}
	c, err := s.repo.MarkExpired(ctx, existing.ProductID, existing.Code)
	if err != nil {
		return nil, err
	}
	s.remirror(ctx, c.ProductID)
	return c, nil
}

// CodesForOrder returns the delivered codes claimed under an order, used to
// re-deliver (resend) them to the customer.
func (s *CodeService) CodesForOrder(ctx context.Context, orderID string) ([]Code, error) {
	return s.repo.FindByOrder(ctx, orderID)
}

// remirror recomputes the product's available-code count onto its stock field
// (best effort; matches the mirror Upload performs).
func (s *CodeService) remirror(ctx context.Context, productID string) {
	if counts, err := s.repo.CountsByProduct(ctx, productID); err == nil {
		_ = s.repo.SetProductStock(ctx, productID, counts[StatusAvailable])
	}
}

// statsFor builds the InventoryStats for one product, querying its counts. Used
// by the single-product SetThreshold path; the bulk Inventory path uses
// buildStats directly with pre-fetched counts.
func (s *CodeService) statsFor(ctx context.Context, meta ProductMeta, threshold int) (InventoryStats, error) {
	counts, err := s.repo.CountsByProduct(ctx, meta.ID)
	if err != nil {
		return InventoryStats{}, err
	}
	return buildStats(meta, counts, threshold), nil
}

// buildStats assembles the InventoryStats for one product from its status→count
// map and threshold (defaulting a non-positive threshold). Pure — no I/O.
func buildStats(meta ProductMeta, counts map[Status]int, threshold int) InventoryStats {
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
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
