// Command bridgedirect reconfigures Lebanese mobile-recharge "direct" products
// so they charge the customer's line via the Android bridge.
//
// Root cause it fixes: the alfa-direct / mtc-direct products were imported in
// three inconsistent half-states — some account_credit+manual_operator, some
// account_credit+bridge_device but with bridge=null (bridge mode, no spec), and
// at least one left as fulfillmentType=code. None carried a bridge spec, none
// used the load-bearing `phone` input-field key, and all still had a `qty`
// (quantity) field that a single-line bridge recharge forbids. The net effect:
// no working "enter the number to charge" flow.
//
// Target state (per selected product):
//   - fulfillmentType = account_credit
//   - fulfillmentMode = bridge_device
//   - bridge          = {provider (from category), method (flag, default recharge_line)}
//   - inputFields     = exactly one field keyed `phone` (reusing the product's
//                       existing localized number label), dropping qty/duplicates.
//
// Provider is derived from the category slug: alfa* -> alfa, mtc*/touch* -> touch.
//
// Reads MONGO_URI (intended for prod). Dry run by default; only touches the
// products whose category is in -categories.
//
// Usage:
//
//	MONGO_URI='<prod>' go run ./cmd/bridgedirect                    # preview
//	MONGO_URI='<prod>' go run ./cmd/bridgedirect --apply            # write
//	MONGO_URI='<prod>' go run ./cmd/bridgedirect -categories alfa-direct,mtc-direct -method recharge_line --apply
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

// rawField decodes just the input-field properties this tool reads.
type rawField struct {
	Key       string            `bson:"key"`
	Label     product.I18nLabel `bson:"label"`
	Type      string            `bson:"type"`
	Sensitive bool              `bson:"sensitive"`
}

// rawProduct decodes just what the migration needs from a product document.
type rawProduct struct {
	ID          bson.ObjectID     `bson:"_id"`
	Category    string            `bson:"category"`
	Title       product.I18nString `bson:"title"`
	FulType     string            `bson:"fulfillmentType"`
	FulMode     string            `bson:"fulfillmentMode"`
	Bridge      *product.BridgeSpec `bson:"bridge"`
	InputFields []rawField        `bson:"inputFields"`
}

// providerFor maps a category slug to the bridge operator it recharges.
func providerFor(category string) (product.BridgeProvider, error) {
	c := strings.ToLower(category)
	switch {
	case strings.HasPrefix(c, "alfa"):
		return product.BridgeProviderAlfa, nil
	case strings.HasPrefix(c, "mtc"), strings.HasPrefix(c, "touch"):
		return product.BridgeProviderTouch, nil
	default:
		return "", fmt.Errorf("cannot derive bridge provider from category %q", category)
	}
}

// phoneField builds the single `phone` input field for a bridge recharge,
// reusing the product's existing number label when present so the localized
// prompt is preserved. The key `phone` is load-bearing (the app validates a
// Lebanese mobile only on it; the server's bridgePhone fallback matches only it).
func phoneField(existing []rawField) product.InputField {
	var label product.I18nLabel
	// Prefer an existing field already keyed phone, else the first text field.
	pick := -1
	for i, f := range existing {
		if f.Key == "phone" {
			pick = i
			break
		}
	}
	if pick == -1 {
		for i, f := range existing {
			if f.Type == "text" {
				pick = i
				break
			}
		}
	}
	if pick >= 0 {
		label = existing[pick].Label
	}
	if strings.TrimSpace(label.En) == "" {
		label.En = "Mobile number"
	}
	if strings.TrimSpace(label.Ar) == "" {
		label.Ar = "رقم الهاتف"
	}
	return product.InputField{
		Key:       "phone",
		Label:     label,
		Type:      product.InputFieldText,
		Sensitive: false,
	}
}

// planUpdate computes the target field values for one product. It always
// produces the same canonical target (the operation is idempotent), so callers
// can compare against the current doc to decide whether a write is needed.
func planUpdate(p rawProduct) (bson.D, product.BridgeSpec, product.InputField, error) {
	prov, err := providerFor(p.Category)
	if err != nil {
		return nil, product.BridgeSpec{}, product.InputField{}, err
	}
	spec := product.BridgeSpec{Provider: prov, Method: bridgeMethod}
	phone := phoneField(p.InputFields)
	set := bson.D{
		{Key: "fulfillmentType", Value: string(product.FulfillmentCredit)},
		{Key: "fulfillmentMode", Value: string(product.FulfillmentModeBridgeDevice)},
		{Key: "bridge", Value: spec},
		{Key: "inputFields", Value: []product.InputField{phone}},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}
	return set, spec, phone, nil
}

// bridgeMethod is the delivery method the tool writes (flag-settable). Package
// global so planUpdate stays a pure function of the product for testing.
var bridgeMethod product.BridgeMethod = product.BridgeMethodRechargeLine

func fieldKeys(fs []rawField) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Key
	}
	return out
}

func main() {
	apply := flag.Bool("apply", false, "perform writes (default is a dry run)")
	cats := flag.String("categories", "alfa-direct,mtc-direct", "comma-separated category slugs to migrate")
	method := flag.String("method", string(product.BridgeMethodRechargeLine), "bridge method: recharge_line | transfer_credit")
	flag.Parse()

	m := product.BridgeMethod(strings.TrimSpace(*method))
	if m != product.BridgeMethodRechargeLine && m != product.BridgeMethodTransferCredit {
		log.Fatalf("invalid -method %q (want recharge_line or transfer_credit)", *method)
	}
	bridgeMethod = m
	if m == product.BridgeMethodTransferCredit {
		log.Printf("WARNING: transfer_credit requires a positive faceValue on every variant; this tool does NOT set faceValue.")
	}

	categories := make([]string, 0)
	for _, c := range strings.Split(*cats, ",") {
		if s := strings.TrimSpace(c); s != "" {
			categories = append(categories, s)
		}
	}
	if len(categories) == 0 {
		log.Fatalf("no categories given")
	}

	_ = godotenv.Load()
	cfg := config.Load()

	if *apply {
		log.Printf("MODE: APPLY (writes enabled)")
	} else {
		log.Printf("MODE: DRY RUN (no writes) — pass --apply to commit")
	}
	log.Printf("db=%s  categories=%v  method=%s", cfg.DBName, categories, bridgeMethod)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)

	col := db.Collection("products")
	cur, err := col.Find(ctx, bson.D{{Key: "category", Value: bson.D{{Key: "$in", Value: categories}}}})
	if err != nil {
		log.Fatalf("find: %v", err)
	}
	var prods []rawProduct
	if err := cur.All(ctx, &prods); err != nil {
		log.Fatalf("decode: %v", err)
	}
	log.Printf("matched %d products", len(prods))

	changed := 0
	for _, p := range prods {
		set, spec, phone, err := planUpdate(p)
		if err != nil {
			log.Printf("SKIP %s (%s): %v", p.ID.Hex(), p.Title.En, err)
			continue
		}
		fmt.Printf("\n%s  %q\n", p.ID.Hex(), p.Title.En)
		fmt.Printf("  before: ft=%s fm=%s bridge=%v fields=%v\n", p.FulType, p.FulMode, p.Bridge, fieldKeys(p.InputFields))
		fmt.Printf("  after : ft=%s fm=%s bridge={%s,%s} fields=[phone(en=%q,ar=%q)]\n",
			product.FulfillmentCredit, product.FulfillmentModeBridgeDevice, spec.Provider, spec.Method, phone.Label.En, phone.Label.Ar)
		changed++

		if *apply {
			res, err := col.UpdateByID(ctx, p.ID, bson.D{{Key: "$set", Value: set}})
			if err != nil {
				log.Fatalf("update %s: %v", p.ID.Hex(), err)
			}
			fmt.Printf("  APPLIED (matched=%d modified=%d)\n", res.MatchedCount, res.ModifiedCount)
		}
	}

	fmt.Printf("\n%d product(s) %s.\n", changed, map[bool]string{true: "updated", false: "would change"}[*apply])
	if !*apply {
		fmt.Printf("re-run with --apply to write.\n")
	}
}
