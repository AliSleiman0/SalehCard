package supplier

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/supplieralert"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

// Mongo-gated: set MONGO_TEST_URI (e.g.
// mongodb://localhost:27017/?directConnection=true). Each test uses a
// throwaway database.

type fakeCat struct {
	*provider.StubProvider
	mu       sync.Mutex
	products []provider.CatalogProduct
	err      error
}

func (f *fakeCat) ListProducts(context.Context) ([]provider.CatalogProduct, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.products, f.err
}

func (f *fakeCat) Profile(context.Context) (provider.Account, error) {
	return provider.Account{Balance: 10, Currency: "USD"}, nil
}

func (f *fakeCat) set(price float64, available bool, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
	f.products = nil
	if err == nil {
		f.products = []provider.CatalogProduct{{UpstreamID: f.up(), Name: "SOUL STAR", Price: price, Currency: "USD", Available: available}}
	}
}

func (f *fakeCat) up() string {
	if f.ID() == 10 {
		return "3258"
	}
	return "1045"
}

func fptr(v float64) *float64 { return &v }

type recAudit struct {
	mu      sync.Mutex
	entries []audit.Entry
}

func (r *recAudit) Record(_ context.Context, e audit.Entry) {
	r.mu.Lock()
	r.entries = append(r.entries, e)
	r.mu.Unlock()
}

type syncFixture struct {
	t        *testing.T
	ctx      context.Context
	db       *mongo.Database
	jentel   *fakeCat
	speed    *fakeCat
	job      *SyncJob
	alerts   *supplieralert.Repo
	rec      *recAudit
	products *product.ProductService
}

func newSyncFixture(t *testing.T, dry bool) *syncFixture {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	db := client.Database("salehcard_test_sync_" + bson.NewObjectID().Hex())
	t.Cleanup(func() { _ = db.Drop(context.Background()) })

	f := &syncFixture{t: t, ctx: ctx, db: db,
		jentel: &fakeCat{StubProvider: provider.NewStub(10)},
		speed:  &fakeCat{StubProvider: provider.NewStub(11)},
		rec:    &recAudit{},
	}
	f.jentel.set(0.0001096, true, nil)
	f.speed.set(0.000110, true, nil)
	probes := provider.NewProbes(provider.NewRegistry(f.jentel, f.speed), provider.ProbesConfig{})
	svc := &Service{
		probes: probes,
		suppliers: []config.SupplierConfig{
			{ID: 10, Name: "jentel", Currency: "USD"},
			{ID: 11, Name: "speedcard", Currency: "USD"},
		},
		settings: newSettingsRepo(db),
		productc: db.Collection("products"),
		orderc:   db.Collection("orders"),
	}
	f.alerts = supplieralert.NewRepo(db)
	if err := f.alerts.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	f.job = NewSyncJob(svc, f.alerts, f.rec, SyncConfig{DryRun: dry})
	f.products = product.NewProductService(product.NewMongoRepository(db))
	return f
}

// insert stores SOUL STAR (per-unit, $1.23 / 10,000) linked to jentel (active)
// and optionally speedcard as a backup.
func (f *syncFixture) insert(backup bool) bson.ObjectID {
	f.t.Helper()
	prov := 10
	p := product.Product{
		ID:          bson.NewObjectID(),
		Title:       product.I18nString{En: "SOUL STAR"},
		Variants:    []product.Variant{{ID: bson.NewObjectID(), Denomination: "per 10,000", Price: 1.23}},
		UnitPricing: &product.UnitPricing{FieldKey: "amount", BlockUnits: 10000},
		InputFields: []product.InputField{{Key: "amount", Label: product.I18nLabel{En: "Enter Amount"}, Type: product.InputFieldAmount,
			Constraints: &product.InputFieldConstraints{Min: fptr(10000), Max: fptr(50000000)}}},
		FulfillmentType:      product.FulfillmentCredit,
		FulfillmentMode:      product.FulfillmentModeAPI,
		FulfillmentProvider:  &prov,
		UpstreamProductID:    "3258",
		Available:            true,
		Images:               []string{},
		SupplierLinks:        []product.SupplierLink{{ID: "la", Provider: 10, UpstreamID: "3258", Enabled: true}},
		ActiveSupplierLinkID: "la",
		SupplierLinksRev:     1,
		CreatedAt:            time.Now().UTC(),
	}
	if backup {
		p.SupplierLinks = append(p.SupplierLinks, product.SupplierLink{ID: "lb", Provider: 11, UpstreamID: "1045", Enabled: true})
	}
	if _, err := f.db.Collection("products").InsertOne(f.ctx, p); err != nil {
		f.t.Fatal(err)
	}
	return p.ID
}

func (f *syncFixture) load(id bson.ObjectID) product.Product {
	f.t.Helper()
	var p product.Product
	if err := f.db.Collection("products").FindOne(f.ctx, bson.D{{Key: "_id", Value: id}}).Decode(&p); err != nil {
		f.t.Fatal(err)
	}
	return p
}

func (f *syncFixture) pass() SyncReport {
	f.t.Helper()
	rep, err := f.job.Pass(f.ctx, 0)
	if err != nil {
		f.t.Fatal(err)
	}
	return rep
}

func (f *syncFixture) openAlerts() map[string]supplieralert.Alert {
	f.t.Helper()
	rows, err := f.alerts.List(f.ctx, true, 0)
	if err != nil {
		f.t.Fatal(err)
	}
	out := map[string]supplieralert.Alert{}
	for _, a := range rows {
		out[a.Kind] = a
	}
	return out
}

func TestSyncPass_HideThenReshowOnlyWhatItHid(t *testing.T) {
	f := newSyncFixture(t, false)
	id := f.insert(false)

	// First pass: snapshot only.
	if rep := f.pass(); rep.Snapshots != 1 || rep.Hidden != 0 {
		t.Fatalf("first pass = %+v", rep)
	}
	p := f.load(id)
	if p.SupplierLinks[0].Sync == nil || p.SupplierLinksRev != 1 {
		t.Fatalf("snapshot not stored or rev bumped: %+v rev=%d", p.SupplierLinks[0].Sync, p.SupplierLinksRev)
	}

	// jentel withdraws it → hidden by sync.
	f.jentel.set(0.0001096, false, nil)
	if rep := f.pass(); rep.Hidden != 1 {
		t.Fatalf("hide pass = %+v", rep)
	}
	p = f.load(id)
	if p.Available || p.HiddenBySyncAt == nil {
		t.Fatalf("available=%v hiddenBySyncAt=%v", p.Available, p.HiddenBySyncAt)
	}
	if _, ok := f.openAlerts()[supplieralert.KindProductHidden]; !ok {
		t.Fatal("no product_hidden alert")
	}

	// Back at jentel → re-shown, hidden alert closed.
	f.jentel.set(0.0001096, true, nil)
	if rep := f.pass(); rep.Shown != 1 {
		t.Fatalf("show pass = %+v", rep)
	}
	p = f.load(id)
	if !p.Available || p.HiddenBySyncAt != nil {
		t.Fatalf("available=%v hiddenBySyncAt=%v", p.Available, p.HiddenBySyncAt)
	}
	open := f.openAlerts()
	if _, ok := open[supplieralert.KindProductHidden]; ok {
		t.Fatal("product_hidden alert still open after re-show")
	}
	if _, ok := open[supplieralert.KindProductShown]; !ok {
		t.Fatal("no product_shown alert")
	}
	if len(f.rec.entries) != 2 {
		t.Fatalf("audit entries = %d, want 2 (hide + show)", len(f.rec.entries))
	}
}

func TestSyncPass_AdminHideIsNeverUndone(t *testing.T) {
	f := newSyncFixture(t, false)
	id := f.insert(false)
	f.jentel.set(0.0001096, false, nil)
	f.pass() // hidden by sync

	// The admin shows it, then hides it again by hand: the marker is gone.
	on, off := true, false
	if _, err := f.products.Update(f.ctx, id.Hex(), product.UpdateProductInput{Available: &on}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.products.Update(f.ctx, id.Hex(), product.UpdateProductInput{Available: &off}); err != nil {
		t.Fatal(err)
	}
	if p := f.load(id); p.HiddenBySyncAt != nil {
		t.Fatal("an admin visibility change must clear hiddenBySyncAt")
	}
	f.jentel.set(0.0001096, true, nil)
	if rep := f.pass(); rep.Shown != 0 {
		t.Fatalf("re-showed an admin-hidden product: %+v", rep)
	}
	if f.load(id).Available {
		t.Fatal("product came back on sale")
	}
}

func TestSyncPass_UnchangedAvailableKeepsMarker(t *testing.T) {
	f := newSyncFixture(t, false)
	id := f.insert(false)
	f.jentel.set(0.0001096, false, nil)
	f.pass()
	// An editor save that re-sends available=false unchanged is not a decision.
	off := false
	title := product.I18nString{En: "SOUL STAR (edited)"}
	if _, err := f.products.Update(f.ctx, id.Hex(), product.UpdateProductInput{Title: &title, Available: &off}); err != nil {
		t.Fatal(err)
	}
	if f.load(id).HiddenBySyncAt == nil {
		t.Fatal("marker cleared by an unchanged available")
	}
}

func TestSyncPass_LossPromotesBackupAndMirrors(t *testing.T) {
	f := newSyncFixture(t, false)
	id := f.insert(true)
	f.jentel.set(0.000130, true, nil) // $1.30 per 10k ≥ $1.23
	rep := f.pass()
	if rep.Paused != 1 || rep.Promoted != 1 || rep.Hidden != 0 {
		t.Fatalf("report = %+v", rep)
	}
	p := f.load(id)
	if p.ActiveSupplierLinkID != "lb" || p.FulfillmentProvider == nil || *p.FulfillmentProvider != 11 || p.UpstreamProductID != "1045" {
		t.Fatalf("active=%q mirror=%v/%q", p.ActiveSupplierLinkID, p.FulfillmentProvider, p.UpstreamProductID)
	}
	if p.SupplierLinksRev != 2 || p.SupplierLinks[0].PausedReason != product.PausedLoss {
		t.Fatalf("rev=%d paused=%q", p.SupplierLinksRev, p.SupplierLinks[0].PausedReason)
	}
	if _, ok := f.openAlerts()[supplieralert.KindLossPaused]; !ok {
		t.Fatal("no loss_paused alert")
	}

	// An admin save of the same table keeps the pause and the snapshots.
	links := product.SupplierLinksInput{Rev: 2, Links: []product.SupplierLinkInput{
		{ID: "la", Provider: 10, UpstreamID: "3258", Enabled: true},
		{ID: "lb", Provider: 11, UpstreamID: "1045", Enabled: true, Active: true},
	}}
	if _, err := f.products.Update(f.ctx, id.Hex(), product.UpdateProductInput{SupplierLinks: &links}); err != nil {
		t.Fatal(err)
	}
	p = f.load(id)
	if p.SupplierLinks[0].PausedReason != product.PausedLoss || p.SupplierLinks[0].Sync == nil || p.SupplierLinks[1].Sync == nil {
		t.Fatalf("admin save wiped sync data: %+v", p.SupplierLinks)
	}

	// Cost recovers → unpaused, active NOT switched back.
	f.jentel.set(0.000100, true, nil)
	if rep := f.pass(); rep.Recovered != 1 {
		t.Fatalf("recovery pass = %+v", rep)
	}
	p = f.load(id)
	if p.SupplierLinks[0].Paused() || p.ActiveSupplierLinkID != "lb" {
		t.Fatalf("paused=%v active=%q", p.SupplierLinks[0].Paused(), p.ActiveSupplierLinkID)
	}
	if _, ok := f.openAlerts()[supplieralert.KindLossPaused]; ok {
		t.Fatal("loss alert still open after recovery")
	}
}

func TestSyncPass_DryRunChangesNothingCustomerFacing(t *testing.T) {
	f := newSyncFixture(t, true)
	id := f.insert(true)
	f.pass() // snapshots
	before := f.load(id)

	f.jentel.set(0.000130, true, nil) // loss → would pause + promote
	f.speed.set(0.000110, false, nil) // backup unavailable → would hide
	rep := f.pass()
	if !rep.DryRun || rep.Paused != 1 || rep.Hidden != 1 {
		t.Fatalf("dry report = %+v", rep)
	}
	p := f.load(id)
	if !p.Available || p.HiddenBySyncAt != nil || p.ActiveSupplierLinkID != before.ActiveSupplierLinkID ||
		p.SupplierLinksRev != before.SupplierLinksRev || p.SupplierLinks[0].Paused() {
		t.Fatalf("dry run changed the product: %+v", p)
	}
	if p.SupplierLinks[0].Sync == nil || p.SupplierLinks[0].Sync.Cost != 0.000130 {
		t.Fatal("dry run still records snapshots")
	}
	for _, a := range f.openAlerts() {
		if !a.DryRun {
			t.Fatalf("alert %s not marked dry run", a.Kind)
		}
		// Plain "would …" wording, no jargon prefix (non-technical admin).
		// A cost change is a fact either way; everything else is a "would".
		if strings.Contains(strings.ToLower(a.Message), "dry run") ||
			(a.Kind != supplieralert.KindCostChanged && !strings.Contains(a.Message, "would")) {
			t.Fatalf("alert %s wording = %q", a.Kind, a.Message)
		}
	}
	if len(f.rec.entries) != 0 {
		t.Fatalf("dry run audited %d changes", len(f.rec.entries))
	}
}

func TestSyncPass_UnreachableSupplierNeverHides(t *testing.T) {
	f := newSyncFixture(t, false)
	id := f.insert(false)
	f.jentel.set(0, false, provider.ErrUnavailable)
	rep := f.pass()
	if rep.Hidden != 0 || !f.load(id).Available {
		t.Fatalf("hid on an unreachable supplier: %+v", rep)
	}
	// The one-supplier button refuses rather than acting on nothing.
	if _, err := f.job.Pass(f.ctx, 10); err == nil {
		t.Fatal("one-supplier pass on an unreachable supplier should fail")
	}
}

func TestSyncPass_StaleRevIsAConflict(t *testing.T) {
	f := newSyncFixture(t, false)
	id := f.insert(true)
	f.jentel.set(0.000130, true, nil)
	p := f.load(id)
	// An admin saved after the pass read the product.
	if _, err := f.db.Collection("products").UpdateByID(f.ctx, id, bson.D{{Key: "$set", Value: bson.D{{Key: "supplierLinksRev", Value: 5}}}}); err != nil {
		t.Fatal(err)
	}
	envv, err := f.job.loadEnv(f.ctx, 0, &SyncReport{})
	if err != nil {
		t.Fatal(err)
	}
	rep := SyncReport{}
	if _, err := f.job.apply(f.ctx, &p, evaluateProduct(&p, envv), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Conflicts != 1 {
		t.Fatalf("conflicts = %d", rep.Conflicts)
	}
	if got := f.load(id); got.ActiveSupplierLinkID != "la" || got.SupplierLinks[0].Paused() {
		t.Fatal("a stale decision was written")
	}
}

func TestSyncPass_TrashedProductsAreSkipped(t *testing.T) {
	f := newSyncFixture(t, false)
	id := f.insert(false)
	now := time.Now().UTC()
	if _, err := f.db.Collection("products").UpdateByID(f.ctx, id, bson.D{{Key: "$set", Value: bson.D{{Key: "deletedAt", Value: now}}}}); err != nil {
		t.Fatal(err)
	}
	f.jentel.set(0.0001096, false, nil)
	if rep := f.pass(); rep.Checked != 0 {
		t.Fatalf("checked a trashed product: %+v", rep)
	}
}

func TestAlerts_DedupeAndAck(t *testing.T) {
	f := newSyncFixture(t, false)
	a := supplieralert.Alert{Kind: supplieralert.KindFxMissing, Key: "fx:20", Message: "m1", Provider: 20}
	if err := f.alerts.Raise(f.ctx, a); err != nil {
		t.Fatal(err)
	}
	a.Message = "m2"
	if err := f.alerts.Raise(f.ctx, a); err != nil {
		t.Fatal(err)
	}
	rows, _ := f.alerts.List(f.ctx, true, 0)
	if len(rows) != 1 || rows[0].Count != 2 || rows[0].Message != "m2" {
		t.Fatalf("rows = %+v", rows)
	}
	if _, err := f.alerts.Ack(f.ctx, rows[0].ID, "admin@x"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.alerts.Ack(f.ctx, rows[0].ID, "admin@x"); err != supplieralert.ErrNotFound {
		t.Fatalf("second ack err = %v", err)
	}
	// Raising again after the ack opens a fresh alert.
	if err := f.alerts.Raise(f.ctx, a); err != nil {
		t.Fatal(err)
	}
	if n, _ := f.alerts.CountOpen(f.ctx); n != 1 {
		t.Fatalf("open = %d", n)
	}
}
