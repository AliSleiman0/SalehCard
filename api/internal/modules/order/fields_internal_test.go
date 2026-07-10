package order

import (
	"testing"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

func TestResolveOrderFields(t *testing.T) {
	p := &product.Product{
		InputFields: []product.InputField{
			{Key: "accountId", Label: product.I18nLabel{En: "Account ID", Ar: "معرّف الحساب"}},
			{Key: "zoneId", Label: product.I18nLabel{En: "Zone ID", Ar: "معرّف المنطقة"}},
			{Key: "password", Label: product.I18nLabel{En: "Password"}, Sensitive: true},
		},
	}
	in := []OrderFieldInput{
		{Key: "accountId", Value: "12345"},
		{Key: "zoneId", Value: "6789"},
		{Key: "password", Value: "hunter2"}, // sensitive → dropped
		{Key: "unknown", Value: "x"},        // not in spec → dropped
		{Key: "accountId2", Value: "  "},    // (unknown + empty) → dropped
	}

	got := resolveOrderFields(p, in, 1)

	if len(got) != 2 {
		t.Fatalf("got %d fields, want 2: %+v", len(got), got)
	}
	if got[0].Key != "accountId" || got[0].Value != "12345" || got[0].Label.En != "Account ID" {
		t.Errorf("field[0] = %+v, want accountId/12345/Account ID (label resolved server-side)", got[0])
	}
	if got[1].Key != "zoneId" || got[1].Label.En != "Zone ID" {
		t.Errorf("field[1] = %+v, want zoneId/Zone ID", got[1])
	}
	for _, f := range got {
		if f.Key == "password" {
			t.Fatal("sensitive field must never be persisted on the order")
		}
	}
}

func TestResolveOrderFields_Empty(t *testing.T) {
	if got := resolveOrderFields(&product.Product{}, nil, 1); got != nil {
		t.Errorf("no inputs → nil, got %+v", got)
	}
	// Inputs but no product spec → nothing resolves (labels can't be trusted).
	p := &product.Product{}
	if got := resolveOrderFields(p, []OrderFieldInput{{Key: "x", Value: "y"}}, 1); got != nil {
		t.Errorf("no spec → nil, got %+v", got)
	}
}

// TestResolveOrderFields_QuantityValueOverridden covers the bug this was
// written to fix: a legacy "quantity"-type field is not a free-text value the
// client can set — its persisted Value must always come from the line's real
// qty, even when the client submits a different (or no) value for it.
func TestResolveOrderFields_QuantityValueOverridden(t *testing.T) {
	p := &product.Product{
		InputFields: []product.InputField{
			{Key: "id", Label: product.I18nLabel{En: "ID"}, Type: product.InputFieldText},
			{Key: "qty", Label: product.I18nLabel{En: "Enter Quantity"}, Type: product.InputFieldQuantity},
		},
	}
	in := []OrderFieldInput{
		{Key: "id", Value: "28472"},
		{Key: "qty", Value: "999"}, // the customer typed a value unrelated to what they're paying for
	}

	got := resolveOrderFields(p, in, 3)

	if len(got) != 2 {
		t.Fatalf("got %d fields, want 2: %+v", len(got), got)
	}
	for _, f := range got {
		if f.Key == "qty" && f.Value != "3" {
			t.Errorf("qty field = %+v, want value overridden to the real qty (3), not the client's 999", f)
		}
	}
}

// TestResolveOrderFields_QuantityAlwaysEmitted ensures the operator always
// sees the real quantity even if the client sends nothing for that field key.
func TestResolveOrderFields_QuantityAlwaysEmitted(t *testing.T) {
	p := &product.Product{
		InputFields: []product.InputField{
			{Key: "qty", Label: product.I18nLabel{En: "Enter Quantity"}, Type: product.InputFieldQuantity},
		},
	}

	got := resolveOrderFields(p, nil, 5)

	if len(got) != 1 || got[0].Key != "qty" || got[0].Value != "5" {
		t.Fatalf("quantity field must always be emitted from qty, got %+v", got)
	}
}

func TestValidateQuantityField_WithinRange(t *testing.T) {
	min, max := 1.0, 5000.0
	p := &product.Product{
		InputFields: []product.InputField{
			{Key: "qty", Type: product.InputFieldQuantity, Constraints: &product.InputFieldConstraints{Min: &min, Max: &max}},
		},
	}
	if err := validateQuantityField(p, 3); err != nil {
		t.Fatalf("qty 3 within [1,5000] should be valid, got %v", err)
	}
}

func TestValidateQuantityField_OutOfRange(t *testing.T) {
	min, max := 30.0, 5000.0
	p := &product.Product{
		InputFields: []product.InputField{
			{Key: "qty", Type: product.InputFieldQuantity, Constraints: &product.InputFieldConstraints{Min: &min, Max: &max}},
		},
	}
	if err := validateQuantityField(p, 5); err == nil {
		t.Fatal("qty 5 below the field's minimum of 30 should be rejected")
	}
}

// TestValidateQuantityField_CorruptZeroZeroIsUnconstrained covers the known
// legacy-import corruption shape ({min:0,max:0} — see
// migration/internal/transform/inputfields.go's corrupt() case): it must not
// literally be enforced, or the affected product becomes permanently
// unorderable.
func TestValidateQuantityField_CorruptZeroZeroIsUnconstrained(t *testing.T) {
	min, max := 0.0, 0.0
	p := &product.Product{
		InputFields: []product.InputField{
			{Key: "qty", Type: product.InputFieldQuantity, Constraints: &product.InputFieldConstraints{Min: &min, Max: &max}},
		},
	}
	if err := validateQuantityField(p, 500); err != nil {
		t.Fatalf("{0,0} is a known corrupt shape and must not block orders, got %v", err)
	}
}

func TestValidateQuantityField_NoQuantityField(t *testing.T) {
	p := &product.Product{}
	if err := validateQuantityField(p, 999999); err != nil {
		t.Fatalf("no quantity field on the product → unconstrained, got %v", err)
	}
}
