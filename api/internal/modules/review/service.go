package review

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// maxBodyLen bounds a review comment to a sane length.
const maxBodyLen = 2000

// Service defines the business-logic operations for the review domain.
type Service interface {
	Create(ctx context.Context, userID bson.ObjectID, input CreateReviewInput) (*Review, error)
}

// ReviewService is the concrete implementation of Service.
type ReviewService struct {
	repo Repository
}

// NewReviewService constructs a ReviewService backed by the given repository.
func NewReviewService(repo Repository) *ReviewService {
	return &ReviewService{repo: repo}
}

// Create validates and stores a customer's review. New reviews are pending
// moderation; an admin approves or rejects them before they count toward a
// product's rating. VerifiedPurchase is left false here (deriving it from order
// history is a noted follow-up).
func (s *ReviewService) Create(ctx context.Context, userID bson.ObjectID, input CreateReviewInput) (*Review, error) {
	if input.ProductID.IsZero() {
		return nil, badRequest("productId is required")
	}
	if input.Rating < 1 || input.Rating > 5 {
		return nil, badRequest("rating must be between 1 and 5")
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return nil, badRequest("body is required")
	}
	if len(body) > maxBodyLen {
		return nil, badRequest("body is too long")
	}

	rv := &Review{
		ProductID:        input.ProductID,
		UserID:           userID,
		Rating:           input.Rating,
		Body:             body,
		VerifiedPurchase: false,
		Status:           StatusPending,
		CreatedAt:        time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, rv); err != nil {
		return nil, err
	}
	return rv, nil
}

// badRequest builds a 400-classified validation error.
func badRequest(msg string) error {
	return &apperrors.AppError{Code: "BAD_REQUEST", Message: msg, Err: apperrors.ErrBadRequest}
}
