// Package writer emits the importer's review artifacts: the JSON seeds, the
// aggregated _review.json worklist, and the human-readable run-summary.md.
package writer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AliSleiman0/salehcard/migration/internal/schema"
)

// Result bundles everything the importer produced for a run.
type Result struct {
	Categories []schema.Category
	Products   []schema.Product
	// ProductDomains maps a product legacyId to its resolved rootDomain
	// ("orphan" when its category couldn't be traced to a root).
	ProductDomains map[int]string
	// CategoryFlags maps a category legacyId to its review flags.
	CategoryFlags map[int][]string
	// Notes are run-level remarks (e.g. the 107 rename recommendation).
	Notes []string
}

// ReviewItem is one flagged entity in _review.json.
type ReviewItem struct {
	Kind     string   `json:"kind"` // "product" | "category"
	LegacyID int      `json:"legacyId"`
	Title    string   `json:"title"`
	Domain   string   `json:"rootDomain,omitempty"`
	Flags    []string `json:"flags"`
}

// Review is the aggregated worklist written to _review.json.
type Review struct {
	GeneratedNote   string         `json:"note"`
	TotalProducts   int            `json:"totalProducts"`
	TotalCategories int            `json:"totalCategories"`
	FlagCounts      map[string]int `json:"flagCounts"`
	PerRoot         map[string]int `json:"productsByRootDomain"`
	Notes           []string       `json:"notes"`
	Items           []ReviewItem   `json:"items"`
}

// WriteAll writes categories.json, products.json, _review.json into outDir and
// run-summary.md alongside it (in outDir's parent).
func WriteAll(outDir string, r Result) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "categories.json"), r.Categories); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "products.json"), r.Products); err != nil {
		return err
	}
	review := buildReview(r)
	if err := writeJSON(filepath.Join(outDir, "_review.json"), review); err != nil {
		return err
	}
	return writeSummary(filepath.Join(filepath.Dir(outDir), "run-summary.md"), r, review)
}

func buildReview(r Result) Review {
	flagCounts := map[string]int{}
	perRoot := map[string]int{}
	var items []ReviewItem

	for _, p := range r.Products {
		dom := r.ProductDomains[p.LegacyID]
		if dom == "" {
			dom = "orphan"
		}
		perRoot[dom]++
		for _, f := range p.Flags {
			flagCounts[f]++
		}
		if hasDecisionFlag(p.Flags) {
			items = append(items, ReviewItem{Kind: "product", LegacyID: p.LegacyID, Title: titleOf(p.Title), Domain: dom, Flags: p.Flags})
		}
	}
	for _, c := range r.Categories {
		f := r.CategoryFlags[c.LegacyID]
		for _, fl := range f {
			flagCounts[fl]++
		}
		if hasDecisionFlag(f) {
			items = append(items, ReviewItem{Kind: "category", LegacyID: c.LegacyID, Title: titleOf(c.Name), Domain: c.RootDomain, Flags: f})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		return items[i].LegacyID < items[j].LegacyID
	})

	return Review{
		GeneratedNote:   "Human worklist: every product/category needing a decision and why. Resolve via overrides.json, then re-run.",
		TotalProducts:   len(r.Products),
		TotalCategories: len(r.Categories),
		FlagCounts:      flagCounts,
		PerRoot:         perRoot,
		Notes:           r.Notes,
		Items:           items,
	}
}

func writeSummary(path string, r Result, rv Review) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Catalog import — run summary\n\n")
	fmt.Fprintf(&b, "- Categories: **%d**\n- Products: **%d**\n\n", rv.TotalCategories, rv.TotalProducts)

	fmt.Fprintf(&b, "## Products by root domain\n\n")
	for _, k := range sortedKeys(rv.PerRoot) {
		fmt.Fprintf(&b, "- %s: %d\n", k, rv.PerRoot[k])
	}

	byType, byMode := map[string]int{}, map[string]int{}
	for _, p := range r.Products {
		byType[p.Fulfillment.Type]++
		byMode[p.Fulfillment.Mode]++
	}
	fmt.Fprintf(&b, "\n## Products by fulfillment type\n\n")
	for _, k := range sortedKeys(byType) {
		fmt.Fprintf(&b, "- %s: %d\n", k, byType[k])
	}
	fmt.Fprintf(&b, "\n## Products by fulfillment mode\n\n")
	for _, k := range sortedKeys(byMode) {
		fmt.Fprintf(&b, "- %s: %d\n", k, byMode[k])
	}

	fmt.Fprintf(&b, "\n## Review flags (counts)\n\n")
	if len(rv.FlagCounts) == 0 {
		fmt.Fprintf(&b, "- (none)\n")
	}
	for _, k := range sortedKeys(rv.FlagCounts) {
		fmt.Fprintf(&b, "- %s: %d\n", k, rv.FlagCounts[k])
	}

	fmt.Fprintf(&b, "\n## Top decisions still needed (owner)\n\n")
	for _, n := range r.Notes {
		fmt.Fprintf(&b, "- %s\n", n)
	}

	fmt.Fprintf(&b, "\n## overrides.json\n\nHand-edit `overrides.json` (keyed by legacyId) to force `fulfillment.{type,mode,provider}` / `pricing.mode` and clear flags; the importer applies it last so corrections survive re-runs. Example:\n\n")
	fmt.Fprintf(&b, "```json\n{\n  \"18\": {\n    \"fulfillment\": { \"type\": \"account_credit\", \"mode\": \"api\", \"provider\": 5 },\n    \"clearFlags\": [\"fulfillment_review\"]\n  }\n}\n```\n")

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// bulkFlags are informational (not a per-item decision): they're counted but
// don't by themselves put an item on the review worklist (every product is
// missing tr translation, and markup stripping is automatic).
var bulkFlags = map[string]bool{
	schema.FlagNeedsTranslation: true,
	schema.FlagMarkupStripped:   true,
}

// hasDecisionFlag reports whether flags contain at least one non-bulk flag that
// warrants a human decision.
func hasDecisionFlag(flags []string) bool {
	for _, f := range flags {
		if !bulkFlags[f] {
			return true
		}
	}
	return false
}

func titleOf(s schema.I18n) string {
	if s.En != nil {
		return *s.En
	}
	if s.Ar != nil {
		return *s.Ar
	}
	return ""
}

func sortedKeys(m map[string]int) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
