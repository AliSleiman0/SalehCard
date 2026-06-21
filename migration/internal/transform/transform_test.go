package transform

import (
	"encoding/json"
	"testing"

	"github.com/AliSleiman0/salehcard/migration/internal/legacy"
	"github.com/AliSleiman0/salehcard/migration/internal/schema"
)

// num builds a "set" FlexNum for fixtures.
func num(v float64) legacy.FlexNum { return legacy.FlexNum{Value: v, Set: true} }

func TestClassifyFulfillment_PUBG_API(t *testing.T) {
	p := &legacy.Product{CheckName: &legacy.CheckName{Provider: 5, App: "pubg"}}
	f, flags := ClassifyFulfillment("games", "uc 60 pubg", p)
	if f.Type != schema.TypeCredit || f.Mode != schema.ModeAPI || f.Confidence != schema.ConfHigh {
		t.Fatalf("PUBG: got type=%s mode=%s conf=%s", f.Type, f.Mode, f.Confidence)
	}
	if f.Provider == nil || *f.Provider != 5 {
		t.Fatalf("PUBG: expected provider 5, got %v", f.Provider)
	}
	if len(flags) != 0 {
		t.Fatalf("PUBG (check_name present): expected no flags, got %v", flags)
	}
}

func TestClassifyFulfillment_GamesNoHook_ParksManual(t *testing.T) {
	f, flags := ClassifyFulfillment("games", "some random topup", &legacy.Product{})
	if f.Mode != schema.ModeManualOperator || f.Confidence != schema.ConfLow {
		t.Fatalf("no-hook game: got mode=%s conf=%s", f.Mode, f.Confidence)
	}
	if !contains(flags, schema.FlagFulfillmentReview) {
		t.Fatalf("no-hook game: expected fulfillment_review flag, got %v", flags)
	}
}

func TestClassifyFulfillment_GiftCard(t *testing.T) {
	f, _ := ClassifyFulfillment("giftcards", "itunes 5$", &legacy.Product{})
	if f.Type != schema.TypeCode || f.Mode != schema.ModeInventory || !f.Cancellable {
		t.Fatalf("giftcard: got type=%s mode=%s cancellable=%v", f.Type, f.Mode, f.Cancellable)
	}
}

func TestClassifyFulfillment_MoneyTransfer(t *testing.T) {
	f, flags := ClassifyFulfillment("money_transfers", "western union", &legacy.Product{})
	if f.Type != schema.TypeTransfer || f.Mode != schema.ModeManualOperator || f.Cancellable {
		t.Fatalf("transfer: got type=%s mode=%s cancellable=%v", f.Type, f.Mode, f.Cancellable)
	}
	if !contains(flags, schema.FlagFxRateReview) {
		t.Fatalf("transfer: expected fx_rate_review, got %v", flags)
	}
}

func TestClassifyFulfillment_TelecomBridgeVsManual(t *testing.T) {
	alfa, _ := ClassifyFulfillment("telecom", "alfa lebanon recharge", &legacy.Product{})
	if alfa.Mode != schema.ModeBridgeDevice {
		t.Fatalf("alfa: expected bridge_device, got %s", alfa.Mode)
	}
	syr, flags := ClassifyFulfillment("telecom", "syriatel سيرياتيل", &legacy.Product{})
	if syr.Mode != schema.ModeManualOperator || !contains(flags, schema.FlagFxRateReview) {
		t.Fatalf("syriatel: got mode=%s flags=%v", syr.Mode, flags)
	}
}

func TestClassifyPricing(t *testing.T) {
	// fixed qty → fixed_package, high confidence, no flag
	pr, flags := ClassifyPricing("giftcards", &legacy.Product{MainPrice: num(5), BasePrice: num(4.83), Qty: &legacy.Range{Min: num(1), Max: num(1)}})
	if pr.Mode != schema.PriceFixedPackage || pr.ModeConfidence != schema.ConfHigh || len(flags) != 0 {
		t.Fatalf("fixed qty: got mode=%s conf=%s flags=%v", pr.Mode, pr.ModeConfidence, flags)
	}
	if pr.Margin != 0.17 {
		t.Fatalf("margin: expected 0.17 (rounded), got %v", pr.Margin)
	}
	// large amount → unit_balance, low, pricing_review
	pr2, flags2 := ClassifyPricing("app_topups", &legacy.Product{MainPrice: num(100), Amount: &legacy.Range{Min: num(10000), Max: num(1500000)}})
	if pr2.Mode != schema.PriceUnitBalance || !contains(flags2, schema.FlagPricingReview) {
		t.Fatalf("unit balance: got mode=%s flags=%v", pr2.Mode, flags2)
	}
}

func TestAmountCorruption(t *testing.T) {
	// qty{0,0} is corruption; qty{1,1} (fixed-1) is NOT.
	_, flags := BuildAmountConstraints(&legacy.Product{Qty: &legacy.Range{Min: num(0), Max: num(0)}})
	if !contains(flags, schema.FlagAmountCorruption) {
		t.Fatalf("qty{0,0}: expected amount_corruption, got %v", flags)
	}
	_, flags2 := BuildAmountConstraints(&legacy.Product{Qty: &legacy.Range{Min: num(1), Max: num(1)}})
	if contains(flags2, schema.FlagAmountCorruption) {
		t.Fatalf("qty{1,1}: should NOT be flagged, got %v", flags2)
	}
	// inverted amount is corruption
	_, flags3 := BuildAmountConstraints(&legacy.Product{Amount: &legacy.Range{Min: num(500), Max: num(10)}})
	if !contains(flags3, schema.FlagAmountCorruption) {
		t.Fatalf("amount{500,10}: expected amount_corruption, got %v", flags3)
	}
}

func TestBuildInputFields_Sensitivity(t *testing.T) {
	reqs := []legacy.Require{
		{ID: 1, Name: "player id", Question: "Enter ID", Type: "text"},
		{ID: 2, Name: "wallet address", Question: "Enter your USDT wallet", Type: "text"},
	}
	fields, sensitive := BuildInputFields(reqs, nil)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	if fields[0].Sensitive {
		t.Fatalf("player id should not be sensitive")
	}
	if !fields[1].Sensitive || !sensitive {
		t.Fatalf("wallet address should be sensitive (field=%v any=%v)", fields[1].Sensitive, sensitive)
	}
}

func TestBuildInputFields_Constraints(t *testing.T) {
	reqs := []legacy.Require{
		{ID: 1, Name: "amount", Question: "Enter Amount", Type: "amount", TypeValue: json.RawMessage(`{"min":"10","max":5000}`)},
		{ID: 2, Name: "qty", Question: "Enter Quantity", Type: "quantity", TypeValue: json.RawMessage(`{"min":1,"max":"1"}`)},
		{ID: 3, Name: "pkg", Question: "Pick", Type: "selectQty", TypeValue: json.RawMessage(`["230","1018"]`)},
	}
	fields, _ := BuildInputFields(reqs, nil)
	if fields[0].Type != "amount" || fields[0].Constraints == nil || *fields[0].Constraints.Max != 5000 {
		t.Fatalf("amount field: %+v", fields[0])
	}
	if fields[1].Type != "quantity" || fields[1].Constraints == nil || *fields[1].Constraints.Min != 1 {
		t.Fatalf("quantity field: %+v", fields[1])
	}
	if fields[2].Type != "select" || len(fields[2].Constraints.Options) != 2 || fields[2].Constraints.Options[0] != "230" {
		t.Fatalf("select field: %+v", fields[2])
	}
}

func TestTransformProduct_PUBG_FullShape(t *testing.T) {
	en := &legacy.Product{
		ID: 18, CategoryID: 7, CategoryName: "PUPG MOBILE ID UC", ProductName: "UC 60",
		Photo: "images/product/x.webp", MainPrice: num(1), BasePrice: num(0.945), Available: true,
		CheckName: &legacy.CheckName{Provider: 5, App: "pubg"},
		Requires:  []legacy.Require{{ID: 1, Name: "player id", Question: "ID", Type: "text"}},
	}
	sp := TransformProduct(en, nil, "games", "pupg-mobile-id-uc")
	if sp.LegacyID != 18 || sp.Title.En == nil || *sp.Title.En != "UC 60" {
		t.Fatalf("title/id: %+v", sp.Title)
	}
	if sp.Fulfillment.Type != schema.TypeCredit || sp.Fulfillment.Mode != schema.ModeAPI || sp.Fulfillment.Provider == nil || *sp.Fulfillment.Provider != 5 {
		t.Fatalf("fulfillment: %+v", sp.Fulfillment)
	}
	if sp.Verification == nil || sp.Verification.App != "pubg" || sp.Verification.Provider != 5 {
		t.Fatalf("verification: %+v", sp.Verification)
	}
	if sp.Status != "active" || len(sp.Images) != 1 || len(sp.InputFields) != 1 {
		t.Fatalf("status/images/inputFields: status=%s images=%d fields=%d", sp.Status, len(sp.Images), len(sp.InputFields))
	}
	if !contains(sp.Flags, schema.FlagNeedsTranslation) {
		t.Fatalf("expected needs_translation (tr null), got %v", sp.Flags)
	}
}

func TestTransformProduct_OrphanFlagged(t *testing.T) {
	en := &legacy.Product{ID: 99, CategoryID: 999999, ProductName: "Mystery", MainPrice: num(1)}
	sp := TransformProduct(en, nil, "", "")
	if !contains(sp.Flags, schema.FlagOrphanCategory) {
		t.Fatalf("expected orphan_category, got %v", sp.Flags)
	}
}

func TestStripHTML(t *testing.T) {
	txt, had := stripHTML("<p>hello &amp; bye</p>")
	if txt != "hello & bye" || !had {
		t.Fatalf("got %q had=%v", txt, had)
	}
	txt2, had2 := stripHTML("plain")
	if txt2 != "plain" || had2 {
		t.Fatalf("plain: got %q had=%v", txt2, had2)
	}
}

func TestApplyOverrides(t *testing.T) {
	products := []schema.Product{{LegacyID: 18, Fulfillment: schema.Fulfillment{Type: schema.TypeCredit, Mode: schema.ModeManualOperator}, Flags: []string{schema.FlagFulfillmentReview, schema.FlagNeedsTranslation}}}
	mode := schema.ModeAPI
	ov := Overrides{"18": {Fulfillment: &struct {
		Type     *string `json:"type"`
		Mode     *string `json:"mode"`
		Provider *int    `json:"provider"`
	}{Mode: &mode}, ClearFlags: []string{schema.FlagFulfillmentReview}}}
	if n := ApplyOverrides(products, ov); n != 1 {
		t.Fatalf("expected 1 touched, got %d", n)
	}
	if products[0].Fulfillment.Mode != schema.ModeAPI {
		t.Fatalf("override mode not applied: %s", products[0].Fulfillment.Mode)
	}
	if contains(products[0].Flags, schema.FlagFulfillmentReview) {
		t.Fatalf("flag not cleared: %v", products[0].Flags)
	}
	if !contains(products[0].Flags, schema.FlagNeedsTranslation) {
		t.Fatalf("unrelated flag should remain: %v", products[0].Flags)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
