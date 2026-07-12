// Command fixadmin restores a single account that was accidentally changed from
// admin to reseller back to role=admin (the reseller-promotion foot-gun lockout).
// It ONLY acts on the one -email account, and ONLY when that account is currently
// role=reseller — it refuses otherwise, so it can never grant admin to an
// arbitrary account. Runs as a DRY RUN by default; pass --apply to write.
//
//	MONGO_URI='<prod>' go run ./cmd/fixadmin -email you@example.com           # preview
//	MONGO_URI='<prod>' go run ./cmd/fixadmin -email you@example.com --apply   # write
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

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

func main() {
	email := flag.String("email", "", "email of the account to restore to admin (required)")
	apply := flag.Bool("apply", false, "perform the write (default is a dry run)")
	flag.Parse()

	if strings.TrimSpace(*email) == "" {
		log.Fatalf("-email is required")
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

	roleID := "<nil>"
	if u.AdminRoleID != nil {
		roleID = u.AdminRoleID.Hex()
	}
	log.Printf("found: id=%s role=%s resellerTier=%q status=%s adminRoleId=%s",
		u.ID.Hex(), u.Role, u.ResellerTier, u.Status, roleID)

	if u.Role == user.RoleAdmin {
		log.Printf("OK: already role=admin — nothing to do")
		return
	}
	if u.Role != user.RoleReseller {
		log.Fatalf("ABORT: account is role=%s, not reseller — refusing (this tool only reverses the reseller lockout)", u.Role)
	}

	log.Printf("WILL SET role=admin and unset resellerTier on %s (adminRoleId left as-is → implicit super admin when nil)", u.ID.Hex())
	if !*apply {
		log.Printf("dry run — no changes written. Re-run with --apply.")
		return
	}

	res, err := col.UpdateByID(ctx, u.ID, bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "role", Value: user.RoleAdmin},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}},
		{Key: "$unset", Value: bson.D{{Key: "resellerTier", Value: ""}}},
	})
	if err != nil {
		log.Fatalf("update: %v", err)
	}
	log.Printf("APPLIED. matched=%d modified=%d — %s is admin again. Log in (2FA to the phone on this account) to confirm.",
		res.MatchedCount, res.ModifiedCount, *email)
}
