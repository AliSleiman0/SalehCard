// Command dbtool is a reusable, safe CLI for inspecting and maintaining the
// SalehCard database — intended for prod via MONGO_URI (see the prod-DB firewall
// recipe in the ops notes). Read-only subcommands run freely; destructive ones
// are a DRY RUN unless you pass --apply.
//
// Connect (prod): add a Cosmos firewall rule for your egress IP, then feed
// MONGO_URI from the gitignored DEPLOY-CREDS.local.md, e.g.
//
//	MONGO_URI="$(grep -oE 'mongodb\+srv://[^[:space:]]*maxIdleTimeMS=120000' ../DEPLOY-CREDS.local.md | head -1)" \
//	  go run ./cmd/dbtool <command> [args]
//
// Commands:
//
//	product <id>              Show a product's fulfillment config + code counts by status
//	codes <productId>         List a product's code rows (masked) and any linked orders
//	codes-clear <productId>   Delete ALL of a product's codes so it can be re-uploaded
//	                          (DRY RUN; pass --apply to write; resets product.stock to 0)
//	order <id>                Show an order summary (status, total, user, items)
//
// Examples:
//
//	MONGO_URI=... go run ./cmd/dbtool product 6a3f0403ea6747f81d0bae28
//	MONGO_URI=... go run ./cmd/dbtool codes-clear 6a3f0403ea6747f81d0bae28 --apply
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

const usage = `dbtool — inspect & maintain the SalehCard DB (reads MONGO_URI)

Usage:
  MONGO_URI=<uri> go run ./cmd/dbtool <command> [args] [--apply]

Commands:
  product <id>              Product fulfillment config + code counts by status (read-only)
  codes <productId>         List a product's code rows (masked) + linked orders (read-only)
  codes-clear <productId>   Delete ALL of a product's codes; resets stock (DRY RUN unless --apply)
  order <id>                Order summary (read-only)
`

// mask hides all but the last 4 characters of a code/PIN value.
func mask(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return "…" + s[len(s)-4:]
}

// isProdURI reports whether the connection string points at the managed
// (Cosmos) cluster, so the tool can warn before a destructive write.
func isProdURI(uri string) bool {
	return strings.Contains(uri, "cosmos.azure.com") || strings.Contains(uri, "mongocluster")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	// Help needs no DB connection.
	if cmd == "-h" || cmd == "--help" || cmd == "help" {
		fmt.Print(usage)
		return
	}

	_ = godotenv.Load()
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)

	env := "local"
	if isProdURI(cfg.MongoURI) {
		env = "PROD"
	}
	log.Printf("dbtool: db=%s (%s)", cfg.DBName, env)

	switch cmd {
	case "product":
		mustArg(args, "product <id>")
		cmdProduct(ctx, db, args[0])
	case "codes":
		mustArg(args, "codes <productId>")
		cmdCodes(ctx, db, args[0])
	case "codes-clear":
		fs := flag.NewFlagSet("codes-clear", flag.ExitOnError)
		apply := fs.Bool("apply", false, "delete the codes (default is a dry run)")
		_ = fs.Parse(args)
		if fs.NArg() < 1 {
			log.Fatalf("usage: codes-clear <productId> [--apply]")
		}
		cmdCodesClear(ctx, db, fs.Arg(0), *apply)
	case "order":
		mustArg(args, "order <id>")
		cmdOrder(ctx, db, args[0])
	default:
		fmt.Printf("unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
}

func mustArg(args []string, form string) {
	if len(args) < 1 {
		log.Fatalf("usage: %s", form)
	}
}

func productByID(ctx context.Context, db *mongo.Database, id string) (bson.M, bson.ObjectID) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		log.Fatalf("bad product id: %v", err)
	}
	var p bson.M
	if err := db.Collection("products").FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&p); err != nil {
		log.Fatalf("product not found: %v", err)
	}
	return p, oid
}

func cmdProduct(ctx context.Context, db *mongo.Database, id string) {
	p, _ := productByID(ctx, db, id)
	fmt.Printf("PRODUCT %s  %v\n", id, p["title"])
	for _, k := range []string{"category", "fulfillmentType", "fulfillmentMode", "stock", "available", "bridge"} {
		fmt.Printf("  %-16s= %v\n", k, p[k])
	}
	counts := codeCounts(ctx, db, id)
	fmt.Printf("  codeCounts     = %v\n", counts)
}

func cmdCodes(ctx context.Context, db *mongo.Database, productID string) {
	codes := findCodes(ctx, db, productID)
	fmt.Printf("%d code row(s) for productId=%q:\n", len(codes), productID)
	for _, c := range codes {
		fmt.Printf("  code=%s status=%v order=%v batch=%v pin=%v\n",
			mask(fmt.Sprint(c["code"])), c["status"], c["orderId"], c["batch"], c["pin"] != nil)
		printLinkedOrder(ctx, db, c["orderId"])
	}
	fmt.Printf("counts: %v\n", tallyStatus(codes))
}

func cmdCodesClear(ctx context.Context, db *mongo.Database, productID string, apply bool) {
	p, oid := productByID(ctx, db, productID)
	fmt.Printf("PRODUCT %s  %v\n\n", productID, p["title"])
	codes := findCodes(ctx, db, productID)
	fmt.Printf("%d code row(s) would be deleted:\n", len(codes))
	for _, c := range codes {
		fmt.Printf("  code=%s status=%v order=%v batch=%v\n", mask(fmt.Sprint(c["code"])), c["status"], c["orderId"], c["batch"])
		printLinkedOrder(ctx, db, c["orderId"])
	}
	if !apply {
		fmt.Printf("\nDRY RUN — re-run with --apply to delete these %d row(s).\n", len(codes))
		return
	}
	res, err := db.Collection("codes").DeleteMany(ctx, bson.D{{Key: "productId", Value: productID}})
	if err != nil {
		log.Fatalf("delete: %v", err)
	}
	_, _ = db.Collection("products").UpdateByID(ctx, oid,
		bson.D{{Key: "$set", Value: bson.D{{Key: "stock", Value: 0}, {Key: "updatedAt", Value: time.Now().UTC()}}}})
	fmt.Printf("\nDELETED %d code row(s); product.stock reset to 0.\n", res.DeletedCount)
}

func cmdOrder(ctx context.Context, db *mongo.Database, id string) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		log.Fatalf("bad order id: %v", err)
	}
	var o bson.M
	if err := db.Collection("orders").FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&o); err != nil {
		log.Fatalf("order not found: %v", err)
	}
	fmt.Printf("ORDER %s\n", id)
	for _, k := range []string{"status", "total", "userId", "paymentMethod", "createdAt"} {
		fmt.Printf("  %-14s= %v\n", k, o[k])
	}
	if items, ok := o["items"].(bson.A); ok {
		fmt.Printf("  items (%d):\n", len(items))
		for _, it := range items {
			if m, ok := it.(bson.M); ok {
				fmt.Printf("    - product=%v qty=%v playerId=%v status=%v\n", m["productId"], m["qty"], m["playerId"], m["status"])
			}
		}
	}
}

// --- shared helpers -------------------------------------------------------

func findCodes(ctx context.Context, db *mongo.Database, productID string) []bson.M {
	cur, err := db.Collection("codes").Find(ctx, bson.D{{Key: "productId", Value: productID}})
	if err != nil {
		log.Fatalf("codes find: %v", err)
	}
	var codes []bson.M
	if err := cur.All(ctx, &codes); err != nil {
		log.Fatalf("codes decode: %v", err)
	}
	return codes
}

func codeCounts(ctx context.Context, db *mongo.Database, productID string) map[string]int {
	return tallyStatus(findCodes(ctx, db, productID))
}

func tallyStatus(codes []bson.M) map[string]int {
	out := map[string]int{}
	for _, c := range codes {
		out[fmt.Sprint(c["status"])]++
	}
	return out
}

// printLinkedOrder surfaces the order behind a delivered code so a caller can
// judge whether it truly fulfilled before wiping the code's trail.
func printLinkedOrder(ctx context.Context, db *mongo.Database, orderID any) {
	s, ok := orderID.(string)
	if !ok || s == "" {
		return
	}
	ooid, err := bson.ObjectIDFromHex(s)
	if err != nil {
		return
	}
	var ord bson.M
	if err := db.Collection("orders").FindOne(ctx, bson.D{{Key: "_id", Value: ooid}}).Decode(&ord); err != nil {
		fmt.Printf("    ↳ order %s not found\n", s)
		return
	}
	fmt.Printf("    ↳ order status=%v total=%v userId=%v createdAt=%v\n",
		ord["status"], ord["total"], ord["userId"], ord["createdAt"])
}
