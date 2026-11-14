package product

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

func titlesOf(ps []Product) []string {
	got := make([]string, len(ps))
	for i := range ps {
		got[i] = ps[i].Title.En
	}
	return got
}

func TestSortProductsCatalog_PositionFirst(t *testing.T) {
	ps := []Product{
		{Title: I18nString{En: "Alpha"}},
		{Title: I18nString{En: "Zulu"}, Position: 2},
		{Title: I18nString{En: "Bravo"}},
		{Title: I18nString{En: "Yankee"}, Position: 1},
	}
	sortProductsCatalog(ps)
	assert.Equal(t, []string{"Yankee", "Zulu", "Alpha", "Bravo"}, titlesOf(ps))
}

func TestSortProductsByName_IgnoresPosition(t *testing.T) {
	ps := []Product{
		{Title: I18nString{En: "Zulu"}, Position: 1},
		{Title: I18nString{En: "Alpha"}},
	}
	sortProductsByName(ps)
	assert.Equal(t, []string{"Alpha", "Zulu"}, titlesOf(ps))
}

// filterRepo records the filter FindAll receives (only FindAll is used).
type filterRepo struct {
	Repository
	last ListFilter
}

func (r *filterRepo) FindAll(_ context.Context, f ListFilter, _ pagination.Params) ([]Product, int64, error) {
	r.last = f
	return nil, 0, nil
}

type descendantsResolver struct {
	fakeResolver
	ids []bson.ObjectID
}

func (d descendantsResolver) DescendantIDs(context.Context, string) ([]bson.ObjectID, error) {
	return d.ids, nil
}

// Ranks are per node: only a one-node listing may honour them.
func TestFindAll_ManualOrderOnlyForOneNode(t *testing.T) {
	node := bson.NewObjectID()
	leaf := descendantsResolver{ids: []bson.ObjectID{node}}
	parent := descendantsResolver{ids: []bson.ObjectID{node, bson.NewObjectID()}}
	cases := []struct {
		name string
		res  CategoryResolver
		f    ListFilter
		want bool
	}{
		{"leaf node", leaf, ListFilter{CategoryID: node.Hex(), Sort: SortByName}, true},
		{"directOnly on a parent", parent, ListFilter{CategoryID: node.Hex(), CategoryDirect: true, Sort: SortByName}, true},
		{"leaf node + search", leaf, ListFilter{CategoryID: node.Hex(), Search: "x", Sort: SortByName}, true},
		{"parent subtree", parent, ListFilter{CategoryID: node.Hex(), Sort: SortByName}, false},
		{"root domain", parent, ListFilter{RootDomain: "games", Sort: SortByName}, false},
		{"catalog search", parent, ListFilter{Search: "pubg", Sort: SortByName}, false},
		{"leaf, newest sort", leaf, ListFilter{CategoryID: node.Hex()}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &filterRepo{}
			_, _, err := NewProductService(repo, WithCategoryResolver(tc.res)).FindAll(context.Background(), tc.f, pagination.Params{Page: 1, Limit: 20})
			require.NoError(t, err)
			assert.Equal(t, tc.want, repo.last.manualOrder)
		})
	}
}

func TestNameCache_StaleReadNotCached(t *testing.T) {
	invalidateNameCache()
	t.Cleanup(invalidateNameCache)
	now := time.Now()
	gen := nameCacheGeneration()
	invalidateNameCache() // a write lands while the read is in flight
	nameCachePutIfCurrent("k", []Product{{}}, now, gen)
	_, ok := nameCacheGet("k", now)
	assert.False(t, ok, "a set read before the write must not be cached")
	nameCachePutIfCurrent("k", []Product{{}}, now, nameCacheGeneration())
	_, ok = nameCacheGet("k", now)
	assert.True(t, ok)
}

// fakePositionStore is an in-memory positionStore keyed by categoryId.
type fakePositionStore struct {
	products  []Product
	setCalls  int
	setNode   bson.ObjectID
	setIDs    []bson.ObjectID
	leaveOnce bson.ObjectID // simulates a product moving away mid-save
	err       error
}

func (f *fakePositionStore) DirectProducts(_ context.Context, node bson.ObjectID, _ int64) ([]Product, error) {
	var out []Product
	for _, p := range f.products {
		if p.CategoryID != nil && *p.CategoryID == node {
			out = append(out, p)
		}
	}
	return out, f.err
}

func (f *fakePositionStore) DirectProductIDs(ctx context.Context, node bson.ObjectID, limit int64) ([]bson.ObjectID, error) {
	ps, err := f.DirectProducts(ctx, node, limit)
	ids := make([]bson.ObjectID, len(ps))
	for i := range ps {
		ids[i] = ps[i].ID
	}
	return ids, err
}

func (f *fakePositionStore) SetPositions(_ context.Context, node bson.ObjectID, ids []bson.ObjectID) (int64, int64, error) {
	f.setCalls++
	f.setNode, f.setIDs = node, ids
	var matched int64
	for i, id := range ids {
		if id == f.leaveOnce {
			continue
		}
		for j := range f.products {
			if f.products[j].ID == id && f.products[j].CategoryID != nil && *f.products[j].CategoryID == node {
				f.products[j].Position = i + 1
				matched++
			}
		}
	}
	return matched, matched, nil
}

func nodeWith(n int) (bson.ObjectID, *fakePositionStore, []string) {
	node := bson.NewObjectID()
	other := bson.NewObjectID()
	st := &fakePositionStore{}
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		p := Product{ID: bson.NewObjectID(), CategoryID: &node}
		st.products = append(st.products, p)
		ids[i] = p.ID.Hex()
	}
	// A product in another node must never be part of the guard set.
	st.products = append(st.products, Product{ID: bson.NewObjectID(), CategoryID: &other})
	return node, st, ids
}

func TestReorder_StampsPositionsInOrder(t *testing.T) {
	node, st, ids := nodeWith(3)
	o := &orderService{store: st}
	reversed := []string{ids[2], ids[0], ids[1]}

	n, err := o.Reorder(context.Background(), ReorderInput{CategoryID: node.Hex(), ProductIDs: reversed})
	require.NoError(t, err)
	assert.EqualValues(t, 3, n)
	assert.Equal(t, node, st.setNode)
	pos := map[string]int{}
	for _, p := range st.products {
		pos[p.ID.Hex()] = p.Position
	}
	assert.Equal(t, 1, pos[ids[2]])
	assert.Equal(t, 2, pos[ids[0]])
	assert.Equal(t, 3, pos[ids[1]])
}

func TestReorder_Guards(t *testing.T) {
	node, st, ids := nodeWith(3)
	foreign := st.products[3].ID.Hex()

	cases := []struct {
		name     string
		in       ReorderInput
		wantCode string
		conflict bool
	}{
		{"empty list", ReorderInput{CategoryID: node.Hex()}, "BAD_REQUEST", false},
		{"bad category id", ReorderInput{CategoryID: "nope", ProductIDs: ids}, "BAD_REQUEST", false},
		{"bad product id", ReorderInput{CategoryID: node.Hex(), ProductIDs: []string{"x", ids[1], ids[2]}}, "BAD_REQUEST", false},
		{"duplicate", ReorderInput{CategoryID: node.Hex(), ProductIDs: []string{ids[0], ids[0], ids[1]}}, "BAD_REQUEST", false},
		{"missing one (stale)", ReorderInput{CategoryID: node.Hex(), ProductIDs: ids[:2]}, "STALE_ORDER", true},
		{"foreign product (stale)", ReorderInput{CategoryID: node.Hex(), ProductIDs: []string{ids[0], ids[1], foreign}}, "STALE_ORDER", true},
		{"extra product (stale)", ReorderInput{CategoryID: node.Hex(), ProductIDs: append(append([]string{}, ids...), foreign)}, "STALE_ORDER", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st.setCalls = 0
			_, err := (&orderService{store: st}).Reorder(context.Background(), tc.in)
			require.Error(t, err)
			var appErr *apperrors.AppError
			require.True(t, errors.As(err, &appErr))
			assert.Equal(t, tc.wantCode, appErr.Code)
			assert.Equal(t, tc.conflict, errors.Is(err, apperrors.ErrConflict))
			assert.Zero(t, st.setCalls, "a rejected order must not write")
		})
	}
}

func TestReorder_TooManyIsBadRequestNotStale(t *testing.T) {
	node, st, ids := nodeWith(maxArrangeable + 1)
	o := &orderService{store: st}
	_, err := o.List(context.Background(), node.Hex())
	assert.ErrorIs(t, err, apperrors.ErrBadRequest, "GET must not silently truncate")
	_, err = o.Reorder(context.Background(), ReorderInput{CategoryID: node.Hex(), ProductIDs: ids[:maxArrangeable]})
	assert.ErrorIs(t, err, apperrors.ErrBadRequest)
	assert.NotErrorIs(t, err, apperrors.ErrConflict)
}

func TestReorder_CategoryLookupFailureIsNotNotFound(t *testing.T) {
	node, st, ids := nodeWith(2)
	boom := errors.New("cosmos timeout")
	o := &orderService{store: st, categories: fakeResolver{err: boom}}
	_, err := o.Reorder(context.Background(), ReorderInput{CategoryID: node.Hex(), ProductIDs: ids})
	assert.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, apperrors.ErrBadRequest)
}

func TestReorder_UnknownCategory(t *testing.T) {
	node, st, ids := nodeWith(2)
	o := &orderService{store: st, categories: fakeResolver{err: apperrors.ErrNotFound}}
	_, err := o.Reorder(context.Background(), ReorderInput{CategoryID: node.Hex(), ProductIDs: ids})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrBadRequest))
	assert.Zero(t, st.setCalls)
}

func TestReorder_ProductLeftMidSaveIsStale(t *testing.T) {
	node, st, ids := nodeWith(3)
	st.leaveOnce = st.products[1].ID
	n, err := (&orderService{store: st}).Reorder(context.Background(), ReorderInput{CategoryID: node.Hex(), ProductIDs: ids})
	assert.ErrorIs(t, err, apperrors.ErrConflict)
	assert.EqualValues(t, 2, n, "the others are still ranked")
}

func TestOrderList_CatalogOrder(t *testing.T) {
	node := bson.NewObjectID()
	st := &fakePositionStore{products: []Product{
		{ID: bson.NewObjectID(), CategoryID: &node, Title: I18nString{En: "Bravo"}},
		{ID: bson.NewObjectID(), CategoryID: &node, Title: I18nString{En: "Zulu"}, Position: 1},
		{ID: bson.NewObjectID(), CategoryID: &node, Title: I18nString{En: "Alpha"}},
	}}
	ps, err := (&orderService{store: st}).List(context.Background(), node.Hex())
	require.NoError(t, err)
	require.Len(t, ps, 3)
	assert.Equal(t, []string{"Zulu", "Alpha", "Bravo"}, []string{ps[0].Title.En, ps[1].Title.En, ps[2].Title.En})
}

type fakeResolver struct{ err error }

func (f fakeResolver) DescendantIDs(context.Context, string) ([]bson.ObjectID, error) {
	return nil, nil
}
func (f fakeResolver) Resolve(context.Context, string) (string, string, error) {
	return "slug", "games", f.err
}
func (f fakeResolver) Names(context.Context, []bson.ObjectID) (map[bson.ObjectID]I18nString, error) {
	return nil, nil
}

type captureRecorder struct{ entries []audit.Entry }

func (c *captureRecorder) Record(_ context.Context, e audit.Entry) { c.entries = append(c.entries, e) }

func TestProductOrderSaveHandler(t *testing.T) {
	node, st, ids := nodeWith(2)
	o := &orderService{store: st}

	put := func(body any) (*httptest.ResponseRecorder, *captureRecorder) {
		rec := &captureRecorder{}
		raw, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		productOrderSaveHandler(o, rec)(w, httptest.NewRequest(http.MethodPut, "/api/admin/products/order", bytes.NewReader(raw)))
		return w, rec
	}

	w, rec := put(ReorderInput{CategoryID: node.Hex(), ProductIDs: []string{ids[1], ids[0]}})
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, rec.entries, 1)
	assert.Equal(t, audit.ActionCategoryProductReorder, rec.entries[0].Action)
	assert.Equal(t, "category", rec.entries[0].TargetType)
	assert.Equal(t, node.Hex(), rec.entries[0].TargetID)
	assert.Equal(t, []string{ids[1], ids[0]}, rec.entries[0].Summary["productIds"])

	w, rec = put(ReorderInput{CategoryID: node.Hex(), ProductIDs: ids[:1]})
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "STALE_ORDER")
	assert.Empty(t, rec.entries, "nothing written → nothing audited")

	w, _ = put(ReorderInput{CategoryID: node.Hex(), ProductIDs: []string{ids[0], ids[0]}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
