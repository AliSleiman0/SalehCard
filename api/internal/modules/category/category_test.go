package category_test

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/category"
)

// fakeRepo implements category.Repository for service tests. FindAll returns the
// same cats for every call; it records the FIRST call's filter (the List method
// issues a second FindAll for the whole tree, which must not clobber the assert).
type fakeRepo struct {
	cats  []category.Category
	err   error
	first category.CategoryFilter
	calls int
}

func (f *fakeRepo) FindAll(_ context.Context, flt category.CategoryFilter) ([]category.Category, error) {
	if f.calls == 0 {
		f.first = flt
	}
	f.calls++
	return f.cats, f.err
}
func (f *fakeRepo) FindByID(context.Context, bson.ObjectID) (*category.Category, error) {
	return nil, nil
}
func (f *fakeRepo) Insert(context.Context, *category.Category) error { return nil }
func (f *fakeRepo) UpdateFields(context.Context, bson.ObjectID, bson.D) (*category.Category, error) {
	return nil, nil
}
func (f *fakeRepo) Delete(context.Context, bson.ObjectID) error { return nil }
func (f *fakeRepo) HasChildren(context.Context, bson.ObjectID) (bool, error) {
	return false, nil
}
func (f *fakeRepo) FindSubtree(context.Context, bson.ObjectID) ([]category.Category, error) {
	return nil, nil
}
func (f *fakeRepo) Upsert(context.Context, category.UpsertCategoryInput) (*category.Category, error) {
	return nil, nil
}
func (f *fakeRepo) FindByLegacyID(context.Context, int) (*category.Category, error) {
	return nil, nil
}

// fakeCounter implements category.ProductCounter.
type fakeCounter struct {
	roots  map[string]int64
	byCat  map[bson.ObjectID]int64
	called bool
}

func (f *fakeCounter) CountByRootDomain(context.Context) (map[string]int64, error) {
	f.called = true
	return f.roots, nil
}
func (f *fakeCounter) CountByCategory(context.Context) (map[bson.ObjectID]int64, error) {
	f.called = true
	return f.byCat, nil
}
func (f *fakeCounter) CountAssignedTo(context.Context, bson.ObjectID) (int64, error) {
	return 0, nil
}

func TestList_WithCounts_RollsUpAndFlagsChildren(t *testing.T) {
	gID := bson.NewObjectID() // root: games
	pID := bson.NewObjectID() // child: pubg (under games)
	tID := bson.NewObjectID() // root: telecom (no products)
	repo := &fakeRepo{cats: []category.Category{
		{ID: gID, RootDomain: "games", Depth: 0},
		{ID: pID, ParentID: &gID, Ancestors: []bson.ObjectID{gID}, RootDomain: "games", Depth: 1},
		{ID: tID, RootDomain: "telecom", Depth: 0},
	}}
	counter := &fakeCounter{
		roots: map[string]int64{"games": 178},
		byCat: map[bson.ObjectID]int64{pID: 20},
	}
	svc := category.NewCategoryService(repo, counter)

	items, err := svc.List(context.Background(), category.CategoryFilter{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %d", len(items))
	}
	// Root uses its denormalized rootDomain count and has a child.
	if items[0].ProductCount == nil || *items[0].ProductCount != 178 {
		t.Fatalf("root games count: %v", items[0].ProductCount)
	}
	if !items[0].HasChildren {
		t.Fatal("root games should report HasChildren")
	}
	// Child (depth 1) uses its rolled-up category count and is a leaf.
	if items[1].ProductCount == nil || *items[1].ProductCount != 20 {
		t.Fatalf("child pubg rolled count: %v", items[1].ProductCount)
	}
	if items[1].HasChildren {
		t.Fatal("leaf pubg should not report HasChildren")
	}
	// Root with no products is 0.
	if items[2].ProductCount == nil || *items[2].ProductCount != 0 {
		t.Fatalf("empty root should be 0, got %v", items[2].ProductCount)
	}
}

func TestList_NoCounts_DoesNotCallCounter(t *testing.T) {
	gID := bson.NewObjectID()
	repo := &fakeRepo{cats: []category.Category{{ID: gID, RootDomain: "games", Depth: 0}}}
	counter := &fakeCounter{roots: map[string]int64{"games": 5}}
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
	if !repo.first.VisibleOnly {
		t.Fatal("VisibleOnly filter not passed through to the repository")
	}
}
