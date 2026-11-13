package order

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
)

// ledgerWallet is a fakeWalletSvc that also serves ledger rows by ref.
type ledgerWallet struct {
	fakeWalletSvc
	rows  []*wallet.WalletTransaction
	err   error
	asked []string
}

func (l *ledgerWallet) LedgerByRefs(_ context.Context, _ bson.ObjectID, refs []string) ([]*wallet.WalletTransaction, error) {
	l.asked = refs
	return l.rows, l.err
}

func TestAttachWalletBalances(t *testing.T) {
	user := bson.NewObjectID()
	paid := &Order{ID: bson.NewObjectID(), UserID: user, PaymentMethod: PaymentMethodWallet}
	refunded := &Order{ID: bson.NewObjectID(), UserID: user, PaymentMethod: PaymentMethodWallet}
	usdt := &Order{ID: bson.NewObjectID(), UserID: user, PaymentMethod: PaymentMethodUSDT}

	lw := &ledgerWallet{rows: []*wallet.WalletTransaction{ // oldest first
		{Type: wallet.TxTypePurchase, Ref: paid.ID.Hex(), Amount: -12.3, BalanceAfter: 87.7},
		{Type: wallet.TxTypePurchase, Ref: refunded.ID.Hex(), Amount: -5, BalanceAfter: 82.7},
		{Type: wallet.TxTypeRefund, Ref: refunded.ID.Hex(), Amount: 5, BalanceAfter: 87.7},
	}}
	s := &OrderService{wallet: lw}
	s.attachWalletBalances(context.Background(), user, []*Order{paid, refunded, usdt})

	if len(lw.asked) != 2 {
		t.Fatalf("only wallet orders are looked up, asked %v", lw.asked)
	}
	if paid.WalletBalanceBefore == nil || *paid.WalletBalanceBefore != 100 || *paid.WalletBalanceAfter != 87.7 {
		t.Errorf("paid: before/after = %v/%v, want 100/87.7", paid.WalletBalanceBefore, paid.WalletBalanceAfter)
	}
	if paid.RefundBalanceAfter != nil {
		t.Error("paid order has no refund")
	}
	if *refunded.WalletBalanceBefore != 87.7 || *refunded.WalletBalanceAfter != 82.7 || refunded.RefundBalanceAfter == nil || *refunded.RefundBalanceAfter != 87.7 {
		t.Errorf("refunded: %v/%v/%v", *refunded.WalletBalanceBefore, *refunded.WalletBalanceAfter, refunded.RefundBalanceAfter)
	}
	if usdt.WalletBalanceBefore != nil || usdt.WalletBalanceAfter != nil {
		t.Error("non-wallet orders get no balances")
	}
}

func TestAttachWalletBalances_BestEffort(t *testing.T) {
	o := &Order{ID: bson.NewObjectID(), PaymentMethod: PaymentMethodWallet}
	(&OrderService{wallet: &ledgerWallet{err: errors.New("down")}}).attachWalletBalances(context.Background(), bson.NewObjectID(), []*Order{o})
	if o.WalletBalanceAfter != nil {
		t.Error("lookup failure must leave balances empty")
	}
	// A wallet without the ledger port (payment-only fake) is fine too.
	(&OrderService{wallet: &fakeWalletSvc{}}).attachWalletBalances(context.Background(), bson.NewObjectID(), []*Order{o})
}
