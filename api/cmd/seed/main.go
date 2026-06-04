package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
	seed "github.com/AliSleiman0/salehcard/api/internal/modules/product/seed"
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
		log.Fatalf("seed failed: %v", err)
	}

	log.Println("seed completed successfully")
}
