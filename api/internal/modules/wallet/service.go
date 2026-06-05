package wallet

import (
	"context"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Service defines the business-logic operations for the wallet domain.
type Service interface {
	TopUp(ctx context.Context, userID bson.ObjectID, input TopUpInput) (*WalletTransaction, error)
	GetBalance(ctx context.Context, userID bson.ObjectID) (float64, error)
	ListTransactions(ctx context.Context, userID bson.ObjectID) ([]*WalletTransaction, error)
	// Debit atomically charges the wallet, recording a purchase ledger row.
	// Returns ErrInsufficientFunds when the balance cannot cover amount.
	Debit(ctx context.Context, userID bson.ObjectID, amount float64, ref string) (*WalletTransaction, error)
	// Refund credits the wallet back, recording a refund ledger row. Used to
	// compensate a failed order after it has already been charged.
	Refund(ctx context.Context, userID bson.ObjectID, amount float64, ref string) (*WalletTransaction, error)
}

// WalletService is the concrete implementation of Service.
type WalletService struct {
	repo Repository
}

// NewWalletService constructs a WalletService backed by the given repository.
func NewWalletService(repo Repository) *WalletService {
	return &WalletService{repo: repo}
}

// TopUp adds funds to a wallet (mock-approved, no payment gateway) and records a
// topup ledger row.
func (s *WalletService) TopUp(ctx context.Context, userID bson.ObjectID, input TopUpInput) (*WalletTransaction, error) {
	if input.Amount <= 0 {
		return nil, &apperrors.AppError{Code: "BAD_REQUEST", Message: "top-up amount must be positive", Err: apperrors.ErrBadRequest}
	}
	balance, err := s.repo.Credit(ctx, userID, input.Amount)
	if err != nil {
		return nil, err
	}
	tx := &WalletTransaction{
		UserID:       userID,
		Type:         TxTypeTopUp,
		Amount:       input.Amount,
		BalanceAfter: balance,
		Method:       input.Method,
		Ref:          input.Ref,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// GetBalance returns the user's current wallet balance.
func (s *WalletService) GetBalance(ctx context.Context, userID bson.ObjectID) (float64, error) {
	return s.repo.GetBalance(ctx, userID)
}

// ListTransactions returns the user's ledger history, newest first.
func (s *WalletService) ListTransactions(ctx context.Context, userID bson.ObjectID) ([]*WalletTransaction, error) {
	return s.repo.FindByUserID(ctx, userID)
}

// Debit atomically charges the wallet and records a purchase ledger row.
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
		return nil, err
	}
	return tx, nil
}

// Refund credits the wallet back and records a refund ledger row.
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
		return nil, err
	}
	return tx, nil
}
