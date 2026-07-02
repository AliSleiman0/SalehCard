package reseller

import "testing"

func TestTierBody_Validate(t *testing.T) {
	cases := []struct {
		name string
		body tierBody
		ok   bool
	}{
		{"valid", tierBody{Name: "Gold", MarginPercent: 12}, true},
		{"valid zero margin", tierBody{Name: "Bronze", MarginPercent: 0}, true},
		{"empty name", tierBody{Name: "  ", MarginPercent: 5}, false},
		{"negative margin", tierBody{Name: "Bad", MarginPercent: -1}, false},
		{"margin over 100", tierBody{Name: "Bad", MarginPercent: 101}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := tc.body.validate()
			if ok != tc.ok {
				t.Fatalf("validate() ok = %v, want %v", ok, tc.ok)
			}
		})
	}
}
