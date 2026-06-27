package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	orderseed "github.com/AliSleiman0/salehcard/api/internal/modules/order/seed"
	seed "github.com/AliSleiman0/salehcard/api/internal/modules/product/seed"
	promoseed "github.com/AliSleiman0/salehcard/api/internal/modules/promo/seed"
	resellerseed "github.com/AliSleiman0/salehcard/api/internal/modules/reseller/seed"
	userseed "github.com/AliSleiman0/salehcard/api/internal/modules/user/seed"
	walletseed "github.com/AliSleiman0/salehcard/api/internal/modules/wallet/seed"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

func main() {
	// Load .env file when present (ignored if absent).
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)

	log.Println("running product seed...")

	if err := seed.Seed(ctx, db); err != nil {
		log.Fatalf("product seed failed: %v", err)
	}

	log.Println("running user seed...")

	if err := userseed.Seed(ctx, db); err != nil {
		log.Fatalf("user seed failed: %v", err)
	}

	log.Println("running reseller seed...")

	if err := resellerseed.Seed(ctx, db); err != nil {
		log.Fatalf("reseller seed failed: %v", err)
	}

	log.Println("running order seed...")

	if err := orderseed.Seed(ctx, db); err != nil {
		log.Fatalf("order seed failed: %v", err)
	}

	log.Println("running wallet seed...")

	if err := walletseed.Seed(ctx, db); err != nil {
		log.Fatalf("wallet seed failed: %v", err)
	}

	log.Println("running promo seed...")

	if err := promoseed.Seed(ctx, db); err != nil {
		log.Fatalf("promo seed failed: %v", err)
	}

	log.Println("seed completed successfully")
}
