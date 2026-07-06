// Command seedadmins performs a targeted, idempotent admin provisioning against
// whatever MONGO_URI points at (intended for prod). It ONLY touches the specific
// accounts described below — it does not run the general dev seed.
//
// Operations (decided with the operator against the real prod state):
//
//  1. admin@salehcard.com — MOVE phone +96178991778 onto it. That number is
//     currently held by a separate phone-only account; the phone is unset there
//     first (that account is intentionally left without a login), then set on the
//     admin. Password/role untouched.
//  2. ousama.admin@gmail.com — PROMOTE the existing phone-only account that holds
//     +96170081637: set email + role=admin + name + a generated password. Keeps
//     the account's phone and wallet.
//  3. saleh.admin@salehcard.com — PROMOTE the existing phone-only account that
//     holds +96171363772 (keeps its $400 wallet): same as above.
//
// Runs as a DRY RUN by default; pass --apply to perform writes.
//
//	MONGO_URI='<prod>' go run ./cmd/seedadmins            # preview
//	MONGO_URI='<prod>' go run ./cmd/seedadmins --apply    # write
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
)

const (
	adminEmail  = "admin@salehcard.com"
	adminPhone  = "+96178991778"
	ousamaEmail = "ousama.admin@gmail.com"
	ousamaPhone = "+96170081637"
	ousamaName  = "Ousama"
	salehEmail  = "saleh.admin@salehcard.com"
	salehPhone  = "+96171363772"
	salehName   = "Saleh"
)

var errAborted = errors.New("precondition failed")

func main() {
	apply := flag.Bool("apply", false, "perform writes (default is a dry run)")
	doAdminPhone := flag.Bool("admin-phone", true, "also move +96178991778 onto admin@salehcard.com (set false to run promotions only)")
	flag.Parse()

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

	if *doAdminPhone {
		if err := moveAdminPhone(ctx, col, now, *apply); err != nil {
			log.Fatalf("admin-phone move failed: %v", err)
		}
	} else {
		fmt.Printf("\n=== 1) admin-phone move SKIPPED (--admin-phone=false) ===\n")
	}
	if err := promote(ctx, col, now, *apply, ousamaEmail, ousamaName, ousamaPhone); err != nil {
		log.Fatalf("ousama promote failed: %v", err)
	}
	if err := promote(ctx, col, now, *apply, salehEmail, salehName, salehPhone); err != nil {
		log.Fatalf("saleh promote failed: %v", err)
	}

	fmt.Printf("\ndone.\n")
}

// moveAdminPhone frees adminPhone from whatever phone-only account currently
// holds it, then sets it on admin@salehcard.com.
func moveAdminPhone(ctx context.Context, col *mongo.Collection, now time.Time, apply bool) error {
	fmt.Printf("\n=== 1) move %s onto %s ===\n", adminPhone, adminEmail)

	admin, err := findOne(ctx, col, bson.D{{Key: "email", Value: adminEmail}})
	if err != nil {
		return fmt.Errorf("finding %s: %w", adminEmail, err)
	}
	if admin == nil {
		fmt.Printf("  ABORT: %s not found\n", adminEmail)
		return errAborted
	}
	fmt.Printf("  admin: id=%s role=%s phone=%s hasPassword=%v\n",
		admin.ID.Hex(), admin.Role, phoneStr(admin.Phone), hasPw(admin))

	// Already correct?
	if admin.Phone != nil && *admin.Phone == adminPhone {
		fmt.Printf("  OK: admin already has %s — nothing to do\n", adminPhone)
		return nil
	}
	if admin.Phone != nil && *admin.Phone != "" {
		fmt.Printf("  ABORT: admin already has a DIFFERENT phone %s — refusing to overwrite\n", *admin.Phone)
		return errAborted
	}

	holder, err := findOne(ctx, col, bson.D{{Key: "phone", Value: adminPhone}})
	if err != nil {
		return fmt.Errorf("finding holder of %s: %w", adminPhone, err)
	}
	if holder != nil {
		if holder.ID == admin.ID {
			fmt.Printf("  OK: admin already holds the phone\n")
			return nil
		}
		fmt.Printf("  holder to free: id=%s email=%q role=%s wallet=%.2f\n",
			holder.ID.Hex(), holder.Email, holder.Role, holder.WalletBalance)
		fmt.Printf("  WILL UNSET phone on holder %s (left without a login; wallet $%.2f stays)\n",
			holder.ID.Hex(), holder.WalletBalance)
		if apply {
			if _, err := col.UpdateByID(ctx, holder.ID, bson.D{{Key: "$unset", Value: bson.D{{Key: "phone", Value: ""}}}}); err != nil {
				return fmt.Errorf("unset holder phone: %w", err)
			}
			fmt.Printf("  UNSET applied.\n")
		}
	} else {
		fmt.Printf("  (phone %s is currently free)\n", adminPhone)
	}

	fmt.Printf("  WILL SET phone=%s on %s\n", adminPhone, adminEmail)
	if apply {
		set := bson.D{{Key: "phone", Value: adminPhone}, {Key: "updatedAt", Value: now}}
		if _, err := col.UpdateByID(ctx, admin.ID, bson.D{{Key: "$set", Value: set}}); err != nil {
			return fmt.Errorf("set admin phone: %w", err)
		}
		fmt.Printf("  SET applied.\n")
	}
	return nil
}

// promote turns the phone-only account holding `phone` into an admin identified
// by `email`, giving it a name and a freshly generated password. Idempotent: if
// `email` already exists it reports and skips (to avoid clobbering).
func promote(ctx context.Context, col *mongo.Collection, now time.Time, apply bool, email, name, phone string) error {
	fmt.Printf("\n=== promote %s -> %s (%s) ===\n", phone, email, name)

	// Guard: the email must not already be taken by another account.
	if existing, err := findOne(ctx, col, bson.D{{Key: "email", Value: email}}); err != nil {
		return fmt.Errorf("email lookup: %w", err)
	} else if existing != nil {
		fmt.Printf("  SKIP: email %s already exists (id=%s role=%s) — not modifying\n",
			email, existing.ID.Hex(), existing.Role)
		return nil
	}

	holder, err := findOne(ctx, col, bson.D{{Key: "phone", Value: phone}})
	if err != nil {
		return fmt.Errorf("phone lookup: %w", err)
	}
	if holder == nil {
		fmt.Printf("  ABORT: no account holds phone %s\n", phone)
		return errAborted
	}
	if holder.Email != "" {
		fmt.Printf("  ABORT: account holding %s already has email %q — refusing\n", phone, holder.Email)
		return errAborted
	}
	fmt.Printf("  holder: id=%s role=%s status=%s wallet=%.2f loyalty=%d\n",
		holder.ID.Hex(), holder.Role, holder.Status, holder.WalletBalance, holder.LoyaltyPoints)

	pw, hash, err := genPassword()
	if err != nil {
		return fmt.Errorf("password gen: %w", err)
	}
	fmt.Printf("  WILL SET: email=%s role=admin name=%q status=active + password (phone %s & wallet kept)\n",
		email, name, phone)
	fmt.Printf("  GENERATED PASSWORD (%s): %s\n", email, pw)
	if apply {
		set := bson.D{
			{Key: "email", Value: email},
			{Key: "name", Value: name},
			{Key: "role", Value: user.RoleAdmin},
			{Key: "status", Value: user.StatusActive},
			{Key: "passwordHash", Value: hash},
			{Key: "updatedAt", Value: now},
		}
		if _, err := col.UpdateByID(ctx, holder.ID, bson.D{{Key: "$set", Value: set}}); err != nil {
			return fmt.Errorf("promote update: %w", err)
		}
		fmt.Printf("  APPLIED. id=%s\n", holder.ID.Hex())
	}
	return nil
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

func hasPw(u *user.User) bool { return u.PasswordHash != nil && *u.PasswordHash != "" }

// genPassword returns a strong random password and its bcrypt hash.
func genPassword() (plain, hash string, err error) {
	b := make([]byte, 18)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b) // ~24 chars, url-safe
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return plain, string(h), nil
}
