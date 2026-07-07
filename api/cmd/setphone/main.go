// Command setphone is a targeted, idempotent update: it sets a new phone on the
// account identified by --email, after verifying the target phone isn't already
// held by a DIFFERENT account (the phone index is unique). Dry run by default;
// pass --apply to write. MONGO_URI is read from env (never logged).
//
//	MONGO_URI='<prod>' go run ./cmd/setphone --email saleh.admin@salehcard.com --phone +96171363778
//	MONGO_URI='<prod>' go run ./cmd/setphone --email saleh.admin@salehcard.com --phone +96171363778 --apply
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

func main() {
	email := flag.String("email", "", "account email to update (required)")
	phone := flag.String("phone", "", "new E.164 phone, e.g. +96171363778 (required)")
	apply := flag.Bool("apply", false, "perform the write (default is a dry run)")
	flag.Parse()

	if *email == "" || *phone == "" {
		log.Fatalf("both --email and --phone are required")
	}

	_ = godotenv.Load()
	cfg := config.Load()

	if *apply {
		log.Printf("MODE: APPLY (writes enabled)")
	} else {
		log.Printf("MODE: DRY RUN (no writes) — pass --apply to commit")
	}
	log.Printf("db=%s", cfg.DBName)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer mongoplatform.Disconnect(context.Background(), client)

	col := db.Collection("users")
	now := time.Now().UTC()

	target, err := findOne(ctx, col, bson.D{{Key: "email", Value: *email}})
	if err != nil {
		log.Fatalf("finding %s: %v", *email, err)
	}
	if target == nil {
		log.Fatalf("ABORT: no account with email %s", *email)
	}
	fmt.Printf("target: id=%s role=%s status=%s phone=%s\n",
		target.ID.Hex(), target.Role, target.Status, phoneStr(target.Phone))

	if target.Phone != nil && *target.Phone == *phone {
		fmt.Printf("OK: %s already has phone %s — nothing to do\n", *email, *phone)
		return
	}

	// Guard: the target phone must not be held by a different account.
	if holder, err := findOne(ctx, col, bson.D{{Key: "phone", Value: *phone}}); err != nil {
		log.Fatalf("checking holder of %s: %v", *phone, err)
	} else if holder != nil && holder.ID != target.ID {
		log.Fatalf("ABORT: phone %s is already held by a different account (id=%s email=%q role=%s) — unique index would reject",
			*phone, holder.ID.Hex(), holder.Email, holder.Role)
	}

	fmt.Printf("WILL SET phone %s -> %s on %s\n", phoneStr(target.Phone), *phone, *email)
	if *apply {
		set := bson.D{{Key: "phone", Value: *phone}, {Key: "updatedAt", Value: now}}
		if _, err := col.UpdateByID(ctx, target.ID, bson.D{{Key: "$set", Value: set}}); err != nil {
			log.Fatalf("update: %v", err)
		}
		fmt.Printf("APPLIED. id=%s now phone=%s\n", target.ID.Hex(), *phone)
	}
}

func findOne(ctx context.Context, col *mongo.Collection, filter bson.D) (*user.User, error) {
	var u user.User
	err := col.FindOne(ctx, filter).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func phoneStr(p *string) string {
	if p == nil {
		return "<none>"
	}
	return *p
}
