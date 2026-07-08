package main

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

func TestProviderFor(t *testing.T) {
	cases := map[string]product.BridgeProvider{
		"alfa-direct":  product.BridgeProviderAlfa,
		"mtc-direct":   product.BridgeProviderTouch,
		"touch-direct": product.BridgeProviderTouch,
	}
	for cat, want := range cases {
		got, err := providerFor(cat)
		if err != nil {
			t.Fatalf("providerFor(%q): %v", cat, err)
		}
		if got != want {
			t.Errorf("providerFor(%q) = %q, want %q", cat, got, want)
		}
	}
	if _, err := providerFor("google-play"); err == nil {
		t.Errorf("providerFor(google-play): want error, got nil")
	}
}

// A code/pin product (the ALFA 3.03 DIRECT case) must come out as a fully-wired
// recharge_line bridge product with exactly one `phone` field reusing the
// existing localized number label, and qty dropped.
func TestPlanUpdate_CodeProductToBridge(t *testing.T) {
	bridgeMethod = product.BridgeMethodRechargeLine
	p := rawProduct{
		ID:       bson.NewObjectID(),
		Category: "alfa-direct",
		Title:    product.I18nString{En: "ALFA 3.03 DIRECT"},
		FulType:  "code",
		FulMode:  "manual_operator",
		InputFields: []rawField{
			{Key: "field_1", Label: product.I18nLabel{En: "Enter phone number", Ar: "ادخل رقم الهاتف"}, Type: "text"},
			{Key: "qty", Label: product.I18nLabel{En: "Enter Quantity"}, Type: "quantity"},
		},
	}
	set, spec, phone, err := planUpdate(p)
	if err != nil {
		t.Fatalf("planUpdate: %v", err)
	}
	if spec.Provider != product.BridgeProviderAlfa || spec.Method != product.BridgeMethodRechargeLine {
		t.Errorf("spec = %+v, want alfa/recharge_line", spec)
	}
	if phone.Key != "phone" || phone.Type != product.InputFieldText {
		t.Errorf("phone field = %+v, want key=phone type=text", phone)
	}
	// Label reused from the existing text field, not the default.
	if phone.Label.En != "Enter phone number" || phone.Label.Ar != "ادخل رقم الهاتف" {
		t.Errorf("phone label not reused: %+v", phone.Label)
	}
	// The $set drives fulfillmentType/Mode to the bridge target.
	m := toMap(set)
	if m["fulfillmentType"] != string(product.FulfillmentCredit) {
		t.Errorf("fulfillmentType = %v, want account_credit", m["fulfillmentType"])
	}
	if m["fulfillmentMode"] != string(product.FulfillmentModeBridgeDevice) {
		t.Errorf("fulfillmentMode = %v, want bridge_device", m["fulfillmentMode"])
	}
	fields, ok := m["inputFields"].([]product.InputField)
	if !ok || len(fields) != 1 || fields[0].Key != "phone" {
		t.Errorf("inputFields = %v, want exactly [phone] (qty dropped)", m["inputFields"])
	}
}

// A product with no usable label falls back to the default bilingual label.
func TestPhoneField_DefaultLabel(t *testing.T) {
	f := phoneField(nil)
	if f.Key != "phone" || f.Label.En != "Mobile number" || f.Label.Ar != "رقم الهاتف" {
		t.Errorf("default phone field wrong: %+v", f)
	}
}

func toMap(d bson.D) map[string]any {
	m := make(map[string]any, len(d))
	for _, e := range d {
		m[e.Key] = e.Value
	}
	return m
}
