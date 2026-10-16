// Command unitpricing switches the migrated live-app recharge products (Up Live,
// Bigo, Poppo, Soul Star…) to per-unit pricing. The old site sold them as
// "unit price × amount typed", but the catalog import kept only a placeholder
// flat price, so today the typed amount never affects what the customer pays.
//
// For every eligible product it sets unitPricing {fieldKey, blockUnits} and
// prices the single variant at cost × blockUnits × (1 + markup%), where
// blockUnits is the smallest power of ten whose cost reaches $1 (so the admin
// sees "$1.77 per 100" rather than "$0.0177 per 1").
//
// It is a DRY RUN unless --apply is passed, writes a CSV report of every
// candidate (changed and skipped, with the reason), and refuses to write to
// prod without --confirm-prod. Products already carrying unitPricing are never
// touched, so a re-run is safe; nor are products whose unitPricing an admin
// explicitly cleared (stored null = "keep fixed packages").
//
// Converting a product also removes every reseller's custom price on its
// variant (reseller_prices): it was a flat-package price and would otherwise
// be read as a per-block price and undercharge. The CSV reports how many
// were (or, in a dry run, would be) removed.
//
//	MONGO_URI=... go run ./cmd/unitpricing --markup 10 --report unitpricing.csv
//	MONGO_URI=... go run ./cmd/unitpricing --markup 10 --report unitpricing.csv --apply [--confirm-prod]
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/reseller"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

// change is the planned conversion of one product.
type change struct {
	FieldKey   string
	Min, Max   float64
	BlockUnits float64
	Price      float64
	Label      string
}

// plan decides whether p converts to per-unit pricing and how; a non-empty
// reason means it is skipped. optedOut is true when the stored document has
// unitPricing: null (an admin switched it back to fixed packages — decoding
// loses the null vs absent distinction, so the caller passes it). Pure —
// unit-tested without a database.
func plan(p product.Product, optedOut bool, markupPct float64) (change, string) {
	if p.UnitPricing != nil {
		return change{}, "already per-unit"
	}
	if optedOut {
		return change{}, "admin opted out (unitPricing cleared to fixed packages)"
	}
	if p.FulfillmentType != product.FulfillmentCredit {
		return change{}, "not account_credit"
	}
	mode := p.FulfillmentMode
	if mode == "" {
		mode = product.DeriveMode(p.FulfillmentType)
	}
	if mode != product.FulfillmentModeManualOperator {
		return change{}, "not manually fulfilled (" + string(mode) + ")"
	}
	if len(p.Variants) != 1 {
		return change{}, fmt.Sprintf("%d variants (needs exactly 1)", len(p.Variants))
	}
	var amount *product.InputField
	for i := range p.InputFields {
		f := &p.InputFields[i]
		switch f.Type {
		case product.InputFieldQuantity:
			return change{}, "has a quantity field"
		case product.InputFieldAmount:
			if amount != nil {
				return change{}, "more than one amount field"
			}
			amount = f
		}
	}
	if amount == nil {
		return change{}, "no amount field"
	}
	c := amount.Constraints
	if c == nil || c.Min == nil || c.Max == nil || *c.Min <= 0 || *c.Max < *c.Min {
		return change{}, "amount field lacks a usable min/max"
	}
	if p.Pricing == nil {
		return change{}, "no pricing data"
	}
	// Only the old site's per-unit products: the cost is per unit only in
	// unit_balance mode, and the markup math assumes a USD cost.
	if p.Pricing.Mode != product.PricingModeUnitBalance {
		return change{}, "pricing mode " + modeOf(p) + " (needs unit_balance)"
	}
	if cur := strings.TrimSpace(p.Pricing.Currency); cur != "" && !strings.EqualFold(cur, "USD") {
		return change{}, "cost currency " + cur + " (needs USD)"
	}
	if p.Pricing.Cost <= 0 {
		return change{}, "no per-unit cost"
	}
	// Coins/diamonds cost fractions of a cent; a unit costing $1+ is a pack
	// product whose "amount" is a pack count (TAGS 500, WE PLAY GOLD…) — those
	// need a human decision, not an automatic per-unit conversion.
	if p.Pricing.Cost >= 1 {
		return change{}, "cost ≥ $1 per unit — looks like a pack product, review manually"
	}
	block := blockFor(p.Pricing.Cost)
	price := product.RoundCents(p.Pricing.Cost * block * (1 + markupPct/100))
	if price <= 0 {
		return change{}, "computed price rounds to 0"
	}
	return change{
		FieldKey:   amount.Key,
		Min:        *c.Min,
		Max:        *c.Max,
		BlockUnits: block,
		Price:      price,
		Label:      "per " + group(block),
	}, ""
}

// blockFor is the smallest power of ten (≥ 1) whose total cost reaches $1.
func blockFor(unitCost float64) float64 {
	block := 1.0
	for unitCost*block < 1 && block < 1e12 {
		block *= 10
	}
	return block
}

// group renders an integral count with thousands separators ("10,000").
func group(v float64) string {
	s := strconv.FormatFloat(v, 'f', 0, 64)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isProdURI(uri string) bool {
	return strings.Contains(uri, "cosmos.azure.com") || strings.Contains(uri, "mongocluster")
}

func main() {
	markup := flag.Float64("markup", -1, "markup percent over per-unit cost (required, e.g. 10)")
	apply := flag.Bool("apply", false, "write the changes (default is a dry run)")
	confirmProd := flag.Bool("confirm-prod", false, "required together with --apply against the prod cluster")
	report := flag.String("report", "unitpricing-report.csv", "CSV report path")
	category := flag.String("category", "", "only products in this flat category slug (e.g. charging-application)")
	flag.Parse()
	if *markup < 0 {
		log.Fatal("--markup is required (percent over cost, e.g. --markup 10)")
	}

	_ = godotenv.Load()
	cfg := config.Load()
	prod := isProdURI(cfg.MongoURI)
	if *apply && prod && !*confirmProd {
		log.Fatal("refusing to write to PROD without --confirm-prod (review a dry-run report first)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)
	env := "local"
	if prod {
		env = "PROD"
	}
	log.Printf("unitpricing: db=%s (%s) markup=%g%% apply=%v", cfg.DBName, env, *markup, *apply)

	col := db.Collection("products")
	filter := bson.D{{Key: "inputFields.type", Value: "amount"}}
	if *category != "" {
		filter = append(filter, bson.E{Key: "category", Value: *category})
	}
	cur, err := col.Find(ctx, filter)
	if err != nil {
		log.Fatalf("find: %v", err)
	}
	var raws []bson.Raw
	if err := cur.All(ctx, &raws); err != nil {
		log.Fatalf("decode: %v", err)
	}
	prods := make([]product.Product, len(raws))
	optedOut := make([]bool, len(raws))
	for i, raw := range raws {
		if err := bson.Unmarshal(raw, &prods[i]); err != nil {
			log.Fatalf("decode: %v", err)
		}
		optedOut[i] = unitPricingIsNull(raw)
	}
	prices := reseller.NewMongoRepository(db)

	f, err := os.Create(*report)
	if err != nil {
		log.Fatalf("report: %v", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"id", "title", "category", "available", "pricing_mode", "cost_per_unit", "old_price", "old_label",
		"min", "max", "block_units", "new_price_per_block", "price_at_min", "price_at_max",
		"reseller_custom_prices_removed", "action"})

	var changed, skipped, failed int
	var pricesRemoved int64
	for i, p := range prods {
		ch, reason := plan(p, optedOut[i], *markup)
		row := []string{p.ID.Hex(), p.Title.En, p.Category, strconv.FormatBool(p.Available), modeOf(p), cost(p), "", ""}
		if len(p.Variants) > 0 {
			row[6] = strconv.FormatFloat(p.Variants[0].Price, 'f', -1, 64)
			row[7] = p.Variants[0].Denomination
		}
		if reason != "" {
			skipped++
			_ = w.Write(append(row, "", "", "", "", "", "", "", "skip: "+reason))
			continue
		}
		row = append(row,
			num(ch.Min), num(ch.Max), num(ch.BlockUnits), num(ch.Price),
			num(product.UnitLinePrice(ch.Price, ch.Min, ch.BlockUnits)),
			num(product.UnitLinePrice(ch.Price, ch.Max, ch.BlockUnits)))
		ids := variantIDs(p)
		if !*apply {
			n, err := prices.CountPricesForVariants(ctx, ids)
			if err != nil {
				failed++
				_ = w.Write(append(row, "", "FAILED: count reseller prices: "+err.Error()))
				continue
			}
			pricesRemoved += n
			changed++
			_ = w.Write(append(row, strconv.FormatInt(n, 10), "would convert"))
			continue
		}
		// Drop the stale flat-price reseller overrides FIRST: if that fails the
		// product stays on fixed pricing, where they are still correct.
		n, err := prices.DeletePricesForVariants(ctx, ids)
		if err != nil {
			failed++
			_ = w.Write(append(row, "", "FAILED: remove reseller prices (not converted): "+err.Error()))
			continue
		}
		pricesRemoved += n
		if err := applyChange(ctx, col, p, ch); err != nil {
			failed++
			_ = w.Write(append(row, strconv.FormatInt(n, 10), "FAILED: "+err.Error()))
			continue
		}
		changed++
		_ = w.Write(append(row, strconv.FormatInt(n, 10), "converted"))
	}
	verb, removed := "would convert", "would remove"
	if *apply {
		verb, removed = "converted", "removed"
	}
	log.Printf("unitpricing: %d candidates — %s %d, skipped %d, failed %d; %s %d reseller custom price(s) — report: %s",
		len(prods), verb, changed, skipped, failed, removed, pricesRemoved, *report)
}

// unitPricingIsNull reports whether the stored document carries
// unitPricing: null — an admin's explicit "back to fixed packages".
func unitPricingIsNull(raw bson.Raw) bool {
	v, err := raw.LookupErr("unitPricing")
	return err == nil && v.Type == bson.TypeNull
}

// variantIDs returns the hex ids of p's variants (reseller_prices keys).
func variantIDs(p product.Product) []string {
	out := make([]string, 0, len(p.Variants))
	for _, v := range p.Variants {
		if !v.ID.IsZero() {
			out = append(out, v.ID.Hex())
		}
	}
	return out
}

func modeOf(p product.Product) string {
	if p.Pricing == nil || p.Pricing.Mode == "" {
		return "(none)"
	}
	return string(p.Pricing.Mode)
}

// applyChange converts one product, guarded on unitPricing still being absent
// so a concurrent admin edit (or a re-run) is never overwritten. A stale
// reseller price on the variant would now mean "per block" and is dropped.
func applyChange(ctx context.Context, col *mongo.Collection, p product.Product, ch change) error {
	res, err := col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: p.ID}, {Key: "unitPricing", Value: bson.D{{Key: "$exists", Value: false}}}},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "unitPricing", Value: product.UnitPricing{FieldKey: ch.FieldKey, BlockUnits: ch.BlockUnits}},
				{Key: "variants.0.price", Value: ch.Price},
				{Key: "variants.0.denomination", Value: ch.Label},
				{Key: "updatedAt", Value: time.Now().UTC()},
			}},
			{Key: "$unset", Value: bson.D{{Key: "variants.0.resellerPrice", Value: ""}}},
		})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("changed concurrently (unitPricing already set)")
	}
	return nil
}

func cost(p product.Product) string {
	if p.Pricing == nil {
		return ""
	}
	return strconv.FormatFloat(p.Pricing.Cost, 'f', -1, 64)
}

func num(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
