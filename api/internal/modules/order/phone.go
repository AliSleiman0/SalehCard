package order

import (
	"regexp"
	"strings"
)

// lebaneseMobile matches a normalized local Lebanese mobile number: an 8-digit
// number whose two-digit prefix is one of the live mobile ranges (03 legacy MIC,
// 70/71/76/78/79/81 touch/alfa). See normalizeLebanesePhone for the canonical form.
var lebaneseMobile = regexp.MustCompile(`^(03|70|71|76|78|79|81)\d{6}$`)

// normalizeLebanesePhone reduces a customer-entered number to the canonical local
// form the bridge dials with: digits only, international/trunk prefixes stripped,
// and the legacy 7-digit "3xxxxxx" form re-expanded to "03xxxxxx". It accepts the
// common shapes a Lebanese customer types — "+961 71 123 456", "0096171123456",
// "03 123 456", "71123456" — and collapses them all to 8 digits. A value that
// still isn't a valid mobile number is caught by validLebaneseMobile.
func normalizeLebanesePhone(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteByte(byte(r))
		}
	}
	d := b.String()
	switch {
	case strings.HasPrefix(d, "00961"):
		d = d[5:]
	case strings.HasPrefix(d, "961"):
		d = d[3:]
	}
	d = strings.TrimPrefix(d, "0") // national trunk prefix
	// The "3" (legacy MIC) range is dialed nationally as 03 — after stripping the
	// trunk 0 it is a bare 7-digit "3xxxxxx"; re-add the 0 so it validates as 03.
	if len(d) == 7 && strings.HasPrefix(d, "3") {
		d = "0" + d
	}
	return d
}

// validLebaneseMobile reports whether a normalized number is a dialable Lebanese
// mobile line. Callers pass the output of normalizeLebanesePhone.
func validLebaneseMobile(normalized string) bool {
	return lebaneseMobile.MatchString(normalized)
}

// bridgePhone extracts the recharge target from a placed line: the explicit
// PlayerID (which the Flutter app snapshots from the first input field) or, failing
// that, a field keyed "phone". Returns "" when neither is present.
func bridgePhone(in PlaceOrderItemInput) string {
	if p := strings.TrimSpace(in.PlayerID); p != "" {
		return p
	}
	for _, f := range in.Fields {
		if f.Key == "phone" {
			return strings.TrimSpace(f.Value)
		}
	}
	return ""
}
