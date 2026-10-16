package main

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

func f64(v float64) *float64 { return &v }

// upLive mirrors the migrated UP LIVE product: flat placeholder price, an ID
// field, an "Enter Amount" field bounded 100–200,000, cost $0.0161/diamond.
func upLive() product.Product {
	return product.Product{
		ID:              bson.NewObjectID(),
		FulfillmentType: product.FulfillmentCredit,
		FulfillmentMode: product.FulfillmentModeManualOperator,
		Variants:        []product.Variant{{ID: bson.NewObjectID(), Denomination: "10", Price: 100}},
		InputFields: []product.InputField{
			{Key: "field_1", Type: product.InputFieldText},
			{Key: "amount", Type: product.InputFieldAmount, Constraints: &product.InputFieldConstraints{Min: f64(100), Max: f64(200000)}},
		},
		Pricing: &product.Pricing{Cost: 0.01610546875, Currency: "USD", Mode: product.PricingModeUnitBalance},
	}
}

func TestPlan_ConvertsUpLive(t *testing.T) {
	ch, reason := plan(upLive(), false, 10)
	if reason != "" {
		t.Fatalf("unexpected skip: %s", reason)
	}
	if ch.FieldKey != "amount" || ch.BlockUnits != 100 || ch.Min != 100 || ch.Max != 200000 {
		t.Fatalf("bad change %+v", ch)
	}
	if ch.Price != 1.77 { // 0.0161 × 100 × 1.10 = 1.7716
		t.Fatalf("price = %v, want 1.77", ch.Price)
	}
	if ch.Label != "per 100" {
		t.Fatalf("label = %q", ch.Label)
	}
}

func TestPlan_Skips(t *testing.T) {
	cases := map[string]func(p *product.Product){
		"already per-unit": func(p *product.Product) { p.UnitPricing = &product.UnitPricing{FieldKey: "amount", BlockUnits: 1} },
		"code":             func(p *product.Product) { p.FulfillmentType = product.FulfillmentCode },
		"api":              func(p *product.Product) { p.FulfillmentMode = product.FulfillmentModeAPI },
		"two variants":     func(p *product.Product) { p.Variants = append(p.Variants, product.Variant{Price: 1}) },
		"no cost":          func(p *product.Product) { p.Pricing.Cost = 0 },
		"no pricing":       func(p *product.Product) { p.Pricing = nil },
		"fixed mode":       func(p *product.Product) { p.Pricing.Mode = product.PricingModeFixedPackage },
		"no mode":          func(p *product.Product) { p.Pricing.Mode = "" },
		"non-USD cost":     func(p *product.Product) { p.Pricing.Currency = "LBP" },
		"pack-priced":      func(p *product.Product) { p.Pricing.Cost = 5.25 },
		"no bounds":        func(p *product.Product) { p.InputFields[1].Constraints = nil },
		"quantity": func(p *product.Product) {
			p.InputFields = append(p.InputFields, product.InputField{Type: product.InputFieldQuantity})
		},
		"two amounts": func(p *product.Product) {
			p.InputFields = append(p.InputFields, product.InputField{Key: "a2", Type: product.InputFieldAmount})
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := upLive()
			mutate(&p)
			if _, reason := plan(p, false, 10); reason == "" {
				t.Fatal("expected a skip")
			}
		})
	}
}

// An admin who switched a product back to fixed packages (unitPricing stored
// as null) is respected: skipped with a clear reason, never "would convert".
func TestPlan_AdminOptedOut(t *testing.T) {
	_, reason := plan(upLive(), true, 10)
	if !strings.Contains(reason, "admin opted out") {
		t.Fatalf("reason = %q, want admin opted out", reason)
	}
}

func TestPlan_EmptyCurrencyAccepted(t *testing.T) {
	p := upLive()
	p.Pricing.Currency = ""
	if _, reason := plan(p, false, 10); reason != "" {
		t.Fatalf("unexpected skip: %s", reason)
	}
	p.Pricing.Currency = "usd"
	if _, reason := plan(p, false, 10); reason != "" {
		t.Fatalf("unexpected skip for lowercase usd: %s", reason)
	}
}

func TestUnitPricingIsNull(t *testing.T) {
	mk := func(d bson.D) bson.Raw {
		b, err := bson.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	if !unitPricingIsNull(mk(bson.D{{Key: "unitPricing", Value: nil}})) {
		t.Fatal("explicit null not detected")
	}
	if unitPricingIsNull(mk(bson.D{{Key: "title", Value: "x"}})) {
		t.Fatal("absent key read as opted out")
	}
	if unitPricingIsNull(mk(bson.D{{Key: "unitPricing", Value: bson.D{{Key: "blockUnits", Value: 100}}}})) {
		t.Fatal("set unitPricing read as opted out")
	}
}

func TestBlockFor(t *testing.T) {
	cases := map[float64]float64{
		2:             1,
		0.5:           10,
		0.0161:        100,
		0.00010668:    10000,
		0.00008678458: 100000,
	}
	for cost, want := range cases {
		if got := blockFor(cost); got != want {
			t.Errorf("blockFor(%v) = %v, want %v", cost, got, want)
		}
	}
}
