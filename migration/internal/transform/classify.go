package transform

import (
	"math"
	"regexp"

	"github.com/AliSleiman0/salehcard/migration/internal/legacy"
	"github.com/AliSleiman0/salehcard/migration/internal/schema"
)

// round6 trims binary-float noise from a computed value (e.g. retail-cost).
func round6(f float64) float64 { return math.Round(f*1e6) / 1e6 }

// Carrier / keyword matchers (English + Arabic). Arabic is unaffected by
// lowercasing, so matchText concatenates lower(en) + " " + ar.
var (
	reAlfa      = regexp.MustCompile(`(?i)alfa|الفا`)
	reTouch     = regexp.MustCompile(`(?i)touch|تاتش|تتش`)
	reSyriatel  = regexp.MustCompile(`(?i)syriatel|سيرياتيل|سيريتل`)
	reMTN       = regexp.MustCompile(`(?i)mtn|ام تي ان|إم تي إن`)
	reIMO       = regexp.MustCompile(`(?i)\bimo\b|ايمو|أيمو`)
	reAutomatic = regexp.MustCompile(`(?i)تلقائي|automatic|auto`)
	reManual    = regexp.MustCompile(`يدوي|عرض`) // manual / offer (Arabic); English "manual" below
	reManualEn  = regexp.MustCompile(`(?i)manual`)
)

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }

// ClassifyFulfillment resolves a product's fulfillment classification from its
// root domain and name (spec §5/§6), returning the classification and any review
// flags. matchText is lower(en name) + " " + ar name.
func ClassifyFulfillment(domain, matchText string, p *legacy.Product) (schema.Fulfillment, []string) {
	var flags []string

	switch domain {
	case "giftcards":
		return schema.Fulfillment{Type: schema.TypeCode, Mode: schema.ModeInventory, Confidence: schema.ConfHigh, Cancellable: true}, nil

	case "money_transfers":
		flags = append(flags, schema.FlagFxRateReview)
		return schema.Fulfillment{Type: schema.TypeTransfer, Mode: schema.ModeManualOperator, Confidence: schema.ConfHigh, Cancellable: false}, flags

	case "wallets_crypto":
		if reIMO.MatchString(matchText) {
			flags = append(flags, schema.FlagFulfillmentReview)
			return schema.Fulfillment{Type: schema.TypeCredit, Mode: schema.ModeManualOperator, Confidence: schema.ConfLow, Cancellable: true}, flags
		}
		return schema.Fulfillment{Type: schema.TypeTransfer, Mode: schema.ModeManualOperator, Confidence: schema.ConfMedium, Cancellable: false}, nil

	case "telecom":
		if reAlfa.MatchString(matchText) || reTouch.MatchString(matchText) {
			return schema.Fulfillment{Type: schema.TypeCredit, Mode: schema.ModeBridgeDevice, Confidence: schema.ConfHigh, Cancellable: false}, nil
		}
		// Syrian carriers (and anything else): manual, low confidence, FX review.
		flags = append(flags, schema.FlagFulfillmentReview, schema.FlagFxRateReview)
		return schema.Fulfillment{Type: schema.TypeCredit, Mode: schema.ModeManualOperator, Confidence: schema.ConfLow, Cancellable: false}, flags

	case "gsm_tools":
		flags = append(flags, schema.FlagFulfillmentReview)
		return schema.Fulfillment{Type: schema.TypeCredit, Mode: schema.ModeManualOperator, Confidence: schema.ConfLow, Cancellable: false}, flags

	case "software":
		flags = append(flags, schema.FlagFulfillmentReview)
		return schema.Fulfillment{Type: schema.TypeCredit, Mode: schema.ModeManualOperator, Confidence: schema.ConfLow, Cancellable: false}, flags

	case "app_topups", "games":
		f, ff := determineMode(matchText, p)
		return f, ff

	default: // orphan / unknown domain → safest default
		flags = append(flags, schema.FlagFulfillmentReview)
		return schema.Fulfillment{Type: schema.TypeCredit, Mode: schema.ModeManualOperator, Confidence: schema.ConfLow, Cancellable: false}, flags
	}
}

// determineMode classifies account_credit game/app top-ups (spec §6).
func determineMode(matchText string, p *legacy.Product) (schema.Fulfillment, []string) {
	base := schema.Fulfillment{Type: schema.TypeCredit, Cancellable: false}
	switch {
	case p.CheckName != nil:
		prov := p.CheckName.Provider
		base.Mode = schema.ModeAPI
		base.Confidence = schema.ConfHigh
		base.Provider = &prov
		return base, nil
	case reAutomatic.MatchString(matchText):
		base.Mode = schema.ModeAPI
		base.Confidence = schema.ConfMedium
		return base, nil
	case reManual.MatchString(matchText) || reManualEn.MatchString(matchText):
		base.Mode = schema.ModeManualOperator
		base.Confidence = schema.ConfMedium
		return base, nil
	default:
		base.Mode = schema.ModeManualOperator
		base.Confidence = schema.ConfLow
		return base, []string{schema.FlagFulfillmentReview}
	}
}

// ClassifyPricing classifies a product's pricing mode without normalizing prices
// (spec §6); raw mainPrice/base_price are preserved and margin computed. Returns
// the pricing block and any review flags. modeConfidence is high|low per schema.
func ClassifyPricing(domain string, p *legacy.Product) (schema.Pricing, []string) {
	retail := p.MainPrice.Value
	cost := p.BasePrice.Value
	pr := schema.Pricing{
		Retail:   retail,
		Cost:     cost,
		Margin:   round6(retail - cost),
		Currency: "USD",
	}

	switch {
	case p.Amount != nil && p.Amount.Max.Value >= 10000:
		pr.Mode = schema.PriceUnitBalance
		pr.ModeConfidence = schema.ConfLow
		return pr, []string{schema.FlagPricingReview}
	case p.Qty != nil:
		pr.Mode = schema.PriceFixedPackage
		pr.ModeConfidence = schema.ConfHigh
		return pr, nil
	case domain == "giftcards" || domain == "money_transfers":
		pr.Mode = schema.PriceCurrencyValue
		pr.ModeConfidence = schema.ConfHigh
		return pr, nil
	default:
		pr.Mode = schema.PriceCurrencyValue
		pr.ModeConfidence = schema.ConfLow
		return pr, []string{schema.FlagPricingReview}
	}
}
