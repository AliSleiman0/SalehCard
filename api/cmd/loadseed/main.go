// Command loadseed reads the catalog importer's JSON seeds and upserts them into
// MongoDB (idempotent, keyed on legacyId). It is invoked manually after the
// seeds under /migration/seed are reviewed. Use --dry-run to preview counts
// without writing.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/migration/loadseed"
	"github.com/AliSleiman0/salehcard/api/internal/modules/category"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

func main() {
	seedDir := flag.String("seed-dir", "../migration/seed", "directory with categories.json + products.json")
	dryRun := flag.Bool("dry-run", false, "report inserts/updates without writing")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)

	if !*dryRun {
		if err := category.EnsureIndexes(ctx, db); err != nil {
			log.Fatalf("category indexes: %v", err)
		}
		if err := product.EnsureIndexes(ctx, db); err != nil {
			log.Fatalf("product indexes: %v", err)
		}
	}

	loader := &loadseed.Loader{
		Cats:  category.NewMongoRepository(db),
		Prods: product.NewMongoRepository(db),
	}

	mode := "LOAD"
	if *dryRun {
		mode = "DRY-RUN"
	}
	log.Printf("%s from %s", mode, *seedDir)

	rep, err := loader.Run(ctx, *seedDir, *dryRun)
	if err != nil {
		log.Fatalf("load failed: %v", err)
	}
	log.Printf("categories: +%d inserted, ~%d updated", rep.CategoriesInserted, rep.CategoriesUpdated)
	log.Printf("products:   +%d inserted, ~%d updated", rep.ProductsInserted, rep.ProductsUpdated)
	if *dryRun {
		log.Println("dry-run: nothing written")
	}
}
