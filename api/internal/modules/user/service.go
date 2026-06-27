package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/sms"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

// minPasswordLen is the minimum accepted password length.
const minPasswordLen = 8

// ErrAccountSuspended is returned by the auth flows when a suspended account
// tries to sign in. It wraps ErrForbidden so the handler maps it to 403.
var ErrAccountSuspended = &apperrors.AppError{
	Code:    "ACCOUNT_SUSPENDED",
	Message: "this account has been suspended",
	Err:     apperrors.ErrForbidden,
}

// isSuspended reports whether an account is suspended. An empty status (legacy
// accounts predating the field) counts as active.
func isSuspended(u *User) bool {
	return u.Status == StatusSuspended
}

// e164 matches a normalized international phone number.
var e164 = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

// OTPConfig groups the tunables for the one-time-passcode flow.
type OTPConfig struct {
	CountryCode    string        // prefixed to local numbers, e.g. "+961"
	Length         int           // digits in a generated code
	TTL            time.Duration // how long a code stays valid
	ResendInterval time.Duration // minimum wait between requests for one number
	MaxAttempts    int           // wrong guesses before a code locks
}

// Service defines the business-logic operations for the user domain.
type Service interface {
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
	LoginByPhone(ctx context.Context, input PhoneLoginInput) (*AuthResult, error)
	RequestOTP(ctx context.Context, input RequestOTPInput) error
	VerifyOTP(ctx context.Context, input VerifyOTPInput) (*AuthResult, error)
	Refresh(ctx context.Context, rawToken string) (*AuthResult, error)
	Logout(ctx context.Context, rawToken string) error
	GetProfile(ctx context.Context, id bson.ObjectID) (*User, error)
	UpdateProfile(ctx context.Context, id bson.ObjectID, input UpdateProfileInput) (*User, error)
}

// UserService is the concrete implementation of Service.
type UserService struct {
	repo       Repository
	refresh    RefreshRepository
	otp        OTPRepository
	sender     sms.Sender
	otpCfg     OTPConfig
	secret     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewUserService constructs a UserService with its dependencies and token config.
func NewUserService(repo Repository, refresh RefreshRepository, otp OTPRepository, sender sms.Sender, otpCfg OTPConfig, secret string, accessTTL, refreshTTL time.Duration) *UserService {
	return &UserService{
		repo:       repo,
		refresh:    refresh,
		otp:        otp,
		sender:     sender,
		otpCfg:     otpCfg,
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
		Name:           strings.TrimSpace(input.Name),
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
	if isSuspended(user) {
		return nil, ErrAccountSuspended
	}
	return s.issueTokens(ctx, user)
}

// LoginByPhone verifies a phone+password pair and issues a token pair. Like
// Login, it returns a uniform ErrUnauthorized for unknown numbers, accounts
// without a password, and bad passwords (no user enumeration).
func (s *UserService) LoginByPhone(ctx context.Context, input PhoneLoginInput) (*AuthResult, error) {
	phone, err := s.normalizePhone(input.Phone)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.FindByPhone(ctx, phone)
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
	if isSuspended(user) {
		return nil, ErrAccountSuspended
	}
	return s.issueTokens(ctx, user)
}

// RequestOTP generates a one-time code for a phone number, stores its hash, and
// sends it via the SMS sender. It throttles repeat requests per number. The
// response never reveals whether an account exists for the number.
func (s *UserService) RequestOTP(ctx context.Context, input RequestOTPInput) error {
	phone, err := s.normalizePhone(input.Phone)
	if err != nil {
		return err
	}

	if existing, err := s.otp.FindByPhone(ctx, phone); err == nil {
		if time.Since(existing.CreatedAt) < s.otpCfg.ResendInterval {
			return &apperrors.AppError{Code: "OTP_THROTTLED", Message: "please wait before requesting another code", Err: apperrors.ErrBadRequest}
		}
	}

	code, err := randomNumericCode(s.otpCfg.Length)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := s.otp.Upsert(ctx, &OtpCode{
		Phone:     phone,
		CodeHash:  hashToken(code),
		ExpiresAt: now.Add(s.otpCfg.TTL),
		CreatedAt: now,
	}); err != nil {
		return err
	}

	msg := fmt.Sprintf("Your SalehCard verification code is %s", code)
	return s.sender.Send(ctx, phone, msg)
}

// VerifyOTP validates a code for a phone number and, on success, finds or creates
// the matching account and issues a token pair. A non-empty Password on a new (or
// password-less) account sets the password, enabling later phone+password login.
func (s *UserService) VerifyOTP(ctx context.Context, input VerifyOTPInput) (*AuthResult, error) {
	phone, err := s.normalizePhone(input.Phone)
	if err != nil {
		return nil, err
	}

	rec, err := s.otp.FindByPhone(ctx, phone)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return nil, &apperrors.AppError{Code: "OTP_INVALID", Message: "invalid or expired code", Err: apperrors.ErrUnauthorized}
		}
		return nil, err
	}
	if rec.ConsumedAt != nil || time.Now().After(rec.ExpiresAt) {
		return nil, &apperrors.AppError{Code: "OTP_EXPIRED", Message: "this code has expired, request a new one", Err: apperrors.ErrUnauthorized}
	}
	if rec.Attempts >= s.otpCfg.MaxAttempts {
		return nil, &apperrors.AppError{Code: "OTP_LOCKED", Message: "too many attempts, request a new code", Err: apperrors.ErrUnauthorized}
	}
	if hashToken(strings.TrimSpace(input.Code)) != rec.CodeHash {
		_ = s.otp.IncrementAttempts(ctx, phone)
		return nil, &apperrors.AppError{Code: "OTP_INVALID", Message: "invalid or expired code", Err: apperrors.ErrUnauthorized}
	}

	// Code is good — consume it so it can't be replayed.
	_ = s.otp.DeleteByPhone(ctx, phone)

	user, err := s.repo.FindByPhone(ctx, phone)
	if err != nil {
		if err != apperrors.ErrNotFound {
			return nil, err
		}
		// First sign-in for this number: create a phone-only customer account.
		user = &User{
			Name:           strings.TrimSpace(input.Name),
			Phone:          &phone,
			Role:           RoleCustomer,
			Locale:         "en",
			SavedPlayerIDs: []string{},
		}
		if err := s.repo.Create(ctx, user); err != nil {
			return nil, err
		}
	}

	if isSuspended(user) {
		return nil, ErrAccountSuspended
	}

	// Optionally set a password (signup flow) when the account has none yet.
	if input.Password != "" && user.PasswordHash == nil {
		if len(input.Password) < minPasswordLen {
			return nil, &apperrors.AppError{Code: "WEAK_PASSWORD", Message: "password must be at least 8 characters", Err: apperrors.ErrBadRequest}
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		if err := s.repo.SetPassword(ctx, user.ID, string(hash)); err != nil {
			return nil, err
		}
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

	if isSuspended(user) {
		return nil, ErrAccountSuspended
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

	// Activity heartbeat for the "active users" metric. Best-effort: a failure
	// here must not fail the login/refresh, so the error is ignored.
	now := time.Now().UTC()
	_ = s.repo.TouchLastSeen(ctx, user.ID, now)
	user.LastSeen = now

	return &AuthResult{User: user, AccessToken: access, RefreshToken: raw}, nil
}

// normalizePhone converts a user-entered number to E.164. It accepts an existing
// "+" prefix, "00" international prefix, or a local number (optionally with a
// leading 0) to which the configured default country code is prepended.
func (s *UserService) normalizePhone(raw string) (string, error) {
	// Keep digits and a leading "+"; drop spaces, dashes, parentheses, etc.
	var b strings.Builder
	for i, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && i == 0:
			b.WriteRune(r)
		}
	}
	p := b.String()

	switch {
	case strings.HasPrefix(p, "+"):
		// already international
	case strings.HasPrefix(p, "00"):
		p = "+" + strings.TrimPrefix(p, "00")
	default:
		p = s.otpCfg.CountryCode + strings.TrimPrefix(p, "0")
	}

	if !e164.MatchString(p) {
		return "", &apperrors.AppError{Code: "INVALID_PHONE", Message: "a valid mobile number is required", Err: apperrors.ErrBadRequest}
	}
	return p, nil
}

// randomNumericCode returns an n-digit numeric code (zero-padded), drawn from a
// cryptographically secure source.
func randomNumericCode(n int) (string, error) {
	if n <= 0 {
		n = 6
	}
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	v, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", n, v), nil
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
