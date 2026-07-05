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

	got := resolveOrderFields(p, in)

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
	if got := resolveOrderFields(&product.Product{}, nil); got != nil {
		t.Errorf("no inputs → nil, got %+v", got)
	}
	// Inputs but no product spec → nothing resolves (labels can't be trusted).
	p := &product.Product{}
	if got := resolveOrderFields(p, []OrderFieldInput{{Key: "x", Value: "y"}}); got != nil {
		t.Errorf("no spec → nil, got %+v", got)
	}
}
