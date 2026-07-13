package order

import (
	"strings"
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

// --- validateFieldInputs -----------------------------------------------------

func selectProduct(options ...string) *product.Product {
	return &product.Product{
		InputFields: []product.InputField{{
			Key:         "server",
			Label:       product.I18nLabel{En: "Server"},
			Type:        product.InputFieldSelect,
			Constraints: &product.InputFieldConstraints{Options: options},
		}},
	}
}

func amountProduct(lo, hi *float64) *product.Product {
	return &product.Product{
		InputFields: []product.InputField{{
			Key:         "topup",
			Label:       product.I18nLabel{En: "Top-up amount"},
			Type:        product.InputFieldAmount,
			Constraints: &product.InputFieldConstraints{Min: lo, Max: hi},
		}},
	}
}

func TestValidateFieldInputs(t *testing.T) {
	lo, hi := 1.0, 100.0
	zero := 0.0
	inverted := 500.0

	tests := []struct {
		name    string
		p       *product.Product
		in      []OrderFieldInput
		wantErr bool
	}{
		{name: "no input fields on product", p: &product.Product{}, in: []OrderFieldInput{{Key: "x", Value: "y"}}},
		{name: "select value in options passes", p: selectProduct("EU", "NA"), in: []OrderFieldInput{{Key: "server", Value: "EU"}}},
		{name: "select value not in options rejected", p: selectProduct("EU", "NA"), in: []OrderFieldInput{{Key: "server", Value: "ASIA"}}, wantErr: true},
		{name: "select matches option with surrounding spaces", p: selectProduct(" EU "), in: []OrderFieldInput{{Key: "server", Value: "EU"}}},
		{name: "empty select value skipped (fields are optional)", p: selectProduct("EU"), in: []OrderFieldInput{{Key: "server", Value: "  "}}},
		{name: "select with empty options is legacy free-entry", p: selectProduct(), in: []OrderFieldInput{{Key: "server", Value: "anything"}}},
		{name: "amount within bounds passes", p: amountProduct(&lo, &hi), in: []OrderFieldInput{{Key: "topup", Value: "50"}}},
		{name: "amount at the boundaries passes", p: amountProduct(&lo, &hi), in: []OrderFieldInput{{Key: "topup", Value: "100"}}},
		{name: "amount below min rejected", p: amountProduct(&lo, &hi), in: []OrderFieldInput{{Key: "topup", Value: "0.5"}}, wantErr: true},
		{name: "amount above max rejected", p: amountProduct(&lo, &hi), in: []OrderFieldInput{{Key: "topup", Value: "500"}}, wantErr: true},
		{name: "non-numeric amount with active bounds rejected", p: amountProduct(&lo, &hi), in: []OrderFieldInput{{Key: "topup", Value: "abc"}}, wantErr: true},
		{name: "non-numeric amount without bounds passes (today's behavior)", p: amountProduct(nil, nil), in: []OrderFieldInput{{Key: "topup", Value: "abc"}}},
		{name: "corrupt zero-zero amount bound skipped", p: amountProduct(&zero, &zero), in: []OrderFieldInput{{Key: "topup", Value: "999999"}}},
		{name: "corrupt min>max amount bound skipped", p: amountProduct(&inverted, &hi), in: []OrderFieldInput{{Key: "topup", Value: "1"}}},
		{name: "one-sided min-only enforced", p: amountProduct(&lo, nil), in: []OrderFieldInput{{Key: "topup", Value: "0.5"}}, wantErr: true},
		{name: "one-sided max-only enforced", p: amountProduct(nil, &hi), in: []OrderFieldInput{{Key: "topup", Value: "101"}}, wantErr: true},
		{name: "one-sided min-only passes above it", p: amountProduct(&lo, nil), in: []OrderFieldInput{{Key: "topup", Value: "999999"}}},
		{name: "empty amount value skipped", p: amountProduct(&lo, &hi), in: []OrderFieldInput{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFieldInputs(tt.p, tt.in)
			if tt.wantErr && err == nil {
				t.Fatal("expected a validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// Quantity-type fields are enforced against the line's real qty by
// validateQuantityField — validateFieldInputs must ignore them entirely,
// whatever the client typed.
func TestValidateFieldInputs_IgnoresQuantityFields(t *testing.T) {
	lo, hi := 1.0, 5.0
	p := &product.Product{
		InputFields: []product.InputField{{
			Key:         "qty",
			Type:        product.InputFieldQuantity,
			Constraints: &product.InputFieldConstraints{Min: &lo, Max: &hi},
		}},
	}
	if err := validateFieldInputs(p, []OrderFieldInput{{Key: "qty", Value: "999"}}); err != nil {
		t.Fatalf("quantity fields are validateQuantityField's job, got %v", err)
	}
}

// A rejection must name the field, never echo the submitted value — the field
// may be sensitive.
func TestValidateFieldInputs_ErrorNeverEchoesValue(t *testing.T) {
	p := &product.Product{
		InputFields: []product.InputField{{
			Key:         "secretChoice",
			Label:       product.I18nLabel{En: "Secret choice"},
			Type:        product.InputFieldSelect,
			Sensitive:   true,
			Constraints: &product.InputFieldConstraints{Options: []string{"a", "b"}},
		}},
	}
	secret := "hunter2-secret-value"
	err := validateFieldInputs(p, []OrderFieldInput{{Key: "secretChoice", Value: secret}})
	if err == nil {
		t.Fatal("out-of-options value should be rejected")
	}
	if got := err.Error(); strings.Contains(got, secret) {
		t.Fatalf("error message must not echo the submitted value: %q", got)
	}
}
