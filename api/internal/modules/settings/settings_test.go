package settings

import (
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	d := defaults()
	if d.ID != settingsID {
		t.Fatalf("defaults ID = %q, want %q", d.ID, settingsID)
	}
	if d.DefaultLanguage != "en" || d.DefaultCurrency != "USD" {
		t.Fatalf("unexpected default locale/currency: %q/%q", d.DefaultLanguage, d.DefaultCurrency)
	}
	if d.LowStockThreshold != 50 {
		t.Fatalf("default LowStockThreshold = %d, want 50", d.LowStockThreshold)
	}
	// Loyalty ships on-by-default at $5 = 1 point.
	if !d.LoyaltyEnabled {
		t.Errorf("LoyaltyEnabled default = false, want true")
	}
	if d.LoyaltyEarnUsdPerPoint != 5.0 {
		t.Errorf("LoyaltyEarnUsdPerPoint default = %v, want 5", d.LoyaltyEarnUsdPerPoint)
	}
}

func TestNormalizeExchangeRates(t *testing.T) {
	// Trims, drops fully blank rows, keeps good rows.
	out, msg := normalizeExchangeRates([]ExchangeRate{
		{Label: "  SYP ", Value: " 89,500 per $1 "},
		{Label: "", Value: ""}, // dropped
		{Label: "EGP", Value: "49.3"},
	})
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if len(out) != 2 {
		t.Fatalf("got %d rates, want 2", len(out))
	}
	if out[0].Label != "SYP" || out[0].Value != "89,500 per $1" {
		t.Errorf("row not trimmed: %+v", out[0])
	}

	// Half-filled row is rejected.
	if _, msg := normalizeExchangeRates([]ExchangeRate{{Label: "SYP", Value: ""}}); msg == "" {
		t.Errorf("half-filled row should error")
	}

	// Length caps enforced.
	long := strings.Repeat("x", maxExchangeRateText+1)
	if _, msg := normalizeExchangeRates([]ExchangeRate{{Label: long, Value: "1"}}); msg == "" {
		t.Errorf("over-long label should error")
	}

	// Count cap enforced.
	many := make([]ExchangeRate, maxExchangeRates+1)
	for i := range many {
		many[i] = ExchangeRate{Label: "C", Value: "1"}
	}
	if _, msg := normalizeExchangeRates(many); msg == "" {
		t.Errorf("over-limit count should error")
	}
}

func TestIntegrationsView_ConfiguredVsLog(t *testing.T) {
	v := integrationsView("monty", "log", "azureopenai")
	if !v["sms"].Configured {
		t.Errorf("sms provider %q should be configured", v["sms"].Provider)
	}
	if v["push"].Configured {
		t.Errorf("push provider %q (log) must not be reported configured", v["push"].Provider)
	}
	if !v["ai"].Configured {
		t.Errorf("ai provider %q should be configured", v["ai"].Provider)
	}
	if integrationsView("", "", "stub")["ai"].Configured {
		t.Errorf("ai stub must not be reported configured")
	}
	// Empty provider is treated as the inert default → not configured.
	if integrationsView("", "", "")["sms"].Configured {
		t.Errorf("empty provider must not be configured")
	}
}
