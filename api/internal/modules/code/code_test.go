package code

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

func TestExpire_RejectsNonAvailableCode(t *testing.T) {
	// The default fake lookup returns a delivered code — it must not be expirable.
	svc := NewCodeService(newFakeRepo())
	_, err := svc.Expire(context.Background(), "X")
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "CODE_NOT_AVAILABLE", appErr.Code)
}

func TestExpire_AvailableCodeSucceeds(t *testing.T) {
	f := newFakeRepo()
	f.lookup = &Code{Code: "A", ProductID: "p1", Status: StatusAvailable}
	svc := NewCodeService(f)
	c, err := svc.Expire(context.Background(), "A")
	require.NoError(t, err)
	assert.Equal(t, StatusExpired, c.Status)
}

// fakeRepo is an in-memory Repository for unit tests.
type fakeRepo struct {
	counts     map[string]map[Status]int
	thresholds map[string]int
	products   []ProductMeta
	stock      map[string]int
	batches    []UploadBatch
	lookup     *Code  // FindByCodeOrSuffix result when set
	orderCodes []Code // FindByOrder result
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{counts: map[string]map[Status]int{}, thresholds: map[string]int{}, stock: map[string]int{}}
}

func (f *fakeRepo) BulkInsert(_ context.Context, productID string, items []UploadItem, _ string) (int, int, error) {
	seen := map[string]struct{}{}
	inserted, dupes := 0, 0
	for _, it := range items {
		if it.Code == "" {
			continue
		}
		if _, ok := seen[it.Code]; ok {
			dupes++
			continue
		}
		seen[it.Code] = struct{}{}
		inserted++
	}
	if f.counts[productID] == nil {
		f.counts[productID] = map[Status]int{}
	}
	f.counts[productID][StatusAvailable] += inserted
	return inserted, dupes, nil
}

func (f *fakeRepo) ListByProduct(_ context.Context, _ string, _ Status, _ pagination.Params) ([]Code, int64, error) {
	return nil, 0, nil
}

func (f *fakeRepo) CountsByProduct(_ context.Context, productID string) (map[Status]int, error) {
	return f.counts[productID], nil
}

func (f *fakeRepo) FindByCodeOrSuffix(_ context.Context, _ string) (*Code, error) {
	if f.lookup != nil {
		return f.lookup, nil
	}
	return &Code{Code: "X", ProductID: "p1", Status: StatusDelivered}, nil
}

func (f *fakeRepo) MarkExpired(_ context.Context, productID, code string) (*Code, error) {
	return &Code{ID: bson.NewObjectID(), Code: code, ProductID: productID, Status: StatusExpired}, nil
}

func (f *fakeRepo) FindByOrder(_ context.Context, _ string) ([]Code, error) {
	return f.orderCodes, nil
}

func (f *fakeRepo) CodeProducts(_ context.Context) ([]ProductMeta, error) { return f.products, nil }

func (f *fakeRepo) ProductMeta(_ context.Context, productID string) (*ProductMeta, error) {
	return &ProductMeta{ID: productID, Title: "Test", Category: "Gift Cards"}, nil
}

func (f *fakeRepo) GetThreshold(_ context.Context, productID string) (int, error) {
	if v, ok := f.thresholds[productID]; ok {
		return v, nil
	}
	return DefaultThreshold, nil
}

func (f *fakeRepo) SetThreshold(_ context.Context, productID string, threshold int) error {
	f.thresholds[productID] = threshold
	return nil
}

func (f *fakeRepo) SetProductStock(_ context.Context, productID string, stock int) error {
	f.stock[productID] = stock
	return nil
}

// ClaimOne pops one available code for the product, decrementing the available
// count and incrementing delivered, mirroring the Mongo claim's effect.
func (f *fakeRepo) ClaimOne(_ context.Context, productID, orderID, deliveredTo string, now time.Time) (*Code, error) {
	if f.counts[productID] == nil || f.counts[productID][StatusAvailable] <= 0 {
		return nil, ErrOutOfStock
	}
	f.counts[productID][StatusAvailable]--
	f.counts[productID][StatusDelivered]++
	return &Code{
		ProductID:   productID,
		Code:        "CODE-" + orderID,
		Status:      StatusDelivered,
		OrderID:     orderID,
		DeliveredTo: deliveredTo,
		DeliveredAt: &now,
	}, nil
}

// ReleaseByOrder is a no-op for the fake (tests assert via stock mirror).
func (f *fakeRepo) ReleaseByOrder(_ context.Context, _ string) error { return nil }

func (f *fakeRepo) RecordBatch(_ context.Context, b UploadBatch) error {
	f.batches = append(f.batches, b)
	return nil
}

func (f *fakeRepo) ListUploadHistory(_ context.Context, _ int) ([]UploadBatchView, error) {
	out := make([]UploadBatchView, len(f.batches))
	for i, b := range f.batches {
		out[i] = UploadBatchView{UploadBatch: b}
	}
	return out, nil
}

func TestComputeLevel(t *testing.T) {
	assert.Equal(t, LevelLo, computeLevel(0, 100))
	assert.Equal(t, LevelLo, computeLevel(30, 100)) // < 40% of threshold
	assert.Equal(t, LevelMid, computeLevel(70, 100))
	assert.Equal(t, LevelHi, computeLevel(120, 100))
}

func TestUpload_CountsAndMirrorsStock(t *testing.T) {
	repo := newFakeRepo()
	svc := NewCodeService(repo)

	in := UploadInput{Codes: []UploadItem{
		{Code: "AAA"}, {Code: "BBB"}, {Code: "AAA"}, {Code: ""},
	}}
	res, err := svc.Upload(context.Background(), "p1", in)
	require.NoError(t, err)

	assert.Equal(t, 2, res.Inserted)   // AAA, BBB
	assert.Equal(t, 1, res.Duplicates) // second AAA
	assert.Equal(t, 1, res.Invalid)    // empty code
	assert.Equal(t, 2, repo.stock["p1"], "available count mirrored to product stock")
}

func TestInventoryAndLowStock(t *testing.T) {
	repo := newFakeRepo()
	repo.products = []ProductMeta{{ID: "p1", Title: "Low one"}, {ID: "p2", Title: "Healthy"}}
	repo.counts["p1"] = map[Status]int{StatusAvailable: 5, StatusDelivered: 95}
	repo.counts["p2"] = map[Status]int{StatusAvailable: 200, StatusDelivered: 10}
	repo.thresholds["p1"] = 100
	repo.thresholds["p2"] = 100

	svc := NewCodeService(repo)

	inv, err := svc.Inventory(context.Background())
	require.NoError(t, err)
	require.Len(t, inv, 2)

	low, err := svc.LowStock(context.Background())
	require.NoError(t, err)
	require.Len(t, low, 1)
	assert.Equal(t, "p1", low[0].ProductID)
	assert.Equal(t, LevelLo, low[0].Level)
	assert.Equal(t, 100, low[0].Uploaded) // 5 available + 95 delivered
}

func TestSetThreshold(t *testing.T) {
	repo := newFakeRepo()
	repo.counts["p1"] = map[Status]int{StatusAvailable: 10}
	svc := NewCodeService(repo)

	stats, err := svc.SetThreshold(context.Background(), "p1", 25)
	require.NoError(t, err)
	assert.Equal(t, 25, stats.Threshold)
	assert.Equal(t, 25, repo.thresholds["p1"])
}
