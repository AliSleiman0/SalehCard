// Command inputlabels inspects (and optionally fixes) English text that is
// actually Arabic on product input-field labels and category names.
//
// Root cause: a legacy import stored the Arabic string directly in the English
// slot (label.en / name.en contain Arabic, ar usually mirrors it). The app's
// I18n resolve('en') returns that en value verbatim, so an English-locale user
// sees Arabic. There is NO English source to fall back to — it must be authored.
//
// Fix model: a translation map keyed by the EXACT Arabic source string ->
// English. For every field/category whose en contains Arabic script, if its en
// string is in the map, en is overwritten with the English (ar is left as-is).
// Nothing is guessed: strings not in the map are reported and left untouched.
//
// Usage:
//
//	go run ./cmd/inputlabels                     # summary counts (read-only)
//	go run ./cmd/inputlabels -distinct           # list unique Arabic strings to translate
//	go run ./cmd/inputlabels -tr tr.json         # dry-run: show en -> English it would set
//	go run ./cmd/inputlabels -tr tr.json -apply  # write the translations
//
// tr.json shape: { "<exact Arabic string>": "<English>", ... }
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

type i18n struct {
	En string `bson:"en"`
	Ar string `bson:"ar"`
	Tr string `bson:"tr"`
}

// label carries only en/ar (I18nLabel), both omitempty in the model.
type label struct {
	En string `bson:"en,omitempty"`
	Ar string `bson:"ar,omitempty"`
}

// inputField mirrors product.InputField; Rest preserves every other field on
// write-back (type/constraints/sensitive/legacyName/...).
type inputField struct {
	Key   string `bson:"key"`
	Label label  `bson:"label"`
	Rest  bson.M `bson:",inline"`
}

type labelDoc struct {
	ID          bson.ObjectID `bson:"_id"`
	Category    string        `bson:"category"`
	Title       i18n          `bson:"title"`
	InputFields []inputField  `bson:"inputFields"`
}

func main() {
	trPath := flag.String("tr", "", "path to JSON {arabicString: english} translation map")
	apply := flag.Bool("apply", false, "write changes (default is dry-run)")
	distinct := flag.Bool("distinct", false, "list unique Arabic-in-en strings to translate")
	flag.Parse()

	tr := loadMap(*trPath)
	if *apply && len(tr) == 0 {
		log.Fatal("-apply requires -tr with at least one entry")
	}

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)

	if *distinct {
		runDistinct(ctx, db)
		return
	}

	catStats := processCategories(ctx, db, tr, *apply)
	fieldStats := processProducts(ctx, db, tr, *apply)

	fmt.Printf("\n========== SUMMARY ==========\n")
	fmt.Printf("categories arabic-in-en: %d  (translated: %d, unmapped: %d)\n",
		catStats.total, catStats.mapped, catStats.unmapped)
	fmt.Printf("input fields arabic-in-en: %d  (translated: %d, unmapped: %d)\n",
		fieldStats.total, fieldStats.mapped, fieldStats.unmapped)
	if !*apply && len(tr) > 0 {
		fmt.Printf("DRY-RUN — re-run with -apply to write.\n")
	}
}

type stats struct{ total, mapped, unmapped int }

// processCategories fixes categories whose name.en contains Arabic.
func processCategories(ctx context.Context, db *mongo.Database, tr map[string]string, apply bool) stats {
	fmt.Printf("========== CATEGORIES ==========\n")
	col := db.Collection("categories")
	cur, err := col.Find(ctx, bson.D{},
		options.Find().SetSort(bson.D{{Key: "depth", Value: 1}, {Key: "sortOrder", Value: 1}}))
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)

	var s stats
	for cur.Next(ctx) {
		var c struct {
			ID     bson.ObjectID `bson:"_id"`
			Slug   string        `bson:"slug"`
			Depth  int           `bson:"depth"`
			Legacy int           `bson:"legacyId"`
			Name   i18n          `bson:"name"`
		}
		if err := cur.Decode(&c); err != nil {
			log.Fatal(err)
		}
		if !hasArabic(c.Name.En) {
			continue
		}
		s.total++
		en, ok := tr[c.Name.En]
		if !ok || en == "" {
			s.unmapped++
			fmt.Printf("• slug=%-16s depth=%d  en=%q  (UNMAPPED)\n", c.Slug, c.Depth, c.Name.En)
			continue
		}
		s.mapped++
		fmt.Printf("• slug=%-16s depth=%d  en=%q -> %q\n", c.Slug, c.Depth, c.Name.En, en)
		if apply {
			_, err := col.UpdateByID(ctx, c.ID, bson.D{{Key: "$set", Value: bson.D{
				{Key: "name.en", Value: en},
				{Key: "updatedAt", Value: time.Now().UTC()},
			}}})
			if err != nil {
				log.Fatalf("update category %s: %v", c.Slug, err)
			}
		}
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	return s
}

// processProducts fixes input-field labels whose label.en contains Arabic.
func processProducts(ctx context.Context, db *mongo.Database, tr map[string]string, apply bool) stats {
	fmt.Printf("\n========== PRODUCT INPUT FIELDS ==========\n")
	col := db.Collection("products")
	cur, err := col.Find(ctx, bson.D{{Key: "inputFields.0", Value: bson.D{{Key: "$exists", Value: true}}}})
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)

	var s stats
	var updatedDocs int
	unmappedSet := map[string]int{}
	for cur.Next(ctx) {
		var d labelDoc
		if err := cur.Decode(&d); err != nil {
			log.Fatal(err)
		}
		changed := false
		for i := range d.InputFields {
			f := &d.InputFields[i]
			if !hasArabic(f.Label.En) {
				continue
			}
			s.total++
			en, ok := tr[f.Label.En]
			if !ok || en == "" {
				s.unmapped++
				unmappedSet[f.Label.En]++
				continue
			}
			s.mapped++
			if apply {
				f.Label.En = en
				changed = true
			}
		}
		if apply && changed {
			_, err := col.UpdateByID(ctx, d.ID, bson.D{{Key: "$set", Value: bson.D{
				{Key: "inputFields", Value: d.InputFields},
				{Key: "updatedAt", Value: time.Now().UTC()},
			}}})
			if err != nil {
				log.Fatalf("update product %s: %v", d.ID.Hex(), err)
			}
			updatedDocs++
		}
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	if len(unmappedSet) > 0 {
		fmt.Printf("UNMAPPED input-field strings (left untouched):\n")
		for _, k := range sortedKeys(unmappedSet) {
			fmt.Printf("  %q ×%d\n", k, unmappedSet[k])
		}
	}
	if apply {
		fmt.Printf("updated product docs: %d\n", updatedDocs)
	}
	return s
}

// runDistinct prints every unique Arabic-in-en string across both collections
// with its occurrence count — the exact keys to put in the translation map.
func runDistinct(ctx context.Context, db *mongo.Database) {
	counts := map[string]int{}
	origin := map[string]map[string]bool{} // string -> set of contexts

	add := func(s, ctxLabel string) {
		if !hasArabic(s) {
			return
		}
		counts[s]++
		if origin[s] == nil {
			origin[s] = map[string]bool{}
		}
		origin[s][ctxLabel] = true
	}

	ccur, err := db.Collection("categories").Find(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	for ccur.Next(ctx) {
		var c struct {
			Slug string `bson:"slug"`
			Name i18n   `bson:"name"`
		}
		if err := ccur.Decode(&c); err != nil {
			log.Fatal(err)
		}
		add(c.Name.En, "category:"+c.Slug)
	}
	ccur.Close(ctx)

	pcur, err := db.Collection("products").Find(ctx,
		bson.D{{Key: "inputFields.0", Value: bson.D{{Key: "$exists", Value: true}}}})
	if err != nil {
		log.Fatal(err)
	}
	for pcur.Next(ctx) {
		var d labelDoc
		if err := pcur.Decode(&d); err != nil {
			log.Fatal(err)
		}
		for _, f := range d.InputFields {
			add(f.Label.En, "field:"+f.Key)
		}
	}
	pcur.Close(ctx)

	fmt.Printf("=== %d distinct Arabic-in-en strings ===\n", len(counts))
	fmt.Printf("(paste into a JSON map: \"<arabic>\": \"<english>\")\n\n")
	// Emit ready-to-edit JSON skeleton, sorted by frequency desc.
	keys := sortedKeys(counts)
	sort.SliceStable(keys, func(i, j int) bool { return counts[keys[i]] > counts[keys[j]] })
	fmt.Println("{")
	for i, k := range keys {
		comma := ","
		if i == len(keys)-1 {
			comma = ""
		}
		b, _ := json.Marshal(k)
		fmt.Printf("  %s: \"\"%s   // ×%d\n", string(b), comma, counts[k])
	}
	fmt.Println("}")
}

func hasArabic(s string) bool {
	for _, r := range s {
		if (r >= 0x0600 && r <= 0x06FF) || (r >= 0x0750 && r <= 0x077F) ||
			(r >= 0xFB50 && r <= 0xFDFF) || (r >= 0xFE70 && r <= 0xFEFF) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func loadMap(path string) map[string]string {
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read tr map: %v", err)
	}
	m := map[string]string{}
	if err := json.Unmarshal(raw, &m); err != nil {
		log.Fatalf("parse tr map: %v", err)
	}
	return m
}
