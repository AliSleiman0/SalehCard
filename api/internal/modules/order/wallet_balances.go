package order

import (
	"context"
	"log/slog"
	"math"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
)

// walletLedger is the optional read port for an order's wallet ledger rows
// (satisfied by *wallet.WalletService). Kept off wallet.Service so payment-only
// fakes don't have to implement it; when absent, balances are simply omitted.
type walletLedger interface {
	LedgerByRefs(ctx context.Context, userID bson.ObjectID, refs []string) ([]*wallet.WalletTransaction, error)
}

// attachWalletBalances fills the response-only balance fields on wallet-paid
// orders from their ledger rows: the purchase row's BalanceAfter is the
// balance after the charge (before = after + charged amount) and the latest
// refund row's BalanceAfter the balance after the refund. Works for historic
// orders too (every wallet charge has always written a ref'd ledger row).
// Best-effort: a lookup failure only drops the extra fields.
func (s *OrderService) attachWalletBalances(ctx context.Context, userID bson.ObjectID, orders []*Order) {
	ledger, ok := s.wallet.(walletLedger)
	if !ok || len(orders) == 0 {
		return
	}
	refs := make([]string, 0, len(orders))
	for _, o := range orders {
		if o.PaymentMethod == PaymentMethodWallet {
			refs = append(refs, o.ID.Hex())
		}
	}
	if len(refs) == 0 {
		return
	}
	rows, err := ledger.LedgerByRefs(ctx, userID, refs)
	if err != nil {
		slog.Warn("order: wallet balance lookup failed", "error", err)
		return
	}
	purchase := map[string]*wallet.WalletTransaction{}
	refund := map[string]*wallet.WalletTransaction{}
	for _, tx := range rows { // oldest first
		switch tx.Type {
		case wallet.TxTypePurchase:
			if _, seen := purchase[tx.Ref]; !seen { // the original charge
				purchase[tx.Ref] = tx
			}
		case wallet.TxTypeRefund:
			refund[tx.Ref] = tx // latest refund wins
		}
	}
	for _, o := range orders {
		ref := o.ID.Hex()
		if tx := purchase[ref]; tx != nil {
			after := roundCents(tx.BalanceAfter)
			before := roundCents(tx.BalanceAfter + math.Abs(tx.Amount))
			o.WalletBalanceAfter = &after
			o.WalletBalanceBefore = &before
		}
		if tx := refund[ref]; tx != nil {
			after := roundCents(tx.BalanceAfter)
			o.RefundBalanceAfter = &after
		}
	}
}

// roundCents rounds to 2 decimals (float sums like 12.3+0.1 drift).
func roundCents(v float64) float64 { return math.Round(v*100) / 100 }
