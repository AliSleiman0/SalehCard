package loadseed

import (
	"fmt"

	"github.com/AliSleiman0/salehcard/api/internal/modules/category"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

// MapCategory converts a seed category into a category upsert input.
func MapCategory(sc SeedCategory) category.UpsertCategoryInput {
	return category.UpsertCategoryInput{
		LegacyID:       sc.LegacyID,
		ParentLegacyID: sc.ParentLegacyID,
		Slug:           sc.Slug,
		Name:           category.I18nString{En: deref(sc.Name.En), Ar: deref(sc.Name.Ar), Tr: deref(sc.Name.Tr)},
		Image:          sc.Image,
		SortOrder:      sc.SortOrder,
		RootDomain:     sc.RootDomain,
		Depth:          sc.Depth,
		Visible:        sc.Visible,
	}
}

// MapProduct converts a seed product into a product upsert input, synthesizing a
// single Variant from pricing.retail so the order engine and storefront (which
// are variant-based) keep working unchanged. rootDomain is resolved by the
// caller from the product's category (seed products don't carry it); it is
// denormalized onto the product so the storefront can browse a whole domain.
func MapProduct(sp SeedProduct, rootDomain string) product.UpsertProductInput {
	var amount *product.AmountConstraints
	if sp.AmountConstraints != nil {
		amount = &product.AmountConstraints{Min: sp.AmountConstraints.Min, Max: sp.AmountConstraints.Max}
	}
	var verification *product.Verification
	if sp.Verification != nil {
		verification = &product.Verification{Provider: sp.Verification.Provider, App: sp.Verification.App}
	}
	cancellable := sp.Fulfillment.Cancellable

	return product.UpsertProductInput{
		LegacyID:               sp.LegacyID,
		LegacyCategoryID:       intPtr(sp.LegacyCategoryID),
		CategorySlug:           sp.CategorySlug,
		RootDomain:             rootDomain,
		Title:                  i18n(sp.Title),
		Description:            i18n(sp.Description),
		DescriptionHadMarkup:   sp.DescriptionHadMarkup,
		Category:               sp.CategorySlug, // flat category string = slug (web/admin filter on this)
		Images:                 sp.Images,
		Variants:               []product.Variant{synthVariant(sp)},
		FulfillmentType:        product.FulfillmentType(sp.Fulfillment.Type),
		FulfillmentMode:        product.FulfillmentMode(sp.Fulfillment.Mode),
		FulfillmentProvider:    sp.Fulfillment.Provider,
		FulfillmentConfidence:  product.Confidence(sp.Fulfillment.Confidence),
		FulfillmentCancellable: &cancellable,
		Pricing: &product.Pricing{
			Retail: sp.Pricing.Retail, Cost: sp.Pricing.Cost, Margin: sp.Pricing.Margin,
			Currency: sp.Pricing.Currency, Mode: product.PricingMode(sp.Pricing.Mode),
			ModeConfidence: product.Confidence(sp.Pricing.ModeConfidence),
		},
		AmountConstraints: amount,
		InputFields:       mapInputFields(sp.InputFields),
		Verification:      verification,
		Available:         sp.Status == "active",
		Status:            sp.Status,
		SortOrder:         sp.SortOrder,
		Flags:             sp.Flags,
	}
}

// synthVariant builds the single denomination representing a legacy SKU.
func synthVariant(sp SeedProduct) product.Variant {
	label := "Default"
	if sp.AmountConstraints != nil {
		label = fmt.Sprintf("%g–%g", sp.AmountConstraints.Min, sp.AmountConstraints.Max)
	}
	return product.Variant{Denomination: label, Price: sp.Pricing.Retail}
}

func mapInputFields(in []SeedInputField) []product.InputField {
	if len(in) == 0 {
		return nil
	}
	out := make([]product.InputField, len(in))
	for i, f := range in {
		var c *product.InputFieldConstraints
		if f.Constraints != nil {
			c = &product.InputFieldConstraints{Min: f.Constraints.Min, Max: f.Constraints.Max, Options: f.Constraints.Options}
		}
		out[i] = product.InputField{
			Key:         f.Key,
			Label:       product.I18nLabel{En: deref(f.Label.En), Ar: deref(f.Label.Ar)},
			LegacyName:  f.LegacyName,
			Type:        product.InputFieldType(f.Type),
			Constraints: c,
			Sensitive:   f.Sensitive,
		}
	}
	return out
}

func i18n(s SeedI18n) product.I18nString {
	return product.I18nString{En: deref(s.En), Ar: deref(s.Ar), Tr: deref(s.Tr)}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intPtr(v int) *int { return &v }
