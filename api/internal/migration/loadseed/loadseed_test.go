package loadseed

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/AliSleiman0/salehcard/api/internal/modules/category"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func sp(s string) *string { return &s }

// --- mapper tests ----------------------------------------------------------

func TestMapProduct_RichFields(t *testing.T) {
	prov := 5
	in := SeedProduct{
		LegacyID: 18, LegacyCategoryID: 7, CategorySlug: "pupg-mobile-id-uc",
		Title:        SeedI18n{En: sp("UC 60"), Ar: sp("شدات")},
		Pricing:      SeedPricing{Retail: 1, Cost: 0.945, Margin: 0.055, Currency: "USD", Mode: "currency_value", ModeConfidence: "low"},
		Verification: &SeedVerification{Provider: 5, App: "pubg"},
		Fulfillment:  SeedFulfillment{Type: "account_credit", Mode: "api", Provider: &prov, Confidence: "high", Cancellable: false},
		Status:       "active",
		Flags:        []string{"needs_translation"},
		InputFields:  []SeedInputField{{Key: "player_id", Label: SeedI18nLabel{En: sp("ID")}, Type: "text"}},
	}
	out := MapProduct(in, "games")

	if out.LegacyID != 18 || out.CategorySlug != "pupg-mobile-id-uc" || out.Category != "pupg-mobile-id-uc" {
		t.Fatalf("identity/category: %+v", out)
	}
	if out.RootDomain != "games" {
		t.Fatalf("rootDomain not stamped: %q", out.RootDomain)
	}
	if out.Title.En != "UC 60" || !out.Available || out.Status != "active" {
		t.Fatalf("title/available: %+v", out)
	}
	if out.FulfillmentType != product.FulfillmentCredit || out.FulfillmentMode != product.FulfillmentModeAPI {
		t.Fatalf("fulfillment: %s/%s", out.FulfillmentType, out.FulfillmentMode)
	}
	if out.FulfillmentProvider == nil || *out.FulfillmentProvider != 5 {
		t.Fatalf("provider: %v", out.FulfillmentProvider)
	}
	if out.Pricing == nil || out.Pricing.Cost != 0.945 || out.Verification == nil || out.Verification.App != "pubg" {
		t.Fatalf("pricing/verification: %+v %+v", out.Pricing, out.Verification)
	}
	if len(out.Variants) != 1 || out.Variants[0].Price != 1 || out.Variants[0].Denomination != "Default" {
		t.Fatalf("synth variant: %+v", out.Variants)
	}
	if len(out.InputFields) != 1 || out.InputFields[0].Key != "player_id" {
		t.Fatalf("input fields: %+v", out.InputFields)
	}
}

func TestMapProduct_StatusUnavailable(t *testing.T) {
	out := MapProduct(SeedProduct{LegacyID: 1, Status: "unavailable", Pricing: SeedPricing{Retail: 2}}, "")
	if out.Available {
		t.Fatal("unavailable status should map to Available=false")
	}
}

func TestSynthVariant_VariableAmountLabel(t *testing.T) {
	out := MapProduct(SeedProduct{LegacyID: 1, Pricing: SeedPricing{Retail: 1}, AmountConstraints: &SeedAmountConstraints{Min: 10, Max: 5000}}, "")
	if out.Variants[0].Denomination != "10–5000" {
		t.Fatalf("variable label: %q", out.Variants[0].Denomination)
	}
	if out.AmountConstraints == nil || out.AmountConstraints.Max != 5000 {
		t.Fatalf("amount constraints: %+v", out.AmountConstraints)
	}
}

func TestMapCategory(t *testing.T) {
	parent := 63
	out := MapCategory(SeedCategory{LegacyID: 7, ParentLegacyID: &parent, Slug: "pubg", Name: SeedI18n{En: sp("PUBG")}, RootDomain: "games", Depth: 1, Visible: true})
	if out.LegacyID != 7 || out.ParentLegacyID == nil || *out.ParentLegacyID != 63 || out.Name.En != "PUBG" || out.RootDomain != "games" {
		t.Fatalf("category map: %+v", out)
	}
}

// --- loader tests (in-memory fakes keyed on legacyId) ----------------------

type fakeCatRepo struct{ byID map[int]*category.Category }
type fakeProdRepo struct{ byID map[int]*product.Product }

func newFakeCatRepo() *fakeCatRepo   { return &fakeCatRepo{byID: map[int]*category.Category{}} }
func newFakeProdRepo() *fakeProdRepo { return &fakeProdRepo{byID: map[int]*product.Product{}} }

func (f *fakeCatRepo) FindByLegacyID(_ context.Context, id int) (*category.Category, error) {
	if c, ok := f.byID[id]; ok {
		return c, nil
	}
	return nil, apperrors.ErrNotFound
}
func (f *fakeCatRepo) Upsert(_ context.Context, in category.UpsertCategoryInput) (*category.Category, error) {
	c := f.byID[in.LegacyID]
	if c == nil {
		c = &category.Category{LegacyID: in.LegacyID}
		f.byID[in.LegacyID] = c
	}
	c.Slug, c.Name = in.Slug, category.I18nString(in.Name)
	return c, nil
}

func (f *fakeProdRepo) FindByLegacyID(_ context.Context, id int) (*product.Product, error) {
	if p, ok := f.byID[id]; ok {
		return p, nil
	}
	return nil, apperrors.ErrNotFound
}
func (f *fakeProdRepo) Upsert(_ context.Context, in product.UpsertProductInput) (*product.Product, error) {
	p := f.byID[in.LegacyID]
	if p == nil {
		// Insert: generate the synthetic variant id once (insert-only), mirroring
		// the real repo's $setOnInsert so re-runs don't churn it.
		variants := make([]product.Variant, len(in.Variants))
		for i, v := range in.Variants {
			if v.ID.IsZero() {
				v.ID = bson.NewObjectID()
			}
			variants[i] = v
		}
		p = &product.Product{Variants: variants}
		f.byID[in.LegacyID] = p
	}
	p.Title = in.Title           // update: never touches variants
	p.RootDomain = in.RootDomain // $set field, refreshed each run
	return p, nil
}

func writeSeeds(t *testing.T, dir string, cats []SeedCategory, prods []SeedProduct) {
	t.Helper()
	for name, v := range map[string]any{"categories.json": cats, "products.json": prods} {
		b, _ := json.Marshal(v)
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRun_DryRunCountsNoWrite(t *testing.T) {
	dir := t.TempDir()
	writeSeeds(t, dir,
		[]SeedCategory{{LegacyID: 7, Slug: "pubg"}, {LegacyID: 8, Slug: "ml"}},
		[]SeedProduct{{LegacyID: 18, Pricing: SeedPricing{Retail: 1}}})
	cat, prod := newFakeCatRepo(), newFakeProdRepo()
	l := &Loader{Cats: cat, Prods: prod}

	rep, err := l.Run(context.Background(), dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if rep.CategoriesInserted != 2 || rep.ProductsInserted != 1 {
		t.Fatalf("dry-run counts: %+v", rep)
	}
	if len(cat.byID) != 0 || len(prod.byID) != 0 {
		t.Fatalf("dry-run wrote data: cats=%d prods=%d", len(cat.byID), len(prod.byID))
	}
}

func TestRun_StampsRootDomainFromCategory(t *testing.T) {
	dir := t.TempDir()
	// Product 18 lives under category 7, whose rootDomain is "games".
	writeSeeds(t, dir,
		[]SeedCategory{{LegacyID: 7, Slug: "pubg", RootDomain: "games"}},
		[]SeedProduct{{LegacyID: 18, LegacyCategoryID: 7, Pricing: SeedPricing{Retail: 1}}})
	cat, prod := newFakeCatRepo(), newFakeProdRepo()
	l := &Loader{Cats: cat, Prods: prod}

	if _, err := l.Run(context.Background(), dir, false); err != nil {
		t.Fatal(err)
	}
	if got := prod.byID[18].RootDomain; got != "games" {
		t.Fatalf("product rootDomain resolved from category = %q, want games", got)
	}
}

func TestRun_Idempotent(t *testing.T) {
	dir := t.TempDir()
	writeSeeds(t, dir,
		[]SeedCategory{{LegacyID: 7, Slug: "pubg"}},
		[]SeedProduct{{LegacyID: 18, Pricing: SeedPricing{Retail: 1}}})
	cat, prod := newFakeCatRepo(), newFakeProdRepo()
	l := &Loader{Cats: cat, Prods: prod}

	r1, err := l.Run(context.Background(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if r1.CategoriesInserted != 1 || r1.ProductsInserted != 1 {
		t.Fatalf("run 1: %+v", r1)
	}
	variantID := prod.byID[18].Variants[0].ID

	r2, err := l.Run(context.Background(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if r2.CategoriesInserted != 0 || r2.ProductsInserted != 0 || r2.CategoriesUpdated != 1 || r2.ProductsUpdated != 1 {
		t.Fatalf("run 2 should be all updates: %+v", r2)
	}
	if len(prod.byID) != 1 || len(cat.byID) != 1 {
		t.Fatalf("re-run duplicated: cats=%d prods=%d", len(cat.byID), len(prod.byID))
	}
	if prod.byID[18].Variants[0].ID != variantID {
		t.Fatal("variant id changed across re-run")
	}
}
