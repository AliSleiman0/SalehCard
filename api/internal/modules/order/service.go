package order

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Service defines the business-logic operations for the order domain.
type Service interface {
	PlaceOrder(ctx context.Context, userID bson.ObjectID, input PlaceOrderInput) (*Order, error)
	GetOrder(ctx context.Context, id bson.ObjectID) (*Order, error)
	ListOrders(ctx context.Context, userID bson.ObjectID) ([]*Order, error)
	CancelOrder(ctx context.Context, id bson.ObjectID) error
}

// OrderService is the concrete implementation of Service.
type OrderService struct {
	repo Repository
}

// NewOrderService constructs an OrderService backed by the given repository.
func NewOrderService(repo Repository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) PlaceOrder(ctx context.Context, userID bson.ObjectID, input PlaceOrderInput) (*Order, error) {
	return nil, errors.New("TODO: not implemented")
}

func (s *OrderService) GetOrder(ctx context.Context, id bson.ObjectID) (*Order, error) {
	return nil, errors.New("TODO: not implemented")
}

func (s *OrderService) ListOrders(ctx context.Context, userID bson.ObjectID) ([]*Order, error) {
	return nil, errors.New("TODO: not implemented")
}

func (s *OrderService) CancelOrder(ctx context.Context, id bson.ObjectID) error {
	return errors.New("TODO: not implemented")
}
