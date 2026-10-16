package product

import (
	"fmt"
	"math"
	"strings"
)

// RoundCents rounds a USD amount to the nearest cent (half away from zero).
func RoundCents(v float64) float64 {
	return math.Round(v*100) / 100
}

// UnitLinePrice is the price of one per-unit line: the block price (already
// reseller/offer-adjusted) scaled to the typed amount, rounded to the cent.
func UnitLinePrice(blockPrice, amount, blockUnits float64) float64 {
	return RoundCents(blockPrice * amount / blockUnits)
}

// isClear reports whether u is the "back to fixed packages" sentinel.
func (u *UnitPricing) isClear() bool {
	return u != nil && u.BlockUnits == 0
}

// UnitAmountField returns the amount-type input field that drives a per-unit
// product's price, or false when the product is not (validly) unit-priced —
// e.g. its driving field was later removed. Callers must refuse to price a
// line in that case rather than fall back to the flat variant price.
func (p *Product) UnitAmountField() (InputField, bool) {
	if p.UnitPricing == nil || p.UnitPricing.BlockUnits <= 0 {
		return InputField{}, false
	}
	for _, f := range p.InputFields {
		if f.Key == p.UnitPricing.FieldKey && f.Type == InputFieldAmount {
			return f, true
		}
	}
	return InputField{}, false
}

// unitPricingTarget is the effective product shape a UnitPricing spec is
// validated against (the stored product overlaid with an update's fields).
type unitPricingTarget struct {
	Type     FulfillmentType
	Mode     FulfillmentMode
	Variants []Variant
	Fields   []InputField
}

// normalizeUnitPricing trims the spec; nil and the clear sentinel pass through.
func normalizeUnitPricing(u *UnitPricing) *UnitPricing {
	if u == nil {
		return nil
	}
	return &UnitPricing{FieldKey: strings.TrimSpace(u.FieldKey), BlockUnits: u.BlockUnits}
}

// validateUnitPricing enforces the per-unit invariants: a positive block size,
// a driving amount field with a positive min and max ≥ min, exactly one
// variant (the block price), an account_credit product fulfilled by hand (v1:
// supplier/bridge dispatch has its own qty semantics), and no quantity-type
// field (its pack-count bounds would contradict qty == 1).
func validateUnitPricing(u *UnitPricing, t unitPricingTarget) error {
	if u == nil || u.isClear() {
		return nil
	}
	if u.BlockUnits < 0 || math.IsNaN(u.BlockUnits) || math.IsInf(u.BlockUnits, 0) {
		return badRequest("per-unit pricing needs a positive number of units per price")
	}
	if t.Type != FulfillmentCredit {
		return badRequest("per-unit pricing is only available on account-credit products")
	}
	mode := t.Mode
	if mode == "" {
		mode = DeriveMode(t.Type)
	}
	if mode != FulfillmentModeManualOperator {
		return badRequest("per-unit pricing is only available on manually fulfilled products")
	}
	if len(t.Variants) != 1 {
		return badRequest("per-unit pricing needs exactly one price row")
	}
	if t.Variants[0].Price <= 0 {
		return badRequest("per-unit pricing needs a positive price")
	}
	var driver *InputField
	for i := range t.Fields {
		f := &t.Fields[i]
		if f.Type == InputFieldQuantity {
			return badRequest(fmt.Sprintf("remove the quantity field %q — per-unit products are bought by amount", f.Key))
		}
		if f.Key == u.FieldKey {
			driver = f
		}
	}
	if driver == nil || driver.Type != InputFieldAmount {
		return badRequest("per-unit pricing must point at an amount-type input field")
	}
	c := driver.Constraints
	if c == nil || c.Min == nil || c.Max == nil || *c.Min <= 0 || *c.Max < *c.Min {
		return badRequest(fmt.Sprintf("the amount field %q needs a minimum above 0 and a maximum", driver.Key))
	}
	return nil
}

// DiscountFixedType is the offer discount type for an absolute amount off
// (mirrors offer.DiscountFixed / Discount.Type; product must not import offer).
const DiscountFixedType = "fixed"

// AcceptsDiscount reports whether an offer of discountType may apply to p.
// Per-unit products refuse fixed-amount offers: the discount lands on the
// block price before it is scaled to the typed amount, so "$X off" would
// become "$X off per block" (and a discount ≥ the block price zeroes every
// line). Percent offers scale correctly and stay allowed.
func (p *Product) AcceptsDiscount(discountType string) bool {
	return p.UnitPricing == nil || discountType != DiscountFixedType
}

// UnitFromPrice rescales a block price to the cheapest purchasable line of a
// validly configured per-unit product: blockPrice × min ÷ blockUnits, rounded
// to the cent, where min is the driving field's minimum (one block when it has
// none). It matches the web/app list-tile "from" price. ok is false when p is
// not (validly) unit-priced.
func (p *Product) UnitFromPrice(blockPrice float64) (float64, bool) {
	f, ok := p.UnitAmountField()
	if !ok || len(p.Variants) != 1 {
		return 0, false
	}
	amount := p.UnitPricing.BlockUnits
	if c := f.Constraints; c != nil && c.Min != nil && *c.Min > 0 {
		amount = *c.Min
	}
	return UnitLinePrice(blockPrice, amount, p.UnitPricing.BlockUnits), true
}

// FromPrice is a product's lowest list price ("from $X"), 0 without variants.
// A per-unit product quotes the price of its minimum amount, not the raw
// block price.
func FromPrice(p *Product) float64 {
	return FromPriceWith(p, func(v float64) float64 { return v })
}

// FromPriceWith is FromPrice over variant prices first transformed by adjust
// (e.g. an offer discount). For a per-unit product the adjusted block price is
// rescaled after adjustment, exactly as checkout prices a line.
func FromPriceWith(p *Product, adjust func(float64) float64) float64 {
	from := 0.0
	for i, v := range p.Variants {
		if a := adjust(v.Price); i == 0 || a < from {
			from = a
		}
	}
	if len(p.Variants) > 0 {
		if u, ok := p.UnitFromPrice(from); ok {
			return u
		}
	}
	return from
}
