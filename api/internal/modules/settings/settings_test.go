package settings

import "testing"

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

func TestIntegrationsView_ConfiguredVsLog(t *testing.T) {
	v := integrationsView("monty", "log")
	if !v["sms"].Configured {
		t.Errorf("sms provider %q should be configured", v["sms"].Provider)
	}
	if v["push"].Configured {
		t.Errorf("push provider %q (log) must not be reported configured", v["push"].Provider)
	}
	// Empty provider is treated as the inert default → not configured.
	if integrationsView("", "")["sms"].Configured {
		t.Errorf("empty provider must not be configured")
	}
}
