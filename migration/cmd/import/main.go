// Command import fetches the legacy SalehCard catalog, transforms it into the
// target seed schema by applying the migration business rules, and writes JSON
// seeds + a review worklist + a run summary. Re-runnable: cached raw responses
// under _raw/ make re-runs offline (pass --refresh to re-fetch).
package main

import (
	"flag"
	"log"
	"sort"
	"time"

	"github.com/AliSleiman0/salehcard/migration/internal/legacy"
	"github.com/AliSleiman0/salehcard/migration/internal/schema"
	"github.com/AliSleiman0/salehcard/migration/internal/transform"
	"github.com/AliSleiman0/salehcard/migration/internal/writer"
)

func main() {
	var (
		refresh   = flag.Bool("refresh", false, "bypass the _raw cache and re-fetch from the API")
		root      = flag.Int("root", 0, "limit to a single root category id (0 = all 8 roots)")
		outDir    = flag.String("out", "./seed", "directory for the JSON seed output")
		rawDir    = flag.String("raw", "./_raw", "directory for the cached raw API responses")
		delayMS   = flag.Int("delay", 150, "politeness delay between network fetches (ms)")
		overrides = flag.String("overrides", "./overrides.json", "hand-edited overrides applied last")
	)
	flag.Parse()

	if err := run(*refresh, *root, *outDir, *rawDir, *overrides, time.Duration(*delayMS)*time.Millisecond); err != nil {
		log.Fatalf("import failed: %v", err)
	}
}

// catAgg accumulates a category's en+ar metadata across the two-language walk.
type catAgg struct {
	id, parentID, sort int
	nameEn, nameAr     string
	image              string
	visible            bool
}

func run(refresh bool, root int, outDir, rawDir, overridesPath string, delay time.Duration) error {
	client := legacy.NewClient(rawDir, delay, refresh)

	roots := transform.RootIDs
	if root != 0 {
		roots = []int{root}
	}

	catByID := map[int]*catAgg{}
	prodEn := map[int]*legacy.Product{}
	prodAr := map[int]*legacy.Product{}
	tree := transform.NewTree()

	absorb := func(fetched []legacy.Fetched, lang string) {
		for _, f := range fetched {
			tree.AddEdge(f.ID, f.ParentID)
			c := catByID[f.ID]
			if c == nil {
				c = &catAgg{id: f.ID, parentID: f.ParentID, visible: true}
				catByID[f.ID] = c
			}
			if f.ParentID != 0 {
				c.parentID = f.ParentID
			}
			if f.Meta != nil {
				c.sort = f.Meta.Sort
				c.image = f.Meta.Photo
				c.visible = f.Meta.Visible == 1
			}
			if lang == "en" {
				c.nameEn = f.Name
			} else {
				c.nameAr = f.Name
			}
			for i := range f.Data.Products {
				p := &f.Data.Products[i]
				if lang == "en" {
					prodEn[p.ID] = p
				} else {
					prodAr[p.ID] = p
				}
			}
		}
	}

	for _, r := range roots {
		for _, lang := range []string{"en", "ar"} {
			fetched, err := client.Walk(r, lang)
			if err != nil {
				return err
			}
			absorb(fetched, lang)
			log.Printf("walked root %d (%s): %d categories", r, lang, len(fetched))
		}
	}

	// Transform categories; build a catID -> slug map for product references.
	slugByCat := map[int]string{}
	catFlags := map[int][]string{}
	categories := make([]schema.Category, 0, len(catByID))
	for _, id := range sortedKeys(catByID) {
		c := catByID[id]
		_, domain, _ := tree.Root(id)
		depth := tree.Depth(id)
		if depth < 0 {
			depth = 0
		}
		sc, flags := transform.TransformCategory(transform.MergedCategory{
			ID: c.id, ParentID: c.parentID, Depth: depth, Sort: c.sort,
			NameEn: c.nameEn, NameAr: c.nameAr, Image: c.image, Visible: c.visible, Domain: domain,
		})
		categories = append(categories, sc)
		slugByCat[id] = sc.Slug
		if len(flags) > 0 {
			catFlags[id] = flags
		}
	}

	// Transform products.
	products := make([]schema.Product, 0, len(prodEn))
	productDomains := map[int]string{}
	for _, id := range sortedProdIDs(prodEn, prodAr) {
		en, ar := prodEn[id], prodAr[id]
		if en == nil {
			en = ar // fall back to the ar record as primary when en is absent
		}
		_, domain, ok := tree.Root(en.CategoryID)
		if !ok {
			domain = ""
		}
		products = append(products, transform.TransformProduct(en, ar, domain, slugByCat[en.CategoryID]))
		if domain == "" {
			productDomains[id] = "orphan"
		} else {
			productDomains[id] = domain
		}
	}

	if touched := transform.ApplyOverrides(products, mustLoadOverrides(overridesPath)); touched > 0 {
		log.Printf("applied overrides to %d product(s)", touched)
	}

	notes := []string{
		"Category 107 \"Cryptocurrency Section\" is mislabeled — recommend splitting/renaming into Crypto and E-Wallets, and moving IMO under app top-ups. Emitted with rootDomain=wallets_crypto; do not silently rename.",
		"Provider IDs behind check_name (e.g. provider 5 = PUBG verifier) need owner confirmation before real api adapters are wired.",
		"Money-transfer and Syrian-telecom pricing carries embedded/stale FX (flag fx_rate_review) — owner decision: live FX vs manual updates.",
		"Products flagged fulfillment_review default to manual_operator (safer) — confirm or set fulfillment.mode in overrides.json.",
	}

	res := writer.Result{
		Categories: categories, Products: products,
		ProductDomains: productDomains, CategoryFlags: catFlags, Notes: notes,
	}
	if err := writer.WriteAll(outDir, res); err != nil {
		return err
	}
	log.Printf("wrote %d categories, %d products → %s", len(categories), len(products), outDir)
	return nil
}

func mustLoadOverrides(path string) transform.Overrides {
	ov, err := transform.LoadOverrides(path)
	if err != nil {
		log.Printf("warning: could not load overrides (%v); continuing without", err)
		return transform.Overrides{}
	}
	return ov
}

func sortedKeys(m map[int]*catAgg) []int {
	ids := make([]int, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func sortedProdIDs(a, b map[int]*legacy.Product) []int {
	seen := map[int]bool{}
	var ids []int
	for id := range a {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for id := range b {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}
