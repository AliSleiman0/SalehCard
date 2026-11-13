package category_test

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/category"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

func TestNode_BreadcrumbCountsAndVisibility(t *testing.T) {
	root := bson.NewObjectID()   // Games (collection)
	mid := bson.NewObjectID()    // Jawaker (category)
	leaf := bson.NewObjectID()   // Jawaker Tokens (subcategory)
	hidden := bson.NewObjectID() // hidden category
	under := bson.NewObjectID()  // visible, but under the hidden one

	repo := &fakeRepo{cats: []category.Category{
		{ID: root, Slug: "games", Name: category.I18nString{En: "Games"}, RootDomain: "games", Depth: 0, Visible: true},
		{ID: mid, Slug: "jawaker", Name: category.I18nString{En: "Jawaker"}, ParentID: &root, Ancestors: []bson.ObjectID{root}, RootDomain: "games", Depth: 1, Visible: true},
		{ID: leaf, Slug: "jawaker-tokens", ParentID: &mid, Ancestors: []bson.ObjectID{root, mid}, RootDomain: "games", Depth: 2, Visible: true},
		{ID: hidden, Slug: "secret", ParentID: &root, Ancestors: []bson.ObjectID{root}, RootDomain: "games", Depth: 1, Visible: false},
		{ID: under, Slug: "under", ParentID: &hidden, Ancestors: []bson.ObjectID{root, hidden}, RootDomain: "games", Depth: 2, Visible: true},
	}}
	counter := &fakeCounter{
		roots: map[string]int64{"games": 10},
		byCat: map[bson.ObjectID]int64{mid: 2, leaf: 5},
	}
	svc := category.NewCategoryService(repo, counter)
	ctx := context.Background()

	v, err := svc.Node(ctx, mid)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Node.HasChildren {
		t.Error("Jawaker has a child → hasChildren")
	}
	if v.Node.ProductCount == nil || *v.Node.ProductCount != 7 {
		t.Errorf("rolled-up count: want 7 (2 direct + 5 in leaf), got %v", v.Node.ProductCount)
	}
	if len(v.Ancestors) != 1 || v.Ancestors[0].ID != root || v.Ancestors[0].Name.En != "Games" {
		t.Errorf("breadcrumb: want [Games], got %+v", v.Ancestors)
	}

	v, err = svc.Node(ctx, leaf)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Ancestors) != 2 || v.Ancestors[0].ID != root || v.Ancestors[1].ID != mid {
		t.Errorf("leaf breadcrumb must be root→parent, got %+v", v.Ancestors)
	}
	if v.Node.HasChildren {
		t.Error("leaf has no children")
	}

	for name, id := range map[string]bson.ObjectID{"hidden node": hidden, "under hidden ancestor": under, "unknown": bson.NewObjectID()} {
		if _, err := svc.Node(ctx, id); !errors.Is(err, apperrors.ErrNotFound) {
			t.Errorf("%s: want ErrNotFound, got %v", name, err)
		}
	}
}
