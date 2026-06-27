package reseller

import "testing"

func TestTierBody_Validate(t *testing.T) {
	cases := []struct {
		name string
		body tierBody
		ok   bool
	}{
		{"valid", tierBody{Name: "Gold", MarginPercent: 12, BalanceLimit: 15000}, true},
		{"valid zero margin/limit", tierBody{Name: "Bronze", MarginPercent: 0, BalanceLimit: 0}, true},
		{"empty name", tierBody{Name: "  ", MarginPercent: 5, BalanceLimit: 1000}, false},
		{"negative margin", tierBody{Name: "Bad", MarginPercent: -1, BalanceLimit: 1000}, false},
		{"margin over 100", tierBody{Name: "Bad", MarginPercent: 101, BalanceLimit: 1000}, false},
		{"negative balance limit", tierBody{Name: "Bad", MarginPercent: 5, BalanceLimit: -1}, false},
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
