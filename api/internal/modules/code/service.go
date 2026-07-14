package code

import (
	"context"
	"errors"
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
	// Normalize every submitted code/pin (strip ALL whitespace) before the dedup
	// check + insert, so "AB CD-12 34" is stored as "ABCD-1234". A value that is
	// empty once whitespace is removed counts as invalid and is dropped.
	invalid := 0
	normalized := make([]UploadItem, 0, len(in.Codes))
	for _, it := range in.Codes {
		code := normalizeCode(it.Code)
		if code == "" {
			invalid++
			continue
		}
		normalized = append(normalized, UploadItem{Code: code, Pin: normalizeCode(it.Pin)})
	}

	batch := in.Batch
	if batch == "" {
		batch = "upload-" + time.Now().UTC().Format("2006-01-02")
	}

	inserted, duplicates, err := s.repo.BulkInsert(ctx, productID, normalized, batch)
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
		// Total stock value sums only priced products (TotalValue != nil); unpriced
		// ones are counted separately, never treated as $0.
		if st.TotalValue != nil {
			totals.TotalValue += *st.TotalValue
		} else {
			totals.UnvaluedProducts++
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

// errCodeEmpty is returned when an add/edit supplies a blank code value.
var errCodeEmpty = &apperrors.AppError{
	Code:    "CODE_EMPTY",
	Message: "code value is required",
	Err:     apperrors.ErrBadRequest,
}

// errCodeNotAvailable is returned when an edit targets a code that exists but is
// no longer available (delivered or expired).
var errCodeNotAvailable = &apperrors.AppError{
	Code:    "CODE_NOT_AVAILABLE",
	Message: "only available (undelivered) codes can be edited",
	Err:     apperrors.ErrConflict,
}

// errCodeDelivered is returned when a delete targets a delivered code, which is
// tied to an order's fulfillment and must be expired rather than removed.
var errCodeDelivered = &apperrors.AppError{
	Code:    "CODE_DELIVERED",
	Message: "delivered codes cannot be deleted; expire them instead",
	Err:     apperrors.ErrConflict,
}

// AddCode adds a single available code to a product's pool (dedup-guarded by the
// unique index → ErrDuplicateCode on collision) and re-mirrors stock.
func (s *CodeService) AddCode(ctx context.Context, productID, code, pin string) (*Code, error) {
	code = normalizeCode(code)
	pin = normalizeCode(pin)
	if code == "" {
		return nil, errCodeEmpty
	}
	batch := "manual-" + time.Now().UTC().Format("2006-01-02")
	c, err := s.repo.InsertOne(ctx, productID, UploadItem{Code: code, Pin: pin}, batch)
	if err != nil {
		return nil, err
	}
	s.remirror(ctx, productID)
	return c, nil
}

// EditCode edits an available code's value/pin. A collision returns
// ErrDuplicateCode; a code that exists but isn't available returns
// CODE_NOT_AVAILABLE; a missing code returns ErrNotFound.
func (s *CodeService) EditCode(ctx context.Context, productID, codeID, code, pin string) (*Code, error) {
	code = normalizeCode(code)
	pin = normalizeCode(pin)
	if code == "" {
		return nil, errCodeEmpty
	}
	c, err := s.repo.UpdateAvailableCode(ctx, productID, codeID, code, pin)
	if err != nil {
		// Disambiguate a no-match: a non-available code that exists → conflict.
		if errors.Is(err, apperrors.ErrNotFound) {
			if existing, ferr := s.repo.FindByID(ctx, productID, codeID); ferr == nil && existing.Status != StatusAvailable {
				return nil, errCodeNotAvailable
			}
		}
		return nil, err
	}
	return c, nil
}

// DeleteCode deletes a non-delivered code and re-mirrors stock. A delivered code
// (tied to an order) returns CODE_DELIVERED; a missing code returns ErrNotFound.
func (s *CodeService) DeleteCode(ctx context.Context, productID, codeID string) (*Code, error) {
	c, err := s.repo.DeleteAvailableCode(ctx, productID, codeID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			if existing, ferr := s.repo.FindByID(ctx, productID, codeID); ferr == nil && existing.Status == StatusDelivered {
				return nil, errCodeDelivered
			}
		}
		return nil, err
	}
	s.remirror(ctx, productID)
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
	// Monetary columns: unit cost from the product, total = available × cost. Both
	// stay nil when no cost is set so the UI can show "—" rather than $0.
	var totalValue *float64
	if meta.Cost != nil {
		v := *meta.Cost * float64(available)
		totalValue = &v
	}
	return InventoryStats{
		ProductID:  meta.ID,
		Title:      meta.Title,
		Category:   meta.Category,
		Uploaded:   uploaded,
		Available:  available,
		Delivered:  delivered,
		Expired:    expired,
		Threshold:  threshold,
		Level:      computeLevel(available, threshold),
		UnitPrice:  meta.Cost,
		TotalValue: totalValue,
	}
}

// normalizeCode removes ALL whitespace (leading, trailing, and internal) from a
// submitted code or pin, so "AB CD-12 34" is stored as "ABCD-1234". Only whitespace
// is removed — hyphens and every other character are preserved.
func normalizeCode(s string) string { return strings.Join(strings.Fields(s), "") }

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
