package category_test

import (
	"context"
	"testing"

	"github.com/AliSleiman0/salehcard/api/internal/modules/category"
)

// fakeRepo implements category.Repository for service tests.
type fakeRepo struct {
	cats []category.Category
	err  error
	last category.CategoryFilter
}

func (f *fakeRepo) FindAll(_ context.Context, flt category.CategoryFilter) ([]category.Category, error) {
	f.last = flt
	return f.cats, f.err
}
func (f *fakeRepo) Upsert(context.Context, category.UpsertCategoryInput) (*category.Category, error) {
	return nil, nil
}
func (f *fakeRepo) FindByLegacyID(context.Context, int) (*category.Category, error) {
	return nil, nil
}

// fakeCounter implements category.ProductCounter.
type fakeCounter struct {
	counts map[string]int64
	called bool
}

func (f *fakeCounter) CountByRootDomain(context.Context) (map[string]int64, error) {
	f.called = true
	return f.counts, nil
}

func TestList_WithCounts_AnnotatesRootsOnly(t *testing.T) {
	repo := &fakeRepo{cats: []category.Category{
		{LegacyID: 63, RootDomain: "games", Depth: 0},
		{LegacyID: 7, RootDomain: "games", Depth: 1},
		{LegacyID: 99, RootDomain: "telecom", Depth: 0}, // domain with no products
	}}
	counter := &fakeCounter{counts: map[string]int64{"games": 178}}
	svc := category.NewCategoryService(repo, counter)

	items, err := svc.List(context.Background(), category.CategoryFilter{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %d", len(items))
	}
	if items[0].ProductCount == nil || *items[0].ProductCount != 178 {
		t.Fatalf("root games count: %v", items[0].ProductCount)
	}
	if items[1].ProductCount != nil {
		t.Fatalf("non-root (depth 1) should carry no count: %v", items[1].ProductCount)
	}
	if items[2].ProductCount == nil || *items[2].ProductCount != 0 {
		t.Fatalf("root with no products should be 0, got %v", items[2].ProductCount)
	}
}

func TestList_NoCounts_DoesNotCallCounter(t *testing.T) {
	repo := &fakeRepo{cats: []category.Category{{LegacyID: 63, RootDomain: "games", Depth: 0}}}
	counter := &fakeCounter{counts: map[string]int64{"games": 5}}
	svc := category.NewCategoryService(repo, counter)

	items, err := svc.List(context.Background(), category.CategoryFilter{VisibleOnly: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	if items[0].ProductCount != nil {
		t.Fatalf("no counts requested but ProductCount set: %v", items[0].ProductCount)
	}
	if counter.called {
		t.Fatal("counter should not be queried when withCounts=false")
	}
	if !repo.last.VisibleOnly {
		t.Fatal("VisibleOnly filter not passed through to the repository")
	}
}
