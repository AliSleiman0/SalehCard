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
