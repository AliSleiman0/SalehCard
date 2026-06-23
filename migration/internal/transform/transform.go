package transform

import (
	"strings"

	"github.com/AliSleiman0/salehcard/migration/internal/legacy"
	"github.com/AliSleiman0/salehcard/migration/internal/schema"
)

// MergedCategory carries a category's en+ar metadata plus its resolved place in
// the tree, ready to transform.
type MergedCategory struct {
	ID       int
	ParentID int
	Depth    int
	Sort     int
	NameEn   string
	NameAr   string
	Image    string
	Visible  bool
	Domain   string
}

// TransformCategory builds a target Category and its review flags.
func TransformCategory(c MergedCategory) (schema.Category, []string) {
	var parent *int
	if c.ParentID != 0 {
		p := c.ParentID
		parent = &p
	}
	name := i18n(c.NameEn, c.NameAr)
	var flags []string
	if needsTranslation(name) {
		flags = append(flags, schema.FlagNeedsTranslation)
	}
	return schema.Category{
		LegacyID:       c.ID,
		ParentLegacyID: parent,
		Slug:           slugWithFallback(c.NameEn, "cat", c.ID),
		Name:           name,
		Image:          imageURL(c.Image),
		SortOrder:      c.Sort,
		RootDomain:     c.Domain,
		Depth:          c.Depth,
		Visible:        c.Visible,
	}, flags
}

// TransformProduct builds a target Product from its en (primary) and ar legacy
// records, applying the §5/§6 classification. domain is the resolved root domain
// ("" when the product's category couldn't be traced to a root → orphan).
func TransformProduct(en, ar *legacy.Product, domain, categorySlug string) schema.Product {
	nameAr, catAr, descArRaw := "", "", ""
	if ar != nil {
		nameAr, catAr, descArRaw = ar.ProductName, ar.CategoryName, ar.Description
	}
	matchText := strings.ToLower(en.ProductName+" "+en.CategoryName) + " " + nameAr + " " + catAr

	title := i18n(en.ProductName, nameAr)
	descEn, hadEn := stripHTML(en.Description)
	descAr, hadAr := stripHTML(descArRaw)
	desc := i18n(descEn, descAr)
	hadMarkup := hadEn || hadAr

	pricing, pflags := ClassifyPricing(domain, en)
	fulfillment, fflags := ClassifyFulfillment(domain, matchText, en)
	amountC, aflags := BuildAmountConstraints(en)

	var arReqs []legacy.Require
	if ar != nil {
		arReqs = ar.Requires
	}
	inputFields, sensitive := BuildInputFields(en.Requires, arReqs)

	var flags []string
	flags = append(flags, pflags...)
	flags = append(flags, fflags...)
	flags = append(flags, aflags...)
	if sensitive {
		flags = append(flags, schema.FlagSensitiveInput)
	}
	if hadMarkup {
		flags = append(flags, schema.FlagMarkupStripped)
	}
	if needsTranslation(title) {
		flags = append(flags, schema.FlagNeedsTranslation)
	}
	if domain == "" {
		flags = append(flags, schema.FlagOrphanCategory)
	}

	var images []string
	if u := imageURL(en.Photo); u != "" {
		images = append(images, u)
	}

	status := "active"
	if !en.Available {
		status = "unavailable"
	}

	var verification *schema.Verification
	if en.CheckName != nil {
		verification = &schema.Verification{Provider: en.CheckName.Provider, App: en.CheckName.App}
	}

	return schema.Product{
		LegacyID:             en.ID,
		LegacyCategoryID:     en.CategoryID,
		CategorySlug:         categorySlug,
		Title:                title,
		Description:          desc,
		DescriptionHadMarkup: hadMarkup,
		Images:               images,
		Pricing:              pricing,
		AmountConstraints:    amountC,
		InputFields:          inputFields,
		Verification:         verification,
		Fulfillment:          fulfillment,
		Status:               status,
		SortOrder:            en.Sort,
		Flags:                dedupe(flags),
	}
}

// imageURL absolutizes a legacy image path against the API host.
func imageURL(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http") {
		return path
	}
	return legacy.DefaultBaseURL + "/" + strings.TrimPrefix(path, "/")
}

// dedupe returns flags with duplicates removed, order preserved.
func dedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
