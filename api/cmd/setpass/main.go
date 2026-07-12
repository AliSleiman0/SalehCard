// Command setpass resets the password on one existing account (by email) — used
// to give a test account a simple, typo-proof password for mobile login. Runs as
// a DRY RUN by default; pass --apply to write.
//
//	MONGO_URI='<prod>' go run ./cmd/setpass -email you@x.com -password salehqa2026          # preview
//	MONGO_URI='<prod>' go run ./cmd/setpass -email you@x.com -password salehqa2026 --apply   # write
package main

import (
	"context"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

func main() {
	email := flag.String("email", "", "email of the account (required)")
	password := flag.String("password", "", "new password, min 8 chars (required)")
	apply := flag.Bool("apply", false, "perform the write (default is a dry run)")
	flag.Parse()

	*email = strings.ToLower(strings.TrimSpace(*email))
	if *email == "" || len(*password) < 8 {
		log.Fatalf("-email and -password (>= 8 chars) are required")
	}
	if *apply {
		log.Printf("MODE: APPLY (writes enabled)")
	} else {
		log.Printf("MODE: DRY RUN (no writes) — pass --apply to commit")
	}

	_ = godotenv.Load()
	cfg := config.Load()
	log.Printf("db=%s target=%s", cfg.DBName, *email)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)
	col := db.Collection("users")

	var u user.User
	err = col.FindOne(ctx, bson.D{{Key: "email", Value: *email}}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		log.Fatalf("ABORT: no account with email %q", *email)
	}
	if err != nil {
		log.Fatalf("lookup: %v", err)
	}
	log.Printf("found: id=%s role=%s status=%s", u.ID.Hex(), u.Role, u.Status)

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}

	log.Printf("WILL SET password on %s (%s)", *email, u.ID.Hex())
	if !*apply {
		log.Printf("dry run — no changes written. Re-run with --apply.")
		return
	}
	res, err := col.UpdateByID(ctx, u.ID, bson.D{{Key: "$set", Value: bson.D{
		{Key: "passwordHash", Value: string(hash)},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}}})
	if err != nil {
		log.Fatalf("update: %v", err)
	}
	log.Printf("APPLIED. matched=%d modified=%d — password reset for %s.", res.MatchedCount, res.ModifiedCount, *email)
}
