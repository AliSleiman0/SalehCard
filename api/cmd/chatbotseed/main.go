// Command chatbotseed writes the AI help assistant's starter knowledge
// (chatbot.StarterKnowledge: the general text plus the Android-only text) when
// no knowledge exists yet. Admins then rewrite it freely in admin → Help bot.
//
// It is a DRY RUN unless --apply is passed, and refuses to write to prod
// without --confirm-prod. It never overwrites: once any knowledge was saved
// (by an admin or a previous run) it does nothing.
//
// --whatsapp optionally sets the support WhatsApp number the Help Center's
// contact button links to (app_settings.supportWhatsapp; free-typed, the
// clients keep only the digits). Phone and email stay in admin → Settings.
//
//	MONGO_URI=... go run ./cmd/chatbotseed [--whatsapp "+961 70 123 456"]
//	MONGO_URI=... go run ./cmd/chatbotseed --apply [--confirm-prod] [--whatsapp ...]
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
	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

func isProdURI(uri string) bool {
	return strings.Contains(uri, "cosmos.azure.com") || strings.Contains(uri, "mongocluster")
}

func main() {
	apply := flag.Bool("apply", false, "write the starter knowledge (default: dry run)")
	confirmProd := flag.Bool("confirm-prod", false, "required together with --apply against the prod cluster")
	whatsapp := flag.String("whatsapp", "", "also set the support WhatsApp number in app settings")
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

	target := "local"
	if prod {
		target = "PROD"
	}
	mode := "DRY RUN"
	if *apply {
		mode = "APPLY"
	}
	fmt.Printf("chatbotseed: %s against %s (db %s)\n\n", mode, target, cfg.DBName)

	wrote, err := chatbot.SeedKnowledge(ctx, chatbot.NewMongoKnowledgeRepository(db), chatbot.StarterKnowledge, *apply, time.Now().UTC())
	if err != nil {
		log.Fatalf("seed knowledge: %v", err)
	}
	switch {
	case wrote && *apply:
		fmt.Println("AI knowledge: starter text written")
	case wrote:
		fmt.Println("AI knowledge: would write the starter text")
	default:
		fmt.Println("AI knowledge: already present — left untouched")
	}

	if w := strings.TrimSpace(*whatsapp); w != "" {
		if !validWhatsapp(w) {
			log.Fatalf("--whatsapp %q: expected a phone number with 6-15 digits", w)
		}
		repo := settings.NewMongoRepository(db)
		cur, err := repo.Get(ctx)
		if err != nil {
			log.Fatalf("read settings: %v", err)
		}
		fmt.Printf("\nsupport WhatsApp: %q -> %q", cur.SupportWhatsapp, w)
		if !*apply {
			fmt.Println(" (dry run, not saved)")
			return
		}
		if _, err := repo.Update(ctx, settings.UpdateInput{SupportWhatsapp: &w}, "chatbotseed"); err != nil {
			log.Fatalf("save settings: %v", err)
		}
		fmt.Println(" (saved)")
	}
}

// validWhatsapp accepts a free-typed phone number: digits plus spaces, dashes,
// dots, parentheses and a leading +, with 6–15 digits (E.164 max).
func validWhatsapp(s string) bool {
	digits := 0
	for i, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '+' && i == 0, r == ' ', r == '-', r == '.', r == '(', r == ')':
		default:
			return false
		}
	}
	return digits >= 6 && digits <= 15
}
