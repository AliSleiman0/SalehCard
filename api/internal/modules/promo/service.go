package promo

import (
	"context"
	"errors"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// badRequest builds a 400-class AppError with a machine code, for promo
// validation failures the customer should see (expired, below minimum, etc.).
func badRequest(code, msg string) error {
	return &apperrors.AppError{Code: code, Message: msg, Err: apperrors.ErrBadRequest}
}

// Service defines the business-logic operations the order/checkout path needs.
// Admin CRUD goes straight through the repository (see admin.go).
type Service interface {
	// Validate checks a code against an order total and returns the promo when it
	// is redeemable, or a bad-request error explaining why it is not.
	Validate(ctx context.Context, input ValidateInput) (*PromoCode, error)
	// Redeem atomically records one use of a code (the authoritative usage guard).
	Redeem(ctx context.Context, code string) error
}

// PromoService is the concrete implementation of Service.
type PromoService struct {
	repo Repository
}

// NewPromoService constructs a PromoService backed by the given repository.
func NewPromoService(repo Repository) *PromoService {
	return &PromoService{repo: repo}
}

// Validate looks up the code and runs the redeemability checks: it must exist, be
// active, be within its [StartsAt, ExpiresAt] window, not be at its usage cap, and
// the order must meet the minimum. A missing code and every failed check surface
// as a bad-request error (so the customer learns why), never a 404.
func (s *PromoService) Validate(ctx context.Context, input ValidateInput) (*PromoCode, error) {
	p, err := s.repo.FindByCode(ctx, input.Code)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, badRequest("PROMO_INVALID", "promo code not found")
		}
		return nil, err
	}
	if !p.Active {
		return nil, badRequest("PROMO_INACTIVE", "promo code is not active")
	}
	now := time.Now().UTC()
	if p.StartsAt != nil && now.Before(*p.StartsAt) {
		return nil, badRequest("PROMO_NOT_STARTED", "promo code is not active yet")
	}
	if p.ExpiresAt != nil && now.After(*p.ExpiresAt) {
		return nil, badRequest("PROMO_EXPIRED", "promo code has expired")
	}
	if p.MaxUses > 0 && p.Uses >= p.MaxUses {
		return nil, badRequest("PROMO_DEPLETED", "promo code usage limit reached")
	}
	if input.OrderTotal < p.MinOrder {
		return nil, badRequest("PROMO_MIN_ORDER", "order total is below the promo minimum")
	}
	return p, nil
}

// Redeem records one use of a code via the repository's atomic, capped increment.
func (s *PromoService) Redeem(ctx context.Context, code string) error {
	return s.repo.IncrementUses(ctx, code)
}
