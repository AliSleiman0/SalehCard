package wallet

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Service defines the business-logic operations for the wallet domain.
type Service interface {
	TopUp(ctx context.Context, userID bson.ObjectID, input TopUpInput) (*WalletTransaction, error)
	GetBalance(ctx context.Context, userID bson.ObjectID) (float64, error)
	ListTransactions(ctx context.Context, userID bson.ObjectID) ([]*WalletTransaction, error)
}

// WalletService is the concrete implementation of Service.
type WalletService struct {
	repo Repository
}

// NewWalletService constructs a WalletService backed by the given repository.
func NewWalletService(repo Repository) *WalletService {
	return &WalletService{repo: repo}
}

func (s *WalletService) TopUp(ctx context.Context, userID bson.ObjectID, input TopUpInput) (*WalletTransaction, error) {
	return nil, errors.New("TODO: not implemented")
}

func (s *WalletService) GetBalance(ctx context.Context, userID bson.ObjectID) (float64, error) {
	return 0, errors.New("TODO: not implemented")
}

func (s *WalletService) ListTransactions(ctx context.Context, userID bson.ObjectID) ([]*WalletTransaction, error) {
	return nil, errors.New("TODO: not implemented")
}
