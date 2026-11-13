package kyc

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/photoupload"
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
	// docOK verifies a document URL is one of OUR kyc/ uploads (host +
	// keyspace when the store can report its public URLs); nil = path shape.
	docOK photoupload.Checker
}

// WithDocChecker pins submitted document URLs to the given checker (built
// from the blob store, so an external host with a look-alike path fails).
func (s *KycService) WithDocChecker(c photoupload.Checker) *KycService {
	s.docOK = c
	return s
}

func (s *KycService) validDoc(u string) bool {
	if s.docOK != nil {
		return s.docOK(u)
	}
	return validDocURL(u)
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
		DocumentFrontURL: strings.TrimSpace(in.DocumentFrontURL),
		DocumentBackURL:  strings.TrimSpace(in.DocumentBackURL),
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
	if !s.validDoc(sub.DocumentFrontURL) {
		return nil, badRequest("a photo of the front of your document is required")
	}
	// Passports are single-sided; every other document needs both sides.
	if sub.DocumentType != DocPassport && sub.DocumentBackURL == "" {
		return nil, badRequest("a photo of the back of your document is required")
	}
	if sub.DocumentBackURL != "" && !s.validDoc(sub.DocumentBackURL) {
		return nil, badRequest("the back document photo URL is invalid")
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

// validDocURL accepts only URLs shaped like our own document uploads (POST
// /api/v1/kyc/documents stores under a kyc/ key as re-encoded JPEG on both the
// azure and local adapters). Not ownership-proof — admins review the images
// visually — but it rejects junk and off-keyspace URLs.
func validDocURL(s string) bool { return photoupload.IsStoredURL(s, "kyc/") }

// badRequest builds a 400-classified validation error.
func badRequest(msg string) error {
	return &apperrors.AppError{Code: "BAD_REQUEST", Message: msg, Err: apperrors.ErrBadRequest}
}
