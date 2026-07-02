package wallet

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Service defines the wallet operations shared with other modules (payment
// and refunds for orders). The top-up request queue lives on the concrete
// WalletService — its consumers (wallet handler + admin) hold that type.
type Service interface {
	GetBalance(ctx context.Context, userID bson.ObjectID) (float64, error)
	ListTransactions(ctx context.Context, userID bson.ObjectID) ([]*WalletTransaction, error)
	// Debit atomically charges the wallet, recording a purchase ledger row.
	// Returns ErrInsufficientFunds when the balance cannot cover amount.
	Debit(ctx context.Context, userID bson.ObjectID, amount float64, ref string) (*WalletTransaction, error)
	// Refund credits the wallet back, recording a refund ledger row. Used to
	// compensate a failed order after it has already been charged.
	Refund(ctx context.Context, userID bson.ObjectID, amount float64, ref string) (*WalletTransaction, error)
}

// Top-up request limits: a sanity cap on a single request and a per-user
// pending-queue guard.
const (
	maxTopUpAmount  = 10_000
	maxTopUpNoteLen = 500
	maxPendingTopUp = 3
)

// ErrLedgerWriteFailed reports an approval whose balance change was reversed
// because the mandatory ledger row could not be written; the request was put
// back to pending and can be retried.
var ErrLedgerWriteFailed = &apperrors.AppError{
	Code:    "LEDGER_WRITE_FAILED",
	Message: "the approval was reverted: the ledger row could not be written",
	Err:     errors.New("ledger write failed"),
}

// WalletService is the concrete implementation of Service plus the top-up
// request queue.
type WalletService struct {
	repo   Repository
	topups TopUpStore
}

// NewWalletService constructs a WalletService backed by the given repositories
// (topups may be nil for consumers that only need the payment surface).
func NewWalletService(repo Repository, topups TopUpStore) *WalletService {
	return &WalletService{repo: repo, topups: topups}
}

// CreateTopUpRequest files a pending top-up request (no money moves — the
// wallet is credited only when an admin approves after receiving the funds
// out-of-band).
func (s *WalletService) CreateTopUpRequest(ctx context.Context, userID bson.ObjectID, input TopUpInput) (*TopUpRequest, error) {
	if input.Amount <= 0 {
		return nil, &apperrors.AppError{Code: "BAD_REQUEST", Message: "top-up amount must be positive", Err: apperrors.ErrBadRequest}
	}
	if input.Amount > maxTopUpAmount {
		return nil, &apperrors.AppError{Code: "BAD_REQUEST", Message: "top-up amount exceeds the per-request limit", Err: apperrors.ErrBadRequest}
	}
	channel := strings.ToLower(strings.TrimSpace(input.Channel))
	if !TopUpChannels[channel] {
		return nil, &apperrors.AppError{Code: "BAD_REQUEST", Message: "channel must be one of usdt, whish, omt, cash, other", Err: apperrors.ErrBadRequest}
	}
	note := strings.TrimSpace(input.Note)
	if len(note) > maxTopUpNoteLen {
		return nil, &apperrors.AppError{Code: "BAD_REQUEST", Message: "note is too long", Err: apperrors.ErrBadRequest}
	}
	if pending, err := s.topups.CountPendingForUser(ctx, userID); err != nil {
		return nil, err
	} else if pending >= maxPendingTopUp {
		return nil, &apperrors.AppError{Code: "TOO_MANY_PENDING", Message: "you already have pending top-up requests awaiting review", Err: apperrors.ErrBadRequest}
	}

	req := &TopUpRequest{
		UserID:   userID,
		Amount:   input.Amount,
		Currency: "USD",
		Channel:  channel,
		Note:     note,
	}
	if err := s.topups.Create(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

// ListTopUpRequests returns the user's own request history, newest first.
func (s *WalletService) ListTopUpRequests(ctx context.Context, userID bson.ObjectID) ([]*TopUpRequest, error) {
	return s.topups.FindByUser(ctx, userID)
}

// ApproveTopUpRequest credits the wallet for a pending request. Sequence under
// the no-transactions model: (1) atomically claim pending→approved — the
// double-approve lock; (2) credit the balance, reverting the claim on failure;
// (3) write the mandatory topup ledger row, compensating with a guarded debit
// + claim revert on failure. An "approved" request therefore always implies
// balance AND ledger landed.
func (s *WalletService) ApproveTopUpRequest(ctx context.Context, id bson.ObjectID, decidedBy string) (*TopUpRequest, error) {
	req, err := s.topups.Claim(ctx, id, TopUpApproved, decidedBy, "")
	if err != nil {
		return nil, err
	}

	balance, err := s.repo.Credit(ctx, req.UserID, req.Amount)
	if err != nil {
		if rerr := s.topups.Revert(ctx, id); rerr != nil {
			slog.Error("topup approve: credit failed AND revert failed — request stuck approved without credit",
				"request", id.Hex(), "creditError", err, "revertError", rerr)
		}
		return nil, err
	}

	tx := &WalletTransaction{
		UserID:       req.UserID,
		Type:         TxTypeTopUp,
		Amount:       req.Amount,
		BalanceAfter: balance,
		Method:       req.Channel,
		Ref:          req.ID.Hex(),
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, tx); err != nil {
		_, undoErr := s.repo.Debit(ctx, req.UserID, req.Amount)
		rerr := s.topups.Revert(ctx, id)
		slog.Error("topup approve: ledger insert failed; credit reverted",
			"request", id.Hex(), "amount", req.Amount, "ledgerError", err,
			"revertBalanceError", undoErr, "revertRequestError", rerr)
		return nil, ErrLedgerWriteFailed
	}

	if err := s.topups.SetTxID(ctx, id, tx.ID.Hex()); err != nil {
		slog.Warn("topup approve: failed to stamp txId (best-effort)", "request", id.Hex(), "error", err)
	} else {
		req.TxID = tx.ID.Hex()
	}
	return req, nil
}

// RejectTopUpRequest declines a pending request with a required reason.
func (s *WalletService) RejectTopUpRequest(ctx context.Context, id bson.ObjectID, decidedBy, reason string) (*TopUpRequest, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, &apperrors.AppError{Code: "BAD_REQUEST", Message: "a rejection reason is required", Err: apperrors.ErrBadRequest}
	}
	return s.topups.Claim(ctx, id, TopUpRejected, decidedBy, reason)
}

// GetBalance returns the user's current wallet balance.
func (s *WalletService) GetBalance(ctx context.Context, userID bson.ObjectID) (float64, error) {
	return s.repo.GetBalance(ctx, userID)
}

// ListTransactions returns the user's ledger history, newest first.
func (s *WalletService) ListTransactions(ctx context.Context, userID bson.ObjectID) ([]*WalletTransaction, error) {
	return s.repo.FindByUserID(ctx, userID)
}

// Debit atomically charges the wallet and records a purchase ledger row. The
// ledger row is mandatory: if its insert fails the debit is reversed
// (compensation, per the no-transactions model) before the error is returned,
// so callers can safely retry without double-charging.
func (s *WalletService) Debit(ctx context.Context, userID bson.ObjectID, amount float64, ref string) (*WalletTransaction, error) {
	if amount <= 0 {
		return nil, &apperrors.AppError{Code: "BAD_REQUEST", Message: "charge amount must be positive", Err: apperrors.ErrBadRequest}
	}
	balance, err := s.repo.Debit(ctx, userID, amount)
	if err != nil {
		return nil, err
	}
	tx := &WalletTransaction{
		UserID:       userID,
		Type:         TxTypePurchase,
		Amount:       -amount,
		BalanceAfter: balance,
		Method:       "wallet",
		Ref:          ref,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, tx); err != nil {
		if _, undoErr := s.repo.Credit(ctx, userID, amount); undoErr != nil {
			slog.Error("wallet: debit ledger insert failed AND reversal failed — balance charged without a ledger row",
				"user", userID.Hex(), "amount", amount, "ref", ref, "ledgerError", err, "revertError", undoErr)
		}
		return nil, err
	}
	return tx, nil
}

// Refund credits the wallet back and records a refund ledger row. The ledger
// row is mandatory: if its insert fails the credit is reversed (guarded debit)
// before the error is returned, so callers can safely retry without
// double-crediting.
func (s *WalletService) Refund(ctx context.Context, userID bson.ObjectID, amount float64, ref string) (*WalletTransaction, error) {
	if amount <= 0 {
		return nil, &apperrors.AppError{Code: "BAD_REQUEST", Message: "refund amount must be positive", Err: apperrors.ErrBadRequest}
	}
	balance, err := s.repo.Credit(ctx, userID, amount)
	if err != nil {
		return nil, err
	}
	tx := &WalletTransaction{
		UserID:       userID,
		Type:         TxTypeRefund,
		Amount:       amount,
		BalanceAfter: balance,
		Method:       "wallet",
		Ref:          ref,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, tx); err != nil {
		if _, undoErr := s.repo.Debit(ctx, userID, amount); undoErr != nil {
			slog.Error("wallet: refund ledger insert failed AND reversal failed — balance credited without a ledger row",
				"user", userID.Hex(), "amount", amount, "ref", ref, "ledgerError", err, "revertError", undoErr)
		}
		return nil, err
	}
	return tx, nil
}
