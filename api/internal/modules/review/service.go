package review

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Service defines the business-logic operations for the review domain.
type Service interface {
	Create(ctx context.Context, userID bson.ObjectID, input CreateReviewInput) (*Review, error)
	ListByProduct(ctx context.Context, productID bson.ObjectID) ([]*Review, error)
	Delete(ctx context.Context, id bson.ObjectID) error
}

// ReviewService is the concrete implementation of Service.
type ReviewService struct {
	repo Repository
}

// NewReviewService constructs a ReviewService backed by the given repository.
func NewReviewService(repo Repository) *ReviewService {
	return &ReviewService{repo: repo}
}

func (s *ReviewService) Create(ctx context.Context, userID bson.ObjectID, input CreateReviewInput) (*Review, error) {
	return nil, errors.New("TODO: not implemented")
}

func (s *ReviewService) ListByProduct(ctx context.Context, productID bson.ObjectID) ([]*Review, error) {
	return nil, errors.New("TODO: not implemented")
}

func (s *ReviewService) Delete(ctx context.Context, id bson.ObjectID) error {
	return errors.New("TODO: not implemented")
}
