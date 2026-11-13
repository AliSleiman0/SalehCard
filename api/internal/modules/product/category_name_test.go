package product_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// namingResolver serves fixed node names and counts Names calls.
type namingResolver struct {
	names map[bson.ObjectID]product.I18nString
	err   error
	calls int
	asked []bson.ObjectID
}

func (r *namingResolver) DescendantIDs(context.Context, string) ([]bson.ObjectID, error) {
	return nil, nil
}
func (r *namingResolver) Resolve(context.Context, string) (string, string, error) {
	return "", "", nil
}
func (r *namingResolver) Names(_ context.Context, ids []bson.ObjectID) (map[bson.ObjectID]product.I18nString, error) {
	r.calls++
	r.asked = ids
	return r.names, r.err
}

func TestProductService_CategoryNameEnrichment(t *testing.T) {
	named := bson.NewObjectID()
	blank := bson.NewObjectID()
	missing := bson.NewObjectID()

	products := func() []product.Product {
		return []product.Product{
			{Category: "cat-447", CategoryID: &named},
			{Category: "cat-447", CategoryID: &named}, // same node → asked once
			{Category: "cat-479", CategoryID: &blank},
			{Category: "cat-500", CategoryID: &missing},
			{Category: "legacy"}, // unassigned
		}
	}

	t.Run("sets the node name, skips blank/missing/unassigned, dedupes ids", func(t *testing.T) {
		res := &namingResolver{names: map[bson.ObjectID]product.I18nString{
			named: {En: "Dahab Transfer", Ar: "تحويل ذهب"},
			blank: {En: "  "},
		}}
		svc := product.NewProductService(&mockRepo{products: products()}, product.WithCategoryResolver(res))

		got, _, err := svc.FindAll(context.Background(), product.ListFilter{}, pagination.Params{})
		require.NoError(t, err)

		assert.Equal(t, 1, res.calls, "one batched lookup per page")
		assert.ElementsMatch(t, []bson.ObjectID{named, blank, missing}, res.asked)
		require.NotNil(t, got[0].CategoryName)
		assert.Equal(t, "Dahab Transfer", got[0].CategoryName.En)
		require.NotNil(t, got[1].CategoryName)
		assert.Nil(t, got[2].CategoryName, "blank node name → no label")
		assert.Nil(t, got[3].CategoryName, "unknown node → no label")
		assert.Nil(t, got[4].CategoryName, "unassigned product → no label")
	})

	t.Run("lookup failure is best-effort (no error, no names)", func(t *testing.T) {
		res := &namingResolver{err: errors.New("boom")}
		svc := product.NewProductService(&mockRepo{products: products()}, product.WithCategoryResolver(res))

		got, _, err := svc.FindAll(context.Background(), product.ListFilter{}, pagination.Params{})
		require.NoError(t, err)
		for _, p := range got {
			assert.Nil(t, p.CategoryName)
		}
	})

	t.Run("FindByID is enriched too", func(t *testing.T) {
		res := &namingResolver{names: map[bson.ObjectID]product.I18nString{named: {En: "Jawaker"}}}
		svc := product.NewProductService(&mockRepo{products: products()}, product.WithCategoryResolver(res))

		p, err := svc.FindByID(context.Background(), "x")
		require.NoError(t, err)
		require.NotNil(t, p.CategoryName)
		assert.Equal(t, "Jawaker", p.CategoryName.En)
	})
}
