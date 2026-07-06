package order

import "testing"

func TestNormalizeAndValidateLebaneseMobile(t *testing.T) {
	valid := map[string]string{
		"71123456":        "71123456",
		"03123456":        "03123456",
		"70 123 456":      "70123456",
		"76-123-456":      "76123456",
		"+961 71 123 456": "71123456",
		"0096171123456":   "71123456",
		"+9613123456":     "03123456", // legacy 3-range re-expands to 03
		"961 3 123 456":   "03123456",
		"071123456":       "71123456", // trunk 0 stripped
		"81 123 456":      "81123456",
	}
	for raw, want := range valid {
		got := normalizeLebanesePhone(raw)
		if got != want {
			t.Errorf("normalize(%q) = %q, want %q", raw, got, want)
		}
		if !validLebaneseMobile(got) {
			t.Errorf("validLebaneseMobile(%q) = false, want true (from %q)", got, raw)
		}
	}

	invalid := []string{
		"",
		"12345678",    // 12 is not a mobile prefix
		"7112345",     // too short
		"711234567",   // too long
		"+1 555 1234", // not Lebanese
		"hello",
	}
	for _, raw := range invalid {
		if validLebaneseMobile(normalizeLebanesePhone(raw)) {
			t.Errorf("validLebaneseMobile(normalize(%q)) = true, want false", raw)
		}
	}
}

func TestBridgePhonePrefersPlayerID(t *testing.T) {
	in := PlaceOrderItemInput{PlayerID: " 71123456 ", Fields: []OrderFieldInput{{Key: "phone", Value: "70000000"}}}
	if got := bridgePhone(in); got != "71123456" {
		t.Errorf("bridgePhone playerID = %q, want 71123456", got)
	}
	in2 := PlaceOrderItemInput{Fields: []OrderFieldInput{{Key: "other", Value: "x"}, {Key: "phone", Value: " 70111222 "}}}
	if got := bridgePhone(in2); got != "70111222" {
		t.Errorf("bridgePhone field = %q, want 70111222", got)
	}
	if got := bridgePhone(PlaceOrderItemInput{}); got != "" {
		t.Errorf("bridgePhone empty = %q, want empty", got)
	}
}
