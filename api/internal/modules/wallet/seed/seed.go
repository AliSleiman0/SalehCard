// Package seed inserts development wallet-ledger rows so the admin finance
// transactions feed, its type/method filters, and the wallet-top-ups figure
// have signal. The order seed inserts orders directly (bypassing the wallet
// debit), so without this the ledger would only ever hold manual adjustments.
package seed

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
)

// row is one ledger entry to ensure for a seeded user (looked up by email).
type row struct {
	Email   string
	Type    wallet.TxType
	Amount  float64 // signed: credits positive, debits negative
	Method  string
	Ref     string // unique per (user) — used as the idempotency key
	DaysAgo int    // how long ago the entry was recorded (for a varied timeline)
}

// Seed ensures a handful of wallet-ledger rows exist for the seeded accounts:
// top-ups (card / USDT), a wallet purchase, and a refund. Idempotent: each row
// is inserted only when an entry with the same (userId, ref) is absent, so
// re-running the seed neither duplicates nor disturbs real activity. Must run
// after the user seed (it resolves users by email).
func Seed(ctx context.Context, db *mongo.Database) error {
	if err := wallet.EnsureIndexes(ctx, db); err != nil {
		return err
	}

	rows := []row{
		{"customer@salehcard.local", wallet.TxTypeTopUp, 50, "card", "seed:topup-card", 9},
		{"customer@salehcard.local", wallet.TxTypeTopUp, 100, "usdt", "seed:topup-usdt", 6},
		{"customer@salehcard.local", wallet.TxTypePurchase, -9.99, "wallet", "seed:purchase", 4},
		{"customer@salehcard.local", wallet.TxTypeRefund, 9.99, "wallet", "seed:refund", 3},
		{"gamehub.store@salehcard.local", wallet.TxTypeTopUp, 2000, "usdt", "seed:topup-usdt", 14},
		{"gamehub.store@salehcard.local", wallet.TxTypeTopUp, 1000, "card", "seed:topup-card", 7},
		{"topup.pro@salehcard.local", wallet.TxTypeTopUp, 500, "card", "seed:topup-card", 11},
		{"omar.haddad@salehcard.local", wallet.TxTypeTopUp, 12, "card", "seed:topup-card", 2},
		{"lina.khoury@salehcard.local", wallet.TxTypeTopUp, 25, "usdt", "seed:topup-usdt", 1},
	}

	users := db.Collection("users")
	tx := db.Collection("wallet_transactions")
	now := time.Now().UTC()

	for _, rw := range rows {
		var u struct {
			ID            bson.ObjectID `bson:"_id"`
			WalletBalance float64       `bson:"walletBalance"`
		}
		err := users.FindOne(ctx, bson.D{{Key: "email", Value: rw.Email}}).Decode(&u)
		if err == mongo.ErrNoDocuments {
			continue // user not seeded on this box; skip its ledger rows
		}
		if err != nil {
			return err
		}

		exists, err := tx.CountDocuments(ctx, bson.D{
			{Key: "userId", Value: u.ID},
			{Key: "ref", Value: rw.Ref},
		})
		if err != nil {
			return err
		}
		if exists > 0 {
			continue
		}

		_, err = tx.InsertOne(ctx, &wallet.WalletTransaction{
			ID:           bson.NewObjectID(),
			UserID:       u.ID,
			Type:         rw.Type,
			Amount:       rw.Amount,
			BalanceAfter: u.WalletBalance, // display-plausible; the user doc holds the authoritative balance
			Method:       rw.Method,
			Ref:          rw.Ref,
			CreatedAt:    now.AddDate(0, 0, -rw.DaysAgo),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
