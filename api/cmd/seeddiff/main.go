// Command seeddiff is a READ-ONLY audit: it compares what `loadseed` WOULD write
// (via the real loadseed.MapProduct mapper) against the current prod documents,
// reporting exactly which catalog-owned fields differ per product. It never
// writes. Use it before a real loadseed run to confirm the only intended change
// (ID input-field labels) is the only change — and to catch any admin edits in
// prod that a load would revert, or products with >1 variant that would collapse.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	"github.com/joho/godotenv"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/migration/loadseed"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

func main() {
	seedDir := flag.String("seed-dir", "../migration/seed", "directory with categories.json + products.json")
	maxDetail := flag.Int("max-detail", 60, "max non-label diff lines to print")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)

	var cats []loadseed.SeedCategory
	if err := readJSON(filepath.Join(*seedDir, "categories.json"), &cats); err != nil {
		log.Fatal(err)
	}
	var prods []loadseed.SeedProduct
	if err := readJSON(filepath.Join(*seedDir, "products.json"), &prods); err != nil {
		log.Fatal(err)
	}
	rootByCat := make(map[int]string, len(cats))
	for _, c := range cats {
		rootByCat[c.LegacyID] = c.RootDomain
	}

	repo := product.NewMongoRepository(db)

	fieldCounts := map[string]int{}
	var missing, multiVariant int
	var nonLabel []string // detail lines for changes other than input-field labels

	for _, sp := range prods {
		in := loadseed.MapProduct(sp, rootByCat[sp.LegacyCategoryID])
		cur, err := repo.FindByLegacyID(ctx, in.LegacyID)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				missing++
				continue
			}
			log.Fatalf("find legacyId %d: %v", in.LegacyID, err)
		}

		if len(cur.Variants) > 1 {
			multiVariant++
			nonLabel = append(nonLabel, fmt.Sprintf("legacyId %d  variants=%d  (load collapses to 1)", in.LegacyID, len(cur.Variants)))
		}

		for _, d := range diffProduct(cur, in) {
			fieldCounts[d.field]++
			if d.field != "inputFields.label" {
				if len(nonLabel) < *maxDetail {
					nonLabel = append(nonLabel, fmt.Sprintf("legacyId %d  %s  prod=%s  seed=%s", in.LegacyID, d.field, d.prod, d.seed))
				}
			}
		}
	}

	fmt.Println("=== PROD vs SEED diff (READ-ONLY, nothing written) ===")
	fmt.Printf("products in seed: %d   found in prod: %d   missing: %d\n", len(prods), len(prods)-missing, missing)
	fmt.Println("\nFields loadseed WOULD overwrite (change counts):")
	keys := make([]string, 0, len(fieldCounts))
	for k := range fieldCounts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		fmt.Println("  (none — prod already identical to seed)")
	}
	for _, k := range keys {
		fmt.Printf("  %-26s %d\n", k, fieldCounts[k])
	}
	fmt.Printf("\nproducts with >1 variant (would collapse): %d\n", multiVariant)

	nonLabelChanges := 0
	for k, v := range fieldCounts {
		if k != "inputFields.label" {
			nonLabelChanges += v
		}
	}
	fmt.Printf("\nNON-label changes (would revert an admin edit): %d\n", nonLabelChanges+multiVariant)
	if len(nonLabel) > 0 {
		fmt.Println("details:")
		for _, l := range nonLabel {
			fmt.Println("  " + l)
		}
	}
	if nonLabelChanges == 0 && multiVariant == 0 {
		fmt.Println("\n✅ SAFE: the ONLY change a load would make is input-field ID labels.")
	} else {
		fmt.Println("\n⚠ REVIEW the above before loading — a load would overwrite those.")
	}
}

type fieldDiff struct{ field, prod, seed string }

// diffProduct returns the catalog-owned fields that differ between the current
// prod document and what loadseed would $set. Insert-only fields (stock, ratings,
// available, createdAt, legacyId, variant _id/resellerPrice) are intentionally
// ignored — the loader preserves them.
func diffProduct(cur *product.Product, in product.UpsertProductInput) []fieldDiff {
	var out []fieldDiff
	add := func(field string, same bool, p, s any) {
		if !same {
			out = append(out, fieldDiff{field, jstr(p), jstr(s)})
		}
	}

	mode := in.FulfillmentMode
	if mode == "" {
		mode = product.DeriveMode(in.FulfillmentType)
	}

	add("title", cur.Title == in.Title, cur.Title, in.Title)
	add("description", cur.Description == in.Description, cur.Description, in.Description)
	add("category", cur.Category == in.Category, cur.Category, in.Category)
	add("categorySlug", cur.CategorySlug == in.CategorySlug, cur.CategorySlug, in.CategorySlug)
	add("rootDomain", cur.RootDomain == in.RootDomain, cur.RootDomain, in.RootDomain)
	add("descriptionHadMarkup", cur.DescriptionHadMarkup == in.DescriptionHadMarkup, cur.DescriptionHadMarkup, in.DescriptionHadMarkup)
	add("images", eqStrings(cur.Images, in.Images), cur.Images, in.Images)
	add("legacyCategoryId", eqIntPtr(cur.LegacyCategoryID, in.LegacyCategoryID), cur.LegacyCategoryID, in.LegacyCategoryID)
	add("fulfillmentType", cur.FulfillmentType == in.FulfillmentType, cur.FulfillmentType, in.FulfillmentType)
	add("fulfillmentMode", cur.FulfillmentMode == mode, cur.FulfillmentMode, mode)
	add("fulfillmentProvider", eqIntPtr(cur.FulfillmentProvider, in.FulfillmentProvider), cur.FulfillmentProvider, in.FulfillmentProvider)
	add("fulfillmentConfidence", cur.FulfillmentConfidence == in.FulfillmentConfidence, cur.FulfillmentConfidence, in.FulfillmentConfidence)
	add("fulfillmentCancellable", eqBoolPtr(cur.FulfillmentCancellable, in.FulfillmentCancellable), cur.FulfillmentCancellable, in.FulfillmentCancellable)
	add("pricing", reflect.DeepEqual(cur.Pricing, in.Pricing), cur.Pricing, in.Pricing)
	add("amountConstraints", reflect.DeepEqual(cur.AmountConstraints, in.AmountConstraints), cur.AmountConstraints, in.AmountConstraints)
	add("verification", reflect.DeepEqual(cur.Verification, in.Verification), cur.Verification, in.Verification)
	add("status", cur.Status == in.Status, cur.Status, in.Status)
	add("sortOrder", cur.SortOrder == in.SortOrder, cur.SortOrder, in.SortOrder)
	add("flags", eqStrings(cur.Flags, in.Flags), cur.Flags, in.Flags)

	// variant[0]: the loader overlays price+denomination onto the existing element.
	if len(in.Variants) > 0 {
		var cp float64
		var cd string
		if len(cur.Variants) > 0 {
			cp, cd = cur.Variants[0].Price, cur.Variants[0].Denomination
		}
		add("variant0.price", cp == in.Variants[0].Price, cp, in.Variants[0].Price)
		add("variant0.denomination", cd == in.Variants[0].Denomination, cd, in.Variants[0].Denomination)
	}

	// input fields: classify as label-only vs structural.
	if !eqInputFields(cur.InputFields, in.InputFields) {
		if labelOnlyChange(cur.InputFields, in.InputFields) {
			out = append(out, fieldDiff{"inputFields.label", "", ""})
		} else {
			out = append(out, fieldDiff{"inputFields.structural", jstr(cur.InputFields), jstr(in.InputFields)})
		}
	}
	return out
}

// labelOnlyChange reports whether a and b differ ONLY in their labels (same
// length, and every field equal except Label).
func labelOnlyChange(a, b []product.InputField) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		x.Label, y.Label = product.I18nLabel{}, product.I18nLabel{}
		if !reflect.DeepEqual(x, y) {
			return false
		}
	}
	return true
}

func eqInputFields(a, b []product.InputField) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func eqStrings(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func eqIntPtr(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func eqBoolPtr(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func jstr(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func readJSON(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
