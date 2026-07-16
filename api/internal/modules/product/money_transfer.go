package product

import "strings"

// maxCurrencyLen caps a currency code's length. Real codes are 3 chars
// (USD, LBP, SYP, EGP…); the ceiling just guards against pasted junk.
const maxCurrencyLen = 8

// hasRates reports whether the rate board offers at least one usable rate.
// A zero rate means "not offered", so a spec with neither is effectively empty.
func (m *MoneyTransfer) hasRates() bool {
	return m != nil && (m.BuyRate > 0 || m.SellRate > 0)
}

// validateMoneyTransfer normalizes and validates a product's buy/sell rate
// board, mirroring validateBridge. It returns the normalized copy (currencies
// upper-cased/trimmed) or a badRequest error.
//
//   - nil in → nil out (no rate board).
//   - An all-empty spec (no currencies, no rates) is the clear sentinel: it is
//     returned as-is so the repository can unset any stored rates on update.
//   - A populated spec is rejected on a non-transfer product (when the type is
//     known), and must carry both currencies plus at least one positive rate;
//     rates must be non-negative and, when both amount limits are set, min ≤ max.
func validateMoneyTransfer(ft FulfillmentType, mt *MoneyTransfer) (*MoneyTransfer, error) {
	if mt == nil {
		return nil, nil
	}
	out := &MoneyTransfer{
		BaseCurrency:  strings.ToUpper(strings.TrimSpace(mt.BaseCurrency)),
		QuoteCurrency: strings.ToUpper(strings.TrimSpace(mt.QuoteCurrency)),
		BuyRate:       mt.BuyRate,
		SellRate:      mt.SellRate,
		MinAmount:     mt.MinAmount,
		MaxAmount:     mt.MaxAmount,
	}
	// Clear sentinel: an explicit "remove the rate board" request. Skip all
	// validation so a clear works regardless of the product's fulfillment type.
	if out.BaseCurrency == "" && out.QuoteCurrency == "" && out.BuyRate == 0 && out.SellRate == 0 {
		return out, nil
	}
	// Only enforce the type guard when the type is known (empty = "unchanged" on
	// a partial update). The admin editor always sends fulfillmentType on save.
	if ft != "" && ft != FulfillmentTransfer {
		return nil, badRequest("exchange rates are only valid on a money-transfer product")
	}
	if out.BaseCurrency == "" || out.QuoteCurrency == "" {
		return nil, badRequest("exchange rates need both a base and a quote currency")
	}
	if len(out.BaseCurrency) > maxCurrencyLen || len(out.QuoteCurrency) > maxCurrencyLen {
		return nil, badRequest("currency codes are too long")
	}
	if out.BuyRate < 0 || out.SellRate < 0 {
		return nil, badRequest("exchange rates cannot be negative")
	}
	if out.BuyRate == 0 && out.SellRate == 0 {
		return nil, badRequest("set at least one of the buy or sell rate")
	}
	if out.MinAmount < 0 || out.MaxAmount < 0 {
		return nil, badRequest("amount limits cannot be negative")
	}
	if out.MinAmount > 0 && out.MaxAmount > 0 && out.MinAmount > out.MaxAmount {
		return nil, badRequest("minimum amount cannot exceed the maximum")
	}
	return out, nil
}
