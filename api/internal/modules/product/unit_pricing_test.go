package product

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func f64(v float64) *float64 { return &v }

// coinTarget is a valid per-unit shape: one priced variant, an ID field and an
// amount field bounded 100–200,000, manually fulfilled account credit.
func coinTarget() unitPricingTarget {
	return unitPricingTarget{
		Type:     FulfillmentCredit,
		Mode:     FulfillmentModeManualOperator,
		Variants: []Variant{{ID: bson.NewObjectID(), Price: 1.75}},
		Fields: []InputField{
			{Key: "id", Type: InputFieldText},
			{Key: "amount", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Min: f64(100), Max: f64(200000)}},
		},
	}
}

func TestValidateUnitPricing(t *testing.T) {
	ok := &UnitPricing{FieldKey: "amount", BlockUnits: 100}
	tests := []struct {
		name    string
		up      *UnitPricing
		mutate  func(t *unitPricingTarget)
		wantErr bool
	}{
		{name: "nil passes", up: nil},
		{name: "clear sentinel passes on anything", up: &UnitPricing{}, mutate: func(t *unitPricingTarget) { t.Type = FulfillmentCode }},
		{name: "valid", up: ok},
		{name: "derived manual mode ok", up: ok, mutate: func(t *unitPricingTarget) { t.Mode = "" }},
		{name: "negative block", up: &UnitPricing{FieldKey: "amount", BlockUnits: -1}, wantErr: true},
		{name: "code product", up: ok, mutate: func(t *unitPricingTarget) { t.Type = FulfillmentCode }, wantErr: true},
		{name: "api mode", up: ok, mutate: func(t *unitPricingTarget) { t.Mode = FulfillmentModeAPI }, wantErr: true},
		{name: "bridge mode", up: ok, mutate: func(t *unitPricingTarget) { t.Mode = FulfillmentModeBridgeDevice }, wantErr: true},
		{name: "two variants", up: ok, mutate: func(t *unitPricingTarget) { t.Variants = append(t.Variants, Variant{Price: 2}) }, wantErr: true},
		{name: "zero price", up: ok, mutate: func(t *unitPricingTarget) { t.Variants[0].Price = 0 }, wantErr: true},
		{name: "unknown field", up: &UnitPricing{FieldKey: "nope", BlockUnits: 100}, wantErr: true},
		{name: "driver not amount", up: &UnitPricing{FieldKey: "id", BlockUnits: 100}, wantErr: true},
		{name: "driver unbounded", up: ok, mutate: func(t *unitPricingTarget) { t.Fields[1].Constraints = nil }, wantErr: true},
		{name: "driver zero min", up: ok, mutate: func(t *unitPricingTarget) { t.Fields[1].Constraints.Min = f64(0) }, wantErr: true},
		{name: "quantity field present", up: ok, mutate: func(t *unitPricingTarget) {
			t.Fields = append(t.Fields, InputField{Key: "qty", Type: InputFieldQuantity})
		}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target := coinTarget()
			if tc.mutate != nil {
				tc.mutate(&target)
			}
			err := validateUnitPricing(tc.up, target)
			if tc.wantErr != (err != nil) {
				t.Fatalf("wantErr=%v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestUnitAmountField(t *testing.T) {
	tg := coinTarget()
	p := &Product{InputFields: tg.Fields, UnitPricing: &UnitPricing{FieldKey: "amount", BlockUnits: 100}}
	if f, ok := p.UnitAmountField(); !ok || f.Key != "amount" {
		t.Fatalf("want amount field, got %+v %v", f, ok)
	}
	p.InputFields = p.InputFields[:1]
	if _, ok := p.UnitAmountField(); ok {
		t.Fatal("removed driving field must not resolve")
	}
	if _, ok := (&Product{}).UnitAmountField(); ok {
		t.Fatal("fixed product must not resolve")
	}
}

func TestUnitLinePrice(t *testing.T) {
	cases := []struct{ block, amount, units, want float64 }{
		{10, 45000, 90000, 5},
		{10, 10000, 90000, 1.11},
		{1.75, 45000, 100, 787.5},
		{0.835, 100000, 100000, 0.84},
	}
	for _, c := range cases {
		if got := UnitLinePrice(c.block, c.amount, c.units); got != c.want {
			t.Errorf("UnitLinePrice(%v,%v,%v)=%v want %v", c.block, c.amount, c.units, got, c.want)
		}
	}
}
