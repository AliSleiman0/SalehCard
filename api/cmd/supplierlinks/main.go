// Command supplierlinks moves every product's single legacy supplier mapping
// (fulfillmentProvider + upstreamProductId) into the supplier links table
// (Product.SupplierLinks): one enabled, active link, rev 1. The legacy fields
// stay as they are — they are now the mirror of that active link.
//
// The API already treats an unmigrated product as having one virtual link, so
// this migration is not required for anything to work; it makes the links
// first-class (supplier pages count and import-dedupe on supplierLinks).
//
// It is a DRY RUN unless --apply is passed, writes a CSV report of every
// candidate, and refuses to write to prod without --confirm-prod. A product
// that already has supplierLinks is never touched, so a re-run is safe; each
// write is guarded on the mapping it read still being in place.
//
//	MONGO_URI=... go run ./cmd/supplierlinks --report supplierlinks.csv
//	MONGO_URI=... go run ./cmd/supplierlinks --report supplierlinks.csv --apply [--confirm-prod]
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
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

// candidate is the slice of a product document the migration reads.
type candidate struct {
	ID                  bson.ObjectID          `bson:"_id"`
	Title               product.I18nString     `bson:"title"`
	FulfillmentMode     string                 `bson:"fulfillmentMode"`
	FulfillmentProvider *int                   `bson:"fulfillmentProvider"`
	UpstreamProductID   string                 `bson:"upstreamProductId"`
	SupplierLinks       []product.SupplierLink `bson:"supplierLinks"`
	DeletedAt           *time.Time             `bson:"deletedAt"`
	CreatedAt           time.Time              `bson:"createdAt"`
}

// plan decides whether c gets a links table; a non-empty reason means skip.
// Pure — unit-tested without a database.
func plan(c candidate, now time.Time) (product.SupplierLink, string) {
	if len(c.SupplierLinks) > 0 {
		return product.SupplierLink{}, "already has supplier links"
	}
	if c.FulfillmentProvider == nil || *c.FulfillmentProvider == 0 {
		return product.SupplierLink{}, "no supplier"
	}
	up := strings.TrimSpace(c.UpstreamProductID)
	if up == "" {
		return product.SupplierLink{}, "supplier set but no supplier product id — link it in the editor"
	}
	added := c.CreatedAt
	if added.IsZero() {
		added = now
	}
	return product.SupplierLink{
		ID:         bson.NewObjectID().Hex(),
		Provider:   *c.FulfillmentProvider,
		UpstreamID: up,
		Enabled:    true,
		AddedAt:    added,
		AddedBy:    "migration:supplierlinks",
	}, ""
}

func isProdURI(uri string) bool {
	return strings.Contains(uri, "cosmos.azure.com") || strings.Contains(uri, "mongocluster")
}

func main() {
	apply := flag.Bool("apply", false, "write the changes (default is a dry run)")
	confirmProd := flag.Bool("confirm-prod", false, "required together with --apply against the prod cluster")
	report := flag.String("report", "supplierlinks-report.csv", "CSV report path")
	flag.Parse()

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
	log.Printf("supplierlinks: db=%s (%s) apply=%v", cfg.DBName, env, *apply)

	col := db.Collection("products")
	cur, err := col.Find(ctx, bson.D{{Key: "fulfillmentProvider", Value: bson.D{{Key: "$nin", Value: bson.A{nil, 0}}}}})
	if err != nil {
		log.Fatalf("find: %v", err)
	}
	var cands []candidate
	if err := cur.All(ctx, &cands); err != nil {
		log.Fatalf("decode: %v", err)
	}

	f, err := os.Create(*report)
	if err != nil {
		log.Fatalf("report: %v", err)
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"product_id", "title", "mode", "trashed", "provider", "upstream_id", "action", "reason"})

	now := time.Now().UTC()
	verb := "would link"
	if *apply {
		verb = "linked"
	}
	var changed, skipped, failed int
	for _, c := range cands {
		prov := ""
		if c.FulfillmentProvider != nil {
			prov = strconv.Itoa(*c.FulfillmentProvider)
		}
		row := []string{c.ID.Hex(), c.Title.En, c.FulfillmentMode, strconv.FormatBool(c.DeletedAt != nil), prov, c.UpstreamProductID}
		link, reason := plan(c, now)
		if reason != "" {
			skipped++
			_ = w.Write(append(row, "skip", reason))
			continue
		}
		if *apply {
			if err := applyLink(ctx, col, c, link); err != nil {
				failed++
				_ = w.Write(append(row, "error", err.Error()))
				continue
			}
		}
		changed++
		_ = w.Write(append(row, verb, ""))
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatalf("report: %v", err)
	}
	_ = f.Close()
	log.Printf("supplierlinks: %d candidates — %s %d, skipped %d, failed %d — report %s",
		len(cands), verb, changed, skipped, failed, *report)
}

// applyLink stores the links table, guarded on the product still having no
// links and the same legacy mapping it was planned from.
func applyLink(ctx context.Context, col *mongo.Collection, c candidate, link product.SupplierLink) error {
	res, err := col.UpdateOne(ctx,
		bson.D{
			{Key: "_id", Value: c.ID},
			{Key: "supplierLinks", Value: bson.D{{Key: "$exists", Value: false}}},
			{Key: "fulfillmentProvider", Value: link.Provider},
			{Key: "upstreamProductId", Value: c.UpstreamProductID},
		},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "supplierLinks", Value: []product.SupplierLink{link}},
			{Key: "activeSupplierLinkId", Value: link.ID},
			{Key: "supplierLinksRev", Value: 1},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("changed since it was read — re-run")
	}
	return nil
}
