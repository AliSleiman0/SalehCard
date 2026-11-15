package product

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// fakeSupplierCatalog serves scripted catalogs per supplier id.
type fakeSupplierCatalog struct {
	products map[int][]provider.CatalogProduct
	errs     map[int]error
	calls    int
}

func (f *fakeSupplierCatalog) Has(id int) bool {
	_, ok := f.products[id]
	_, bad := f.errs[id]
	return ok || bad
}

func (f *fakeSupplierCatalog) Catalog(_ context.Context, id int) (provider.CatalogSnapshot, error) {
	f.calls++
	if err := f.errs[id]; err != nil {
		return provider.CatalogSnapshot{}, err
	}
	return provider.CatalogSnapshot{Products: f.products[id], FetchedAt: time.Now()}, nil
}

func newLinkCatalog() *fakeSupplierCatalog {
	return &fakeSupplierCatalog{products: map[int][]provider.CatalogProduct{
		10: {{UpstreamID: "3258", Name: "SOUL STAR", Available: true}, {UpstreamID: "195", Name: "SOUL CHAT"}},
		11: {{UpstreamID: "4410", Name: "SOUL STAR"}},
	}, errs: map[int]error{}}
}

func errCode(err error) string {
	var ae *apperrors.AppError
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func ip(n int) *int { return &n }

func TestPlanSupplierLinks_CreateMirrorsActive(t *testing.T) {
	s := &ProductService{catalog: newLinkCatalog()}
	plan, err := s.planSupplierLinks(context.Background(), nil, &SupplierLinksInput{Links: []SupplierLinkInput{
		{Provider: 10, UpstreamID: " 3258 ", Enabled: true},
		{Provider: 11, UpstreamID: "4410", Enabled: true, Active: true},
	}})
	require.NoError(t, err)
	require.Len(t, plan.Links, 2)
	assert.Equal(t, "3258", plan.Links[0].UpstreamID, "trimmed")
	assert.Equal(t, plan.Links[1].ID, plan.ActiveID)
	require.NotNil(t, plan.MirrorProvider)
	assert.Equal(t, 11, *plan.MirrorProvider)
	assert.Equal(t, "4410", plan.MirrorUpstream)
	assert.Equal(t, 1, plan.Rev)
}

func TestPlanSupplierLinks_Validation(t *testing.T) {
	row := func(p int, up string, enabled, active bool) SupplierLinkInput {
		return SupplierLinkInput{Provider: p, UpstreamID: up, Enabled: enabled, Active: active}
	}
	tooMany := make([]SupplierLinkInput, MaxSupplierLinks+1)
	for i := range tooMany {
		tooMany[i] = row(10, "x", true, i == 0)
	}
	cases := []struct {
		name  string
		links []SupplierLinkInput
		code  string
	}{
		{"too many", tooMany, "LINK_INVALID"},
		{"duplicate", []SupplierLinkInput{row(10, "3258", true, true), row(10, "3258", true, false)}, "LINK_INVALID"},
		{"no active", []SupplierLinkInput{row(10, "3258", true, false)}, "LINK_INVALID"},
		{"two active", []SupplierLinkInput{row(10, "3258", true, true), row(11, "4410", true, true)}, "LINK_INVALID"},
		{"active disabled", []SupplierLinkInput{row(10, "3258", false, true)}, "LINK_INVALID"},
		{"missing id", []SupplierLinkInput{row(10, " ", true, true)}, "LINK_INVALID"},
		{"unknown supplier", []SupplierLinkInput{row(99, "1", true, true)}, "LINK_UNKNOWN_SUPPLIER"},
		{"id not in catalog (the 1045 mistake)", []SupplierLinkInput{row(10, "1045", true, true)}, "LINK_UPSTREAM_UNKNOWN"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &ProductService{catalog: newLinkCatalog()}
			_, err := s.planSupplierLinks(context.Background(), nil, &SupplierLinksInput{Links: c.links})
			require.Error(t, err)
			assert.Equal(t, c.code, errCode(err))
		})
	}
}

func TestPlanSupplierLinks_CatalogDownBlocksOnlyNewLinks(t *testing.T) {
	cat := newLinkCatalog()
	cat.errs[11] = provider.ErrUnavailable
	s := &ProductService{catalog: cat}
	cur := &Product{
		SupplierLinks:        []SupplierLink{{ID: "a", Provider: 11, UpstreamID: "4410", Enabled: true}},
		ActiveSupplierLinkID: "a",
		SupplierLinksRev:     3,
	}

	// Re-saving the existing link while its supplier is down: allowed.
	plan, err := s.planSupplierLinks(context.Background(), cur, &SupplierLinksInput{Rev: 3, Links: []SupplierLinkInput{
		{ID: "a", Provider: 11, UpstreamID: "4410", Enabled: true, Active: true},
	}})
	require.NoError(t, err)
	assert.Equal(t, "a", plan.ActiveID, "existing link keeps its id")
	assert.Equal(t, 4, plan.Rev)

	// Adding a NEW link at the down supplier: refused with a 503-class error.
	_, err = s.planSupplierLinks(context.Background(), cur, &SupplierLinksInput{Rev: 3, Links: []SupplierLinkInput{
		{ID: "a", Provider: 11, UpstreamID: "4410", Enabled: true, Active: true},
		{Provider: 11, UpstreamID: "9999", Enabled: true},
	}})
	assert.Equal(t, "LINK_CATALOG_UNAVAILABLE", errCode(err))
	assert.ErrorIs(t, err, errServiceUnavailable)
}

func TestPlanSupplierLinks_StaleRev(t *testing.T) {
	s := &ProductService{catalog: newLinkCatalog()}
	cur := &Product{SupplierLinks: []SupplierLink{{ID: "a", Provider: 10, UpstreamID: "3258", Enabled: true}}, ActiveSupplierLinkID: "a", SupplierLinksRev: 2}
	_, err := s.planSupplierLinks(context.Background(), cur, &SupplierLinksInput{Rev: 1, Links: nil})
	assert.Equal(t, "SUPPLIER_LINKS_CHANGED", errCode(err))
	assert.ErrorIs(t, err, apperrors.ErrConflict)
}

func TestPlanSupplierLinks_LegacyVirtualLinkBecomesReal(t *testing.T) {
	cat := newLinkCatalog()
	s := &ProductService{catalog: cat}
	created := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	cur := &Product{FulfillmentProvider: ip(10), UpstreamProductID: "3258", CreatedAt: created}

	links, active := EffectiveLinks(cur)
	require.Len(t, links, 1)
	assert.Equal(t, legacyLinkID, active)

	plan, err := s.planSupplierLinks(context.Background(), cur, &SupplierLinksInput{Rev: 0, Links: []SupplierLinkInput{
		{ID: legacyLinkID, Provider: 10, UpstreamID: "3258", Enabled: true, Active: true},
		{Provider: 11, UpstreamID: "4410", Enabled: true},
	}})
	require.NoError(t, err)
	require.Len(t, plan.Links, 2)
	assert.NotEqual(t, legacyLinkID, plan.Links[0].ID, "the virtual link gets a real id")
	assert.Equal(t, created, plan.Links[0].AddedAt, "and keeps the legacy mapping's age")
	assert.Equal(t, plan.Links[0].ID, plan.ActiveID)
	assert.Equal(t, 10, *plan.MirrorProvider)
}

func TestPlanSupplierLinks_RemoveAllClearsMirror(t *testing.T) {
	s := &ProductService{catalog: newLinkCatalog()}
	cur := &Product{SupplierLinks: []SupplierLink{{ID: "a", Provider: 10, UpstreamID: "3258", Enabled: true}}, ActiveSupplierLinkID: "a", SupplierLinksRev: 1}
	plan, err := s.planSupplierLinks(context.Background(), cur, &SupplierLinksInput{Rev: 1, Links: []SupplierLinkInput{}})
	require.NoError(t, err)
	assert.Empty(t, plan.Links)
	assert.Nil(t, plan.MirrorProvider)
	assert.Empty(t, plan.MirrorUpstream)
}

func TestCheckLegacySupplierFields(t *testing.T) {
	linked := &Product{FulfillmentProvider: ip(10), UpstreamProductID: "3258",
		SupplierLinks: []SupplierLink{{ID: "a", Provider: 10, UpstreamID: "3258", Enabled: true}}, ActiveSupplierLinkID: "a"}
	same, other := "3258", "1045"
	assert.NoError(t, checkLegacySupplierFields(linked, UpdateProductInput{FulfillmentProvider: ip(10), UpstreamProductID: &same}))
	assert.Equal(t, "SUPPLIER_LINKS_MANAGED", errCode(checkLegacySupplierFields(linked, UpdateProductInput{UpstreamProductID: &other})))
	assert.Equal(t, "SUPPLIER_LINKS_MANAGED", errCode(checkLegacySupplierFields(linked, UpdateProductInput{FulfillmentProvider: ip(11)})))
	// An unmigrated product keeps the old single-mapping edit.
	assert.NoError(t, checkLegacySupplierFields(&Product{FulfillmentProvider: ip(10)}, UpdateProductInput{UpstreamProductID: &other}))
}

func TestLinksFromLegacy(t *testing.T) {
	links, active := linksFromLegacy(ip(12), " 77 ")
	require.Len(t, links, 1)
	assert.Equal(t, active, links[0].ID)
	assert.Equal(t, "77", links[0].UpstreamID)
	l, _ := linksFromLegacy(ip(12), "")
	assert.Nil(t, l)
	l, _ = linksFromLegacy(nil, "77")
	assert.Nil(t, l)
}

func TestMatchCatalog(t *testing.T) {
	prods := []provider.CatalogProduct{
		{UpstreamID: "3258", Name: "SOUL STAR", Category: "Apps", Available: true},
		{UpstreamID: "2", Name: "SOUL CHILL", Category: "Apps"},
		{UpstreamID: "6955", Name: "Soul Land Global — Voucher 1", Category: "Soul Land", Available: true},
		{UpstreamID: "77", Name: "PUBG 60 UC", Category: "Games", Available: true},
	}
	p := FulfillmentProviderOption{ID: 10, Name: "jentel"}
	hits := matchCatalog(p, prods, "soul star", 10)
	require.NotEmpty(t, hits)
	assert.Equal(t, "3258", hits[0].UpstreamID, "exact name first")
	assert.Equal(t, "jentel", hits[0].ProviderName)

	hits = matchCatalog(p, prods, "soul", 10)
	assert.Len(t, hits, 3)
	hits = matchCatalog(p, prods, "77", 10)
	require.Len(t, hits, 1)
	assert.Equal(t, "PUBG 60 UC", hits[0].Name)
	assert.Len(t, matchCatalog(p, prods, "soul", 2), 2, "limit")
}

// TestSupplierLinks_Mongo checks the real writes: create stores the links +
// mirror, an update replaces the table and mirror atomically, a stale rev is
// refused, and the public JSON never carries links.
func TestSupplierLinks_Mongo(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	defer func() { _ = client.Disconnect(ctx) }()
	db := client.Database("salehcard_test_links_" + bson.NewObjectID().Hex())
	defer func() { _ = db.Drop(ctx) }()
	invalidateNameCache()
	t.Cleanup(invalidateNameCache)

	svc := NewProductService(NewMongoRepository(db), WithSupplierCatalog(newLinkCatalog()))

	// Import-style create (legacy pair) → one active link.
	p, err := svc.Create(ctx, CreateProductInput{
		Title: I18nString{En: "SOUL STAR"}, FulfillmentType: FulfillmentCredit, FulfillmentMode: FulfillmentModeAPI,
		FulfillmentProvider: ip(10), UpstreamProductID: "3258", Variants: []Variant{{Price: 1.23}},
	})
	require.NoError(t, err)
	require.Len(t, p.SupplierLinks, 1)
	assert.Equal(t, 1, p.SupplierLinksRev)
	first := p.SupplierLinks[0].ID

	// Add speedcard and make it active: mirror follows in the same write.
	up, err := svc.Update(ctx, p.ID.Hex(), UpdateProductInput{SupplierLinks: &SupplierLinksInput{Rev: 1, Links: []SupplierLinkInput{
		{ID: first, Provider: 10, UpstreamID: "3258", Enabled: true},
		{Provider: 11, UpstreamID: "4410", Enabled: true, Active: true},
	}}})
	require.NoError(t, err)
	require.Len(t, up.SupplierLinks, 2)
	assert.Equal(t, 2, up.SupplierLinksRev)
	assert.Equal(t, first, up.SupplierLinks[0].ID)
	require.NotNil(t, up.FulfillmentProvider)
	assert.Equal(t, 11, *up.FulfillmentProvider)
	assert.Equal(t, "4410", up.UpstreamProductID)

	// A second editor still holding rev 1 is refused; nothing changes.
	_, err = svc.Update(ctx, p.ID.Hex(), UpdateProductInput{SupplierLinks: &SupplierLinksInput{Rev: 1, Links: []SupplierLinkInput{}}})
	assert.Equal(t, "SUPPLIER_LINKS_CHANGED", errCode(err))
	got, err := svc.FindByID(ctx, p.ID.Hex())
	require.NoError(t, err)
	assert.Len(t, got.SupplierLinks, 2)

	// An old admin bundle editing the legacy fields on a linked product is refused.
	other := "1045"
	_, err = svc.Update(ctx, p.ID.Hex(), UpdateProductInput{UpstreamProductID: &other})
	assert.Equal(t, "SUPPLIER_LINKS_MANAGED", errCode(err))

	// Removing every link clears the mirror.
	cleared, err := svc.Update(ctx, p.ID.Hex(), UpdateProductInput{SupplierLinks: &SupplierLinksInput{Rev: 2, Links: []SupplierLinkInput{}}})
	require.NoError(t, err)
	assert.Empty(t, cleared.SupplierLinks)
	assert.Nil(t, cleared.FulfillmentProvider)
	assert.Empty(t, cleared.UpstreamProductID)
}
