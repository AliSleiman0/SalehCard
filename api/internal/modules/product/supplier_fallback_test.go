package product

import (
	"encoding/json"
	"testing"
)

func TestAdminProductView_SupplierFallback(t *testing.T) {
	p := &Product{}
	if !NewAdminProductView(p).SupplierFallback {
		t.Fatal("fallback defaults to on for every existing product")
	}
	p.SupplierFallbackOff = true
	if NewAdminProductView(p).SupplierFallback {
		t.Fatal("the stored switch turns it off")
	}
	// Never in public product JSON.
	b, _ := json.Marshal(p)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if _, ok := m["supplierFallbackOff"]; ok {
		t.Fatal("supplierFallbackOff leaked into public JSON")
	}
}
