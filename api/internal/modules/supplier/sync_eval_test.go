package supplier

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/supplieralert"
	"github.com/AliSleiman0/salehcard/api/internal/platform/provider"
)

var evalNow = time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC)

// soulStar is a per-unit product: $1.23 per 10,000 units → $0.000123/unit.
func soulStar(links ...product.SupplierLink) *product.Product {
	p := &product.Product{
		ID:          bson.NewObjectID(),
		Title:       product.I18nString{En: "SOUL STAR"},
		Variants:    []product.Variant{{ID: bson.NewObjectID(), Price: 1.23}},
		UnitPricing: &product.UnitPricing{FieldKey: "amount", BlockUnits: 10000},
		Available:   true,
	}
	p.SupplierLinks = links
	if len(links) > 0 {
		p.ActiveSupplierLinkID = links[0].ID
	}
	p.SupplierLinksRev = 3
	return p
}

func link(id string, prov int, up string) product.SupplierLink {
	return product.SupplierLink{ID: id, Provider: prov, UpstreamID: up, Enabled: true}
}

// cat builds a known catalog with one product.
func cat(up string, price float64, available bool, currency string) catalogView {
	return catalogView{Known: true, Products: map[string]provider.CatalogProduct{
		up: {UpstreamID: up, Name: "SOUL STAR", Price: price, Currency: currency, Available: available},
	}}
}

func env(cats map[int]catalogView) syncEnv {
	return syncEnv{
		Catalogs: cats,
		FX: func(_ int, c string) (float64, bool) {
			if c == "" || c == "USD" {
				return 1, true
			}
			return 0, false
		},
		Names: map[int]string{10: "jentel", 11: "speedcard", 12: "gift4card"},
		Now:   evalNow,
	}
}

func kinds(res evalResult) []string {
	var out []string
	for _, c := range res.Changes {
		out = append(out, c.Kind)
	}
	return out
}

func alertKinds(res evalResult) []string {
	var out []string
	for _, a := range res.Alerts {
		out = append(out, a.Kind)
	}
	return out
}

func prevSync(cost float64, at time.Time) *product.LinkSync {
	return &product.LinkSync{CheckedAt: at, Found: true, Available: true, Cost: cost, Currency: "USD", Name: "SOUL STAR"}
}

func TestEvaluate_FirstCheckRecordsSnapshotOnly(t *testing.T) {
	p := soulStar(link("a", 10, "3258"))
	res := evaluateProduct(p, env(map[int]catalogView{10: cat("3258", 0.0001096, true, "USD")}))
	if res.Structural || res.Hide || res.Show || len(res.Alerts) != 0 {
		t.Fatalf("unexpected decision: %+v", res)
	}
	if len(res.Snapshots) != 1 || res.Snapshots[0] != "a" {
		t.Fatalf("snapshots = %v, want [a]", res.Snapshots)
	}
	s := res.Links[0].Sync
	if s == nil || !s.Found || s.Cost != 0.0001096 || s.CostUSD == nil || *s.CostUSD != 0.0001096 {
		t.Fatalf("snapshot = %+v", s)
	}
}

func TestEvaluate_SameSnapshotIsNotRewritten(t *testing.T) {
	l := link("a", 10, "3258")
	usd := 0.0001096
	l.Sync = prevSync(0.0001096, evalNow.Add(-10*time.Minute))
	l.Sync.CostUSD = &usd
	res := evaluateProduct(soulStar(l), env(map[int]catalogView{10: cat("3258", 0.0001096, true, "USD")}))
	if len(res.Snapshots) != 0 {
		t.Fatalf("snapshots = %v, want none (nothing changed)", res.Snapshots)
	}
}

func TestEvaluate_CostMoveAlertsPerBlock(t *testing.T) {
	// 0.0001096 → 0.0001150 per unit = $1.096 → $1.150 per 10,000 (+4.9%).
	l := link("a", 10, "3258")
	l.Sync = prevSync(0.0001096, evalNow.Add(-time.Hour))
	res := evaluateProduct(soulStar(l), env(map[int]catalogView{10: cat("3258", 0.000115, true, "USD")}))
	if got := alertKinds(res); len(got) != 1 || got[0] != supplieralert.KindCostChanged {
		t.Fatalf("alerts = %v, want [cost_changed]", got)
	}
	if !strings.Contains(res.Alerts[0].Message, "per 10,000") || !strings.Contains(res.Alerts[0].Message, "$1.10") {
		t.Fatalf("message = %q", res.Alerts[0].Message)
	}
	if res.Links[0].Sync.CostChangedAt == nil || *res.Links[0].Sync.PrevCost != 0.0001096 {
		t.Fatalf("cost change not recorded: %+v", res.Links[0].Sync)
	}
	if res.Structural || res.Hide {
		t.Fatal("a cost move must never change the product")
	}
}

func TestEvaluate_TinyCostMoveIsNoise(t *testing.T) {
	// $1.096 → $1.1005 per 10,000: +0.4%, under both 0.5% and $0.01.
	l := link("a", 10, "3258")
	l.Sync = prevSync(0.0001096, evalNow.Add(-time.Hour))
	res := evaluateProduct(soulStar(l), env(map[int]catalogView{10: cat("3258", 0.00011005, true, "USD")}))
	if len(res.Alerts) != 0 {
		t.Fatalf("alerts = %v, want none", alertKinds(res))
	}
}

func TestEvaluate_LossPausesActiveAndPromotesBackup(t *testing.T) {
	p := soulStar(link("a", 10, "3258"), link("b", 11, "1045"))
	res := evaluateProduct(p, env(map[int]catalogView{
		10: cat("3258", 0.000130, true, "USD"), // $1.30 per 10k ≥ $1.23
		11: cat("1045", 0.000110, true, "USD"), // $1.10 per 10k
	}))
	if !res.Structural || res.ActiveID != "b" {
		t.Fatalf("active = %q structural=%v, want b / true", res.ActiveID, res.Structural)
	}
	if res.Links[0].PausedReason != product.PausedLoss || res.Links[0].PausedAt == nil {
		t.Fatalf("link a not paused: %+v", res.Links[0])
	}
	if res.Hide {
		t.Fatal("a promoted product stays on sale")
	}
	if got := kinds(res); strings.Join(got, ",") != "loss_paused,promoted" {
		t.Fatalf("changes = %v", got)
	}
	if !strings.Contains(res.Alerts[0].Message, "Orders now go to speedcard (#11)") {
		t.Fatalf("loss alert = %q", res.Alerts[0].Message)
	}
}

// Warning-only mode words every alert as what WOULD happen, in plain words:
// the admin is non-technical and nothing was actually changed.
func TestEvaluate_PreviewAlertsSayWould(t *testing.T) {
	e := env(map[int]catalogView{
		10: cat("3258", 0.000130, true, "USD"), // loss
		11: cat("1045", 0.000110, true, "USD"),
	})
	e.Preview = true
	res := evaluateProduct(soulStar(link("a", 10, "3258"), link("b", 11, "1045")), e)
	msg := res.Alerts[0].Message
	if !strings.Contains(msg, "Orders would switch to speedcard (#11)") || !strings.Contains(msg, "Nothing was changed yet") ||
		strings.Contains(msg, "paused.") || strings.Contains(strings.ToLower(msg), "dry run") {
		t.Fatalf("preview loss alert = %q", msg)
	}

	e = env(map[int]catalogView{10: cat("3258", 0.000110, false, "USD")}) // out of stock everywhere
	e.Preview = true
	res = evaluateProduct(soulStar(link("a", 10, "3258")), e)
	if !res.Hide || len(res.Alerts) != 1 {
		t.Fatalf("want one hide alert, got hide=%v alerts=%v", res.Hide, alertKinds(res))
	}
	if msg := res.Alerts[0].Message; msg != "SOUL STAR would be taken off sale: no linked supplier has it available. Nothing was changed — it is still on sale." {
		t.Fatalf("preview hide alert = %q", msg)
	}
}

// Every alert carries its values as Args so the admin console can write it
// in Arabic / Turkish.
func TestEvaluate_AlertsCarryArgs(t *testing.T) {
	res := evaluateProduct(soulStar(link("a", 10, "3258"), link("b", 11, "1045")), env(map[int]catalogView{
		10: cat("3258", 0.000130, true, "USD"), // loss → switch to b
		11: cat("1045", 0.000110, true, "USD"),
	}))
	got := res.Alerts[0].Args
	want := map[string]string{"outcome": "switched", "to": "speedcard (#11)", "toProduct": "1045",
		"product": "SOUL STAR", "supplier": "jentel (#10)"}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("loss args[%q] = %q, want %q (all: %v)", k, got[k], v, got)
		}
	}
	if got["cost"] == "" || got["sell"] == "" || got["per"] == "" {
		t.Fatalf("loss args miss amounts: %v", got)
	}

	res = evaluateProduct(soulStar(link("a", 10, "3258")), env(map[int]catalogView{10: cat("3258", 0.000110, false, "USD")}))
	if a := res.Alerts[0]; a.Args["product"] != "SOUL STAR" || a.Args["reason"] != "unavailable" {
		t.Fatalf("hide args = %v", a.Args)
	}
}

func TestEvaluate_LossWithNoBackupHides(t *testing.T) {
	p := soulStar(link("a", 10, "3258"), link("b", 11, "1045"))
	res := evaluateProduct(p, env(map[int]catalogView{
		10: cat("3258", 0.000130, true, "USD"),
		11: cat("1045", 0.000125, true, "USD"), // also a loss → paused too
	}))
	if res.ActiveID != "a" {
		t.Fatalf("active = %q, want a (nothing to promote)", res.ActiveID)
	}
	if !res.Hide || res.HideReason != "loss" {
		t.Fatalf("hide=%v reason=%q, want loss hide", res.Hide, res.HideReason)
	}
	if !res.Links[1].Paused() {
		t.Fatal("a backup at a loss is paused too (so the order fallback skips it)")
	}
}

func TestEvaluate_PromotionSkipsUnavailableAndUnknown(t *testing.T) {
	p := soulStar(link("a", 10, "3258"), link("b", 11, "1045"), link("c", 12, "226"), link("d", 13, "9"))
	res := evaluateProduct(p, env(map[int]catalogView{
		10: cat("3258", 0.000130, true, "USD"),
		11: cat("1045", 0.000100, false, "USD"), // unavailable
		12: cat("226", 0.000100, true, "USD"),   // ok
		// 13: catalog unknown
	}))
	if res.ActiveID != "c" {
		t.Fatalf("active = %q, want c", res.ActiveID)
	}
}

func TestEvaluate_RecoveryUnpausesWithoutSwitchingBack(t *testing.T) {
	a := link("a", 10, "3258")
	at := evalNow.Add(-time.Hour)
	a.PausedReason, a.PausedAt = product.PausedLoss, &at
	b := link("b", 11, "1045")
	p := soulStar(a, b)
	p.ActiveSupplierLinkID = "b"
	res := evaluateProduct(p, env(map[int]catalogView{
		10: cat("3258", 0.000100, true, "USD"),
		11: cat("1045", 0.000110, true, "USD"),
	}))
	if res.Links[0].Paused() || !res.Structural {
		t.Fatal("link a should be unpaused (structural write)")
	}
	if res.ActiveID != "b" {
		t.Fatalf("active = %q: recovery must never switch back", res.ActiveID)
	}
	if got := alertKinds(res); len(got) != 1 || got[0] != supplieralert.KindLossRecovered {
		t.Fatalf("alerts = %v", got)
	}
	if len(res.Resolve) != 1 || res.Resolve[0] != "loss:"+p.ID.Hex()+":a" {
		t.Fatalf("resolve = %v", res.Resolve)
	}
}

func TestEvaluate_RecoveredLossHiddenProductComesBack(t *testing.T) {
	a := link("a", 10, "3258")
	at := evalNow.Add(-time.Hour)
	a.PausedReason, a.PausedAt = product.PausedLoss, &at
	p := soulStar(a)
	p.Available, p.HiddenBySyncAt = false, &at
	res := evaluateProduct(p, env(map[int]catalogView{10: cat("3258", 0.000100, true, "USD")}))
	if !res.Show || res.Hide {
		t.Fatalf("show=%v hide=%v, want show", res.Show, res.Hide)
	}
}

func TestEvaluate_UnknownCatalogNeverHides(t *testing.T) {
	l := link("a", 10, "3258")
	l.Sync = prevSync(0.0001, evalNow.Add(-time.Hour))
	p := soulStar(l)
	res := evaluateProduct(p, env(map[int]catalogView{})) // jentel didn't load
	if res.Hide || res.Structural || len(res.Snapshots) != 0 || len(res.Alerts) != 0 {
		t.Fatalf("unknown data must decide nothing: %+v", res)
	}
	if res.Links[0].Sync.CheckedAt != l.Sync.CheckedAt {
		t.Fatal("the last snapshot is kept")
	}
}

func TestEvaluate_HideNeedsEveryWatchedLinkKnown(t *testing.T) {
	p := soulStar(link("a", 10, "3258"), link("b", 11, "1045"))
	// jentel says unavailable, speedcard unknown → stay on sale.
	res := evaluateProduct(p, env(map[int]catalogView{10: cat("3258", 0.0001, false, "USD")}))
	if res.Hide {
		t.Fatal("hid on partly unknown data")
	}
	// both known and unavailable → hide.
	res = evaluateProduct(p, env(map[int]catalogView{
		10: cat("3258", 0.0001, false, "USD"),
		11: {Known: true, Products: map[string]provider.CatalogProduct{}}, // withdrawn
	}))
	if !res.Hide || res.HideReason != "unavailable" {
		t.Fatalf("hide=%v reason=%q, want unavailable", res.Hide, res.HideReason)
	}
	if got := alertKinds(res); len(got) != 1 || got[0] != supplieralert.KindProductHidden {
		t.Fatalf("alerts = %v", got)
	}
}

func TestEvaluate_ReshowOnlyWhatSyncHid(t *testing.T) {
	p := soulStar(link("a", 10, "3258"))
	p.Available = false // hidden by an admin: no HiddenBySyncAt
	res := evaluateProduct(p, env(map[int]catalogView{10: cat("3258", 0.0001, true, "USD")}))
	if res.Show {
		t.Fatal("re-showed a product an admin hid")
	}
	at := evalNow.Add(-time.Hour)
	p.HiddenBySyncAt = &at
	res = evaluateProduct(p, env(map[int]catalogView{10: cat("3258", 0.0001, true, "USD")}))
	if !res.Show {
		t.Fatal("did not re-show a product the sync hid")
	}
}

func TestEvaluate_StatusSyncOffAndDisabledLinksDontHide(t *testing.T) {
	a := link("a", 10, "3258")
	a.StatusSyncOff = true
	b := link("b", 11, "1045")
	b.Enabled = false
	res := evaluateProduct(soulStar(a, b), env(map[int]catalogView{
		10: cat("3258", 0.0001, false, "USD"),
		11: cat("1045", 0.0001, false, "USD"),
	}))
	if res.Hide {
		t.Fatal("hid via a link whose status watch is off / a disabled link")
	}
}

// The active link has its stock watch off and the only watched link is a
// backup that would be paused for a loss: orders still go to the active
// supplier, so the product stays on sale.
func TestEvaluate_UnwatchedActiveLinkKeepsProductOnSale(t *testing.T) {
	active := link("b", 11, "1045")
	active.StatusSyncOff = true
	p := soulStar(active, link("a", 10, "3258"))
	res := evaluateProduct(p, env(map[int]catalogView{
		11: cat("1045", 0.000110, true, "USD"),
		10: cat("3258", 0.000130, true, "USD"), // loss → backup paused
	}))
	if res.Hide {
		t.Fatalf("hid a product whose active supplier is not stock-watched: %v", alertKinds(res))
	}
	if !res.Links[1].Paused() {
		t.Fatal("the losing backup is still paused")
	}
}

// Out of stock at the active supplier, in stock at a backup that would sell
// at a loss: the product goes off sale for the loss, and says so.
func TestEvaluate_OnlyStockIsAtALossSaysLoss(t *testing.T) {
	p := soulStar(link("a", 10, "3258"), link("b", 11, "1045"))
	res := evaluateProduct(p, env(map[int]catalogView{
		10: cat("3258", 0.000110, false, "USD"), // active: no stock
		11: cat("1045", 0.000130, true, "USD"),  // backup: stock, but a loss
	}))
	if !res.Hide || res.HideReason != "loss" {
		t.Fatalf("hide=%v reason=%q, want a loss hide", res.Hide, res.HideReason)
	}
	last := res.Alerts[len(res.Alerts)-1].Message
	if !strings.Contains(last, "costs at least our selling price") {
		t.Fatalf("hide alert = %q", last)
	}
}

func TestEvaluate_PriceSyncOff(t *testing.T) {
	a := link("a", 10, "3258")
	a.PriceSyncOff = true
	a.Sync = prevSync(0.0001, evalNow.Add(-time.Hour))
	res := evaluateProduct(soulStar(a), env(map[int]catalogView{10: cat("3258", 0.0002, true, "USD")}))
	if len(res.Alerts) != 0 || res.Links[0].Paused() || res.Hide {
		t.Fatalf("price watch off must not alert/pause/hide: %+v", res)
	}
	// A link paused before the watch was switched off is released.
	at := evalNow.Add(-time.Hour)
	a.PausedReason, a.PausedAt = product.PausedLoss, &at
	res = evaluateProduct(soulStar(a), env(map[int]catalogView{10: cat("3258", 0.0002, true, "USD")}))
	if res.Links[0].Paused() || !res.Structural || kinds(res)[0] != changeUnpaused {
		t.Fatalf("pause not cleared: %v", kinds(res))
	}
}

func TestEvaluate_MissingFxSkipsLossCheck(t *testing.T) {
	p := &product.Product{
		ID: bson.NewObjectID(), Title: product.I18nString{En: "Alfa 10$"}, Available: true,
		Variants: []product.Variant{{ID: bson.NewObjectID(), Price: 11}},
	}
	p.SupplierLinks = []product.SupplierLink{link("u", 20, "credit:5")}
	p.ActiveSupplierLinkID = "u"
	res := evaluateProduct(p, env(map[int]catalogView{20: cat("credit:5", 1_500_000, true, "LBP")}))
	if res.FxMissing[20] != "LBP" {
		t.Fatalf("fxMissing = %v", res.FxMissing)
	}
	if res.Links[0].Paused() || res.Hide {
		t.Fatal("no rate → no loss decision")
	}
	if res.Links[0].Sync.CostUSD != nil {
		t.Fatal("costUsd must stay unknown without a rate")
	}
}

func TestEvaluate_PackageProductUsesCheapestVariant(t *testing.T) {
	p := &product.Product{
		ID: bson.NewObjectID(), Title: product.I18nString{En: "PUBG 60 UC"}, Available: true,
		Variants: []product.Variant{{ID: bson.NewObjectID(), Price: 1.20}, {ID: bson.NewObjectID(), Price: 0.95}},
	}
	p.SupplierLinks = []product.SupplierLink{link("a", 10, "364")}
	p.ActiveSupplierLinkID = "a"
	res := evaluateProduct(p, env(map[int]catalogView{10: cat("364", 0.97, true, "USD")}))
	if !res.Links[0].Paused() || !res.Hide {
		t.Fatalf("cost 0.97 ≥ cheapest 0.95 should pause+hide: paused=%v hide=%v", res.Links[0].Paused(), res.Hide)
	}
}

func TestEvaluate_LegacyProductHidesOnLossWithoutStoringLinks(t *testing.T) {
	prov := 10
	p := &product.Product{
		ID: bson.NewObjectID(), Title: product.I18nString{En: "Old"}, Available: true,
		Variants:            []product.Variant{{ID: bson.NewObjectID(), Price: 1}},
		FulfillmentProvider: &prov, UpstreamProductID: "77",
	}
	res := evaluateProduct(p, env(map[int]catalogView{10: cat("77", 1.5, true, "USD")}))
	if res.Structural || len(res.Snapshots) != 0 {
		t.Fatal("an unmigrated product's virtual link is never written")
	}
	if !res.Hide || res.HideReason != "loss" {
		t.Fatalf("hide=%v reason=%q", res.Hide, res.HideReason)
	}
}

func TestEvaluate_AlreadyHiddenProductIsNotHiddenAgain(t *testing.T) {
	p := soulStar(link("a", 10, "3258"))
	p.Available = false
	res := evaluateProduct(p, env(map[int]catalogView{10: {Known: true, Products: map[string]provider.CatalogProduct{}}}))
	if res.Hide || res.Show || len(res.Alerts) != 0 {
		t.Fatalf("no-op expected: %+v", res)
	}
}

func TestUnitSellPrice(t *testing.T) {
	if v, ok := unitSellPrice(soulStar()); !ok || v != 0.000123 {
		t.Fatalf("per-unit = %v %v", v, ok)
	}
	if _, ok := unitSellPrice(&product.Product{}); ok {
		t.Fatal("no variants → unknown")
	}
}

func TestTrimMoney(t *testing.T) {
	for in, want := range map[float64]string{1.096: "1.10", 0.0001096: "0.00011", 0: "0.00", 12.5: "12.50"} {
		if got := trimMoney(in); got != want {
			t.Errorf("trimMoney(%v) = %q, want %q", in, got, want)
		}
	}
}

// A supplier listing a product at price 0 (torosfon has one, 2026-11-17) is
// unknown cost, not free: no loss decision, no cost alert, no pause.
func TestEvaluate_ZeroCostIsNotJudged(t *testing.T) {
	p := soulStar(link("t", 14, "372"))
	p.SupplierLinks[0].Sync = prevSync(0.0001, evalNow.Add(-time.Hour))
	res := evaluateProduct(p, env(map[int]catalogView{14: cat("372", 0, true, "USD")}))
	if res.Links[0].Paused() || res.Hide {
		t.Fatal("zero cost must not pause or hide")
	}
	if res.Links[0].Sync.CostUSD != nil {
		t.Fatal("costUsd must stay unknown for a zero price")
	}
	if len(res.Alerts) != 0 {
		t.Fatalf("alerts = %v, want none", alertKinds(res))
	}
}
