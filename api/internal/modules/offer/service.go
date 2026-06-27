package offer

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Service defines the business-logic operation the order/checkout path needs.
// Admin CRUD and the public listing go straight through the repository (see
// admin.go / handler.go).
type Service interface {
	// FindLiveByProduct returns the live offer for a product (or ErrNotFound when
	// the product has none), so the order engine can re-price at the sale price.
	FindLiveByProduct(ctx context.Context, productID bson.ObjectID, now time.Time) (*Offer, error)
}

// OfferService is the concrete implementation of Service.
type OfferService struct {
	repo Repository
}

// NewOfferService constructs an OfferService backed by the given repository.
func NewOfferService(repo Repository) *OfferService {
	return &OfferService{repo: repo}
}

// FindLiveByProduct delegates to the repository's live lookup.
func (s *OfferService) FindLiveByProduct(ctx context.Context, productID bson.ObjectID, now time.Time) (*Offer, error) {
	return s.repo.FindLiveByProduct(ctx, productID, now)
}
