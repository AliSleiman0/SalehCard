package wallet

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ForfeitLedger records the ledger row for a wallet balance zeroed by customer
// self-deletion. It satisfies user.ForfeitLedger (wired in server.go); the
// balance itself is zeroed by the user module's guarded anonymizing update, so
// this only writes the row (amount -forfeited, balanceAfter 0).
type ForfeitLedger struct{ repo Repository }

// NewForfeitLedger constructs a ForfeitLedger over the wallet ledger.
func NewForfeitLedger(db *mongo.Database) *ForfeitLedger {
	return &ForfeitLedger{repo: NewMongoRepository(db)}
}

// RecordForfeit inserts the forfeit ledger row for amount (> 0).
func (l *ForfeitLedger) RecordForfeit(ctx context.Context, userID bson.ObjectID, amount float64) error {
	return l.repo.Create(ctx, &WalletTransaction{
		UserID:       userID,
		Type:         TxTypeForfeit,
		Amount:       -amount,
		BalanceAfter: 0,
		Method:       "account_deletion",
		Ref:          "Balance forfeited on account deletion",
	})
}
