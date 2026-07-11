package product

// ResellerUnitPrice computes the price a reseller pays for a variant: the
// lowest of retail, the tier-margin price (retail * (1 - margin%/100)), the
// per-variant global reseller override, and a per-reseller custom price
// (variantId → price). margin and custom are preloaded once per request by the
// caller.
//
// It is the single pricing rule shared by checkout (order.PlaceOrder) and the
// catalog's reseller enrichment (ProductService.enrichReseller) — keep both on
// this function so the price a reseller sees always equals the price they are
// charged.
func ResellerUnitPrice(v Variant, marginPct float64, custom map[string]float64) float64 {
	price := v.Price
	if marginPct > 0 && marginPct < 100 {
		if marginPrice := v.Price * (1 - marginPct/100); marginPrice < price {
			price = marginPrice
		}
	}
	if v.ResellerPrice != nil && *v.ResellerPrice < price {
		price = *v.ResellerPrice
	}
	if custom != nil {
		if cp, ok := custom[v.ID.Hex()]; ok && cp < price {
			price = cp
		}
	}
	return price
}
