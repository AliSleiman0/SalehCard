package supplier

import "testing"

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		v        float64
		currency string
		want     string
	}{
		{450.29, "USD", "$450.29"},
		{-450.29, "USD", "-$450.29"}, // was "$-450.-29" on the dashboard
		{-0.5, "USD", "-$0.50"},
		{-0.001, "USD", "$0.00"}, // rounds to zero → no stray minus
		{1234567.891, "USD", "$1,234,567.89"},
		{-1234.999, "USD", "-$1,235.00"}, // rounding carries into the whole part
		{0.995, "USD", "$1.00"},
		{0, "USD", "$0.00"},
		{1500000, "LBP", "1,500,000 LBP"},
		{-1500000.6, "LBP", "-1,500,001 LBP"},
		{12.5, "EUR", "12.50 EUR"},
		{-12.5, "EUR", "-12.50 EUR"},
	}
	for _, c := range cases {
		if got := formatMoney(c.v, c.currency); got != c.want {
			t.Errorf("formatMoney(%v, %q) = %q, want %q", c.v, c.currency, got, c.want)
		}
	}
}
