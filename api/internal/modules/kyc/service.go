package kyc

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// Service defines the customer-facing KYC operations. Admin moderation goes
// straight through the repository (see admin.go).
type Service interface {
	Submit(ctx context.Context, userID bson.ObjectID, in SubmitInput) (*Profile, error)
	GetProfile(ctx context.Context, userID bson.ObjectID) (*Profile, error)
}

// KycService is the concrete implementation of Service.
type KycService struct {
	repo Repository
}

// NewService constructs a KycService backed by the given repository.
func NewService(repo Repository) *KycService {
	return &KycService{repo: repo}
}

// Submit validates the background info and stores it pending moderation. One
// submission per user — re-submitting replaces the prior one and re-enters the
// queue (any earlier approval/rejection is cleared).
func (s *KycService) Submit(ctx context.Context, userID bson.ObjectID, in SubmitInput) (*Profile, error) {
	sub := &Submission{
		UserID:           userID,
		FullName:         strings.TrimSpace(in.FullName),
		DateOfBirth:      strings.TrimSpace(in.DateOfBirth),
		PlaceOfBirth:     strings.TrimSpace(in.PlaceOfBirth),
		PlaceOfResidence: strings.TrimSpace(in.PlaceOfResidence),
		DocumentType:     strings.TrimSpace(in.DocumentType),
		DocumentNumber:   strings.TrimSpace(in.DocumentNumber),
	}
	if sub.FullName == "" {
		return nil, badRequest("full name is required")
	}
	if _, err := time.Parse("2006-01-02", sub.DateOfBirth); err != nil {
		return nil, badRequest("date of birth must be a valid date (YYYY-MM-DD)")
	}
	if sub.PlaceOfBirth == "" {
		return nil, badRequest("place of birth is required")
	}
	if sub.PlaceOfResidence == "" {
		return nil, badRequest("place of residence is required")
	}
	switch sub.DocumentType {
	case DocPassport, DocIDCard, DocLicense:
	default:
		return nil, badRequest("document type must be passport, id_card, or license")
	}
	if sub.DocumentNumber == "" {
		return nil, badRequest("document number is required")
	}

	saved, err := s.repo.Upsert(ctx, sub)
	if err != nil {
		return nil, err
	}
	return profileFor(saved), nil
}

// GetProfile returns the user's derived KYC profile (unverified when they have
// no submission).
func (s *KycService) GetProfile(ctx context.Context, userID bson.ObjectID) (*Profile, error) {
	sub, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return profileFor(nil), nil
		}
		return nil, err
	}
	return profileFor(sub), nil
}

// badRequest builds a 400-classified validation error.
func badRequest(msg string) error {
	return &apperrors.AppError{Code: "BAD_REQUEST", Message: msg, Err: apperrors.ErrBadRequest}
}
