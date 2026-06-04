package user

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Service defines the business-logic operations for the user domain.
type Service interface {
	Register(ctx context.Context, input RegisterInput) (*User, error)
	Login(ctx context.Context, input LoginInput) (token string, err error)
	GetProfile(ctx context.Context, id bson.ObjectID) (*User, error)
	UpdateProfile(ctx context.Context, id bson.ObjectID, input UpdateProfileInput) (*User, error)
}

// UserService is the concrete implementation of Service.
type UserService struct {
	repo Repository
}

// NewUserService constructs a UserService backed by the given repository.
func NewUserService(repo Repository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(ctx context.Context, input RegisterInput) (*User, error) {
	return nil, errors.New("TODO: not implemented")
}

func (s *UserService) Login(ctx context.Context, input LoginInput) (string, error) {
	return "", errors.New("TODO: not implemented")
}

func (s *UserService) GetProfile(ctx context.Context, id bson.ObjectID) (*User, error) {
	return nil, errors.New("TODO: not implemented")
}

func (s *UserService) UpdateProfile(ctx context.Context, id bson.ObjectID, input UpdateProfileInput) (*User, error) {
	return nil, errors.New("TODO: not implemented")
}
