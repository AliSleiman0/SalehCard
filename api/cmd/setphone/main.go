// Command setphone attaches a phone number to one existing account (by email),
// so it can log in on the phone-only mobile app. It refuses if the phone is
// already held by a different account (the users.phone index is sparse-unique).
// Store the number in E.164 (+CCXXXXXXXX) — that's what the backend normalizes
// login input to. Runs as a DRY RUN by default; pass --apply to write.
//
//	MONGO_URI='<prod>' go run ./cmd/setphone -email you@x.com -phone +96176010101          # preview
//	MONGO_URI='<prod>' go run ./cmd/setphone -email you@x.com -phone +96176010101 --apply   # write
package main

import (
	"context"
	"flag"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

var e164 = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

func main() {
	email := flag.String("email", "", "email of the account to attach the phone to (required)")
	phone := flag.String("phone", "", "phone in E.164, e.g. +96176010101 (required)")
	apply := flag.Bool("apply", false, "perform the write (default is a dry run)")
	flag.Parse()

	*email = strings.ToLower(strings.TrimSpace(*email))
	*phone = strings.TrimSpace(*phone)
	if *email == "" || *phone == "" {
		log.Fatalf("-email and -phone are required")
	}
	if !e164.MatchString(*phone) {
		log.Fatalf("-phone must be E.164, e.g. +96176010101 (got %q)", *phone)
	}
	if *apply {
		log.Printf("MODE: APPLY (writes enabled)")
	} else {
		log.Printf("MODE: DRY RUN (no writes) — pass --apply to commit")
	}

	_ = godotenv.Load()
	cfg := config.Load()
	log.Printf("db=%s target=%s phone=%s", cfg.DBName, *email, *phone)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)
	col := db.Collection("users")

	// Target account must exist.
	var u user.User
	err = col.FindOne(ctx, bson.D{{Key: "email", Value: *email}}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		log.Fatalf("ABORT: no account with email %q", *email)
	}
	if err != nil {
		log.Fatalf("lookup: %v", err)
	}
	cur := "<none>"
	if u.Phone != nil {
		cur = *u.Phone
	}
	log.Printf("found: id=%s role=%s currentPhone=%s status=%s", u.ID.Hex(), u.Role, cur, u.Status)

	// The phone must not already belong to a different account.
	var holder user.User
	err = col.FindOne(ctx, bson.D{{Key: "phone", Value: *phone}}).Decode(&holder)
	if err == nil && holder.ID != u.ID {
		log.Fatalf("ABORT: phone %s already belongs to a different account (id=%s email=%q)", *phone, holder.ID.Hex(), holder.Email)
	}
	if err != nil && err != mongo.ErrNoDocuments {
		log.Fatalf("phone lookup: %v", err)
	}
	if u.Phone != nil && *u.Phone == *phone {
		log.Printf("OK: account already has this phone — nothing to do")
		return
	}

	log.Printf("WILL SET phone=%s on %s (%s)", *phone, *email, u.ID.Hex())
	if !*apply {
		log.Printf("dry run — no changes written. Re-run with --apply.")
		return
	}
	res, err := col.UpdateByID(ctx, u.ID, bson.D{{Key: "$set", Value: bson.D{
		{Key: "phone", Value: *phone},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}}})
	if err != nil {
		log.Fatalf("update: %v", err)
	}
	log.Printf("APPLIED. matched=%d modified=%d — %s can now log in on the app with %s + its password.",
		res.MatchedCount, res.ModifiedCount, *email, *phone)
}
