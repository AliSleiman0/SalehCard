package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

// minPasswordLen is the minimum accepted password length.
const minPasswordLen = 8

// Service defines the business-logic operations for the user domain.
type Service interface {
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
	Refresh(ctx context.Context, rawToken string) (*AuthResult, error)
	Logout(ctx context.Context, rawToken string) error
	GetProfile(ctx context.Context, id bson.ObjectID) (*User, error)
	UpdateProfile(ctx context.Context, id bson.ObjectID, input UpdateProfileInput) (*User, error)
}

// UserService is the concrete implementation of Service.
type UserService struct {
	repo       Repository
	refresh    RefreshRepository
	secret     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewUserService constructs a UserService with its dependencies and token config.
func NewUserService(repo Repository, refresh RefreshRepository, secret string, accessTTL, refreshTTL time.Duration) *UserService {
	return &UserService{
		repo:       repo,
		refresh:    refresh,
		secret:     secret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Register validates input, hashes the password, creates a customer account, and
// issues an access + refresh token pair.
func (s *UserService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, &apperrors.AppError{Code: "INVALID_EMAIL", Message: "a valid email is required", Err: apperrors.ErrBadRequest}
	}
	if len(input.Password) < minPasswordLen {
		return nil, &apperrors.AppError{Code: "WEAK_PASSWORD", Message: "password must be at least 8 characters", Err: apperrors.ErrBadRequest}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashed := string(hash)

	locale := input.Locale
	if locale == "" {
		locale = "en"
	}

	user := &User{
		Email:          email,
		PasswordHash:   &hashed,
		Role:           RoleCustomer,
		Locale:         locale,
		SavedPlayerIDs: []string{},
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err // ErrConflict on duplicate email
	}

	return s.issueTokens(ctx, user)
}

// Login verifies credentials and issues a token pair. It returns a uniform
// ErrUnauthorized for both unknown emails and bad passwords (no user enumeration).
func (s *UserService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	user, err := s.repo.FindByEmail(ctx, input.Email)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return nil, apperrors.ErrUnauthorized
		}
		return nil, err
	}
	if user.PasswordHash == nil {
		return nil, apperrors.ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, apperrors.ErrUnauthorized
	}
	return s.issueTokens(ctx, user)
}

// Refresh validates the presented refresh token, rotates it (revoke old, issue
// new), and returns a fresh token pair.
func (s *UserService) Refresh(ctx context.Context, rawToken string) (*AuthResult, error) {
	if rawToken == "" {
		return nil, apperrors.ErrUnauthorized
	}
	rec, err := s.refresh.FindByHash(ctx, hashToken(rawToken))
	if err != nil {
		if err == apperrors.ErrNotFound {
			return nil, apperrors.ErrUnauthorized
		}
		return nil, err
	}
	if rec.RevokedAt != nil || time.Now().After(rec.ExpiresAt) {
		return nil, apperrors.ErrUnauthorized
	}

	user, err := s.repo.FindByID(ctx, rec.UserID)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return nil, apperrors.ErrUnauthorized
		}
		return nil, err
	}

	// Rotate: revoke the consumed token before minting a replacement.
	if err := s.refresh.Revoke(ctx, rec.ID); err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, user)
}

// Logout revokes the presented refresh token. Unknown/already-revoked tokens are
// treated as success (idempotent).
func (s *UserService) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	rec, err := s.refresh.FindByHash(ctx, hashToken(rawToken))
	if err != nil {
		if err == apperrors.ErrNotFound {
			return nil
		}
		return err
	}
	return s.refresh.Revoke(ctx, rec.ID)
}

// GetProfile returns the user identified by id.
func (s *UserService) GetProfile(ctx context.Context, id bson.ObjectID) (*User, error) {
	return s.repo.FindByID(ctx, id)
}

// UpdateProfile applies the supplied mutable fields and returns the updated user.
func (s *UserService) UpdateProfile(ctx context.Context, id bson.ObjectID, input UpdateProfileInput) (*User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Locale != nil {
		user.Locale = *input.Locale
	}
	if input.SavedPlayerIDs != nil {
		user.SavedPlayerIDs = input.SavedPlayerIDs
	}
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// issueTokens mints an access JWT and a fresh, persisted refresh token for user.
func (s *UserService) issueTokens(ctx context.Context, user *User) (*AuthResult, error) {
	access, err := auth.IssueAccessToken(s.secret, auth.Claims{
		UserID: user.ID.Hex(),
		Email:  user.Email,
		Role:   string(user.Role),
	}, s.accessTTL)
	if err != nil {
		return nil, err
	}

	raw, err := newOpaqueToken()
	if err != nil {
		return nil, err
	}
	rec := &RefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(raw),
		ExpiresAt: time.Now().Add(s.refreshTTL).UTC(),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.refresh.Create(ctx, rec); err != nil {
		return nil, err
	}

	return &AuthResult{User: user, AccessToken: access, RefreshToken: raw}, nil
}

// newOpaqueToken returns a 32-byte cryptographically random, URL-safe token.
func newOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken returns the hex-encoded SHA-256 of a raw refresh token; only the hash
// is ever persisted.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
