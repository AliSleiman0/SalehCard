package reseller

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Service defines the business-logic operations for the reseller domain.
type Service interface {
	GetTier(ctx context.Context, userID bson.ObjectID) (*ResellerTier, error)
	UpdateTier(ctx context.Context, tier *ResellerTier) error
	ApplyPricing(ctx context.Context, userID bson.ObjectID, basePrice float64) (float64, error)
}

// ResellerService is the concrete implementation of Service.
type ResellerService struct {
	repo Repository
}

// NewResellerService constructs a ResellerService backed by the given repository.
func NewResellerService(repo Repository) *ResellerService {
	return &ResellerService{repo: repo}
}

func (s *ResellerService) GetTier(ctx context.Context, userID bson.ObjectID) (*ResellerTier, error) {
	return nil, errors.New("TODO: not implemented")
}

func (s *ResellerService) UpdateTier(ctx context.Context, tier *ResellerTier) error {
	return errors.New("TODO: not implemented")
}

func (s *ResellerService) ApplyPricing(ctx context.Context, userID bson.ObjectID, basePrice float64) (float64, error) {
	return 0, errors.New("TODO: not implemented")
}
