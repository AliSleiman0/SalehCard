package legacy

import (
	"encoding/json"
	"os"
	"testing"
)

// TestParse_PUBGFixture parses a committed real response and asserts the legacy
// shapes decode, including the check_name hook and the string/number-mixed
// numerics — guarding the importer against upstream shape drift.
func TestParse_PUBGFixture(t *testing.T) {
	b, err := os.ReadFile("testdata/category_7_en.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var resp CategoryResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "OK" || len(resp.Data.Products) == 0 {
		t.Fatalf("unexpected envelope: status=%s products=%d", resp.Status, len(resp.Data.Products))
	}
	var uc60 *Product
	for i := range resp.Data.Products {
		if resp.Data.Products[i].ID == 18 {
			uc60 = &resp.Data.Products[i]
		}
	}
	if uc60 == nil {
		t.Fatal("product 18 (UC 60) not found in fixture")
	}
	if uc60.CheckName == nil || uc60.CheckName.Provider != 5 || uc60.CheckName.App != "pubg" {
		t.Fatalf("check_name: %+v", uc60.CheckName)
	}
	if !uc60.MainPrice.Set || uc60.MainPrice.Value != 1 {
		t.Fatalf("mainPrice: %+v", uc60.MainPrice)
	}
}

// TestParse_GiftCardFixture asserts the iTunes fixture decodes its qty range
// (a string/number-mixed {min:1,max:"1"}).
func TestParse_GiftCardFixture(t *testing.T) {
	b, err := os.ReadFile("testdata/category_116_en.json")
	if err != nil {
		t.Skip("itunes fixture absent")
	}
	var resp CategoryResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data.Products) == 0 {
		t.Fatal("no products")
	}
	p := resp.Data.Products[0]
	if p.Qty == nil || !p.Qty.Max.Set || p.Qty.Max.Value != 1 {
		t.Fatalf("qty range did not decode the mixed string/number: %+v", p.Qty)
	}
}
