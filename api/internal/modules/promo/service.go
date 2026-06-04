package promo

import (
	"context"
	"errors"
)

// Service defines the business-logic operations for the promo domain.
type Service interface {
	Validate(ctx context.Context, input ValidateInput) (*PromoCode, error)
	Apply(ctx context.Context, input ValidateInput) (discount float64, err error)
	Create(ctx context.Context, promo *PromoCode) error
}

// PromoService is the concrete implementation of Service.
type PromoService struct {
	repo Repository
}

// NewPromoService constructs a PromoService backed by the given repository.
func NewPromoService(repo Repository) *PromoService {
	return &PromoService{repo: repo}
}

func (s *PromoService) Validate(ctx context.Context, input ValidateInput) (*PromoCode, error) {
	return nil, errors.New("TODO: not implemented")
}

func (s *PromoService) Apply(ctx context.Context, input ValidateInput) (float64, error) {
	return 0, errors.New("TODO: not implemented")
}

func (s *PromoService) Create(ctx context.Context, promo *PromoCode) error {
	return errors.New("TODO: not implemented")
}
