// Command chatbotseed loads the help bot's starter questions and answers
// (chatbot.StarterContent, English + Arabic) so the help button appears for
// customers. Admins then edit or remove any of it in admin → Help bot.
//
// It is a DRY RUN unless --apply is passed, and refuses to write to prod
// without --confirm-prod. It never overwrites or deletes: a question whose
// English text already exists at the same level is skipped, so a re-run, or a
// run after an admin added their own entries, is safe.
//
// The support contacts (WhatsApp, phone, email) are NOT seeded: set them in
// admin → Settings.
//
//	MONGO_URI=... go run ./cmd/chatbotseed
//	MONGO_URI=... go run ./cmd/chatbotseed --apply [--confirm-prod]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/chatbot"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

func isProdURI(uri string) bool {
	return strings.Contains(uri, "cosmos.azure.com") || strings.Contains(uri, "mongocluster")
}

func main() {
	apply := flag.Bool("apply", false, "write the entries (default: dry run)")
	confirmProd := flag.Bool("confirm-prod", false, "required together with --apply against the prod cluster")
	flag.Parse()

	_ = godotenv.Load()
	cfg := config.Load()
	prod := isProdURI(cfg.MongoURI)
	if *apply && prod && !*confirmProd {
		log.Fatal("refusing to write to PROD without --confirm-prod (review a dry run first)")
	}

	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := chatbot.EnsureIndexes(ctx, db); err != nil {
		log.Printf("warning: ensure indexes: %v", err)
	}

	target := "local"
	if prod {
		target = "PROD"
	}
	mode := "DRY RUN"
	if *apply {
		mode = "APPLY"
	}
	fmt.Printf("chatbotseed: %s against %s (db %s)\n\n", mode, target, cfg.DBName)

	res, err := chatbot.Seed(ctx, chatbot.NewMongoRepository(db), chatbot.StarterContent, *apply, time.Now().UTC())
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	verb := "would create"
	if *apply {
		verb = "created"
	}
	fmt.Printf("%s %d:\n", verb, len(res.Created))
	for _, q := range res.Created {
		fmt.Println("  " + q)
	}
	fmt.Printf("\nskipped %d (already present):\n", len(res.Skipped))
	for _, q := range res.Skipped {
		fmt.Println("  " + q)
	}
}
