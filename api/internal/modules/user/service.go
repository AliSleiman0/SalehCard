package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
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

// isDeleted reports whether an account has been soft-deleted (anonymized).
// Auth call sites check isSuspended before isDeleted so a suspended account
// keeps its explicit 403 ACCOUNT_SUSPENDED, while a deleted one gets a plain
// 401 — deletion is never revealed to a lingering credential/token holder.
func isDeleted(u *User) bool {
	return u.Status == StatusDeleted
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
	VerifyAdmin2FA(ctx context.Context, input VerifyTwoFactorInput) (*AuthResult, error)
	ResendAdmin2FA(ctx context.Context, input ResendTwoFactorInput) (*AuthResult, error)
	Refresh(ctx context.Context, rawToken string) (*AuthResult, error)
	Logout(ctx context.Context, rawToken string) error
	GetProfile(ctx context.Context, id bson.ObjectID) (*User, error)
	UpdateProfile(ctx context.Context, id bson.ObjectID, input UpdateProfileInput) (*User, error)
	// DeleteAccount permanently retires the caller's own account (see
	// delete.go); hadKyc reports whether a KYC submission was purged with it.
	DeleteAccount(ctx context.Context, id bson.ObjectID) (hadKyc bool, err error)
}

// settingsSource reads the app-settings singleton (kept narrow so the service can
// be unit-tested without Mongo). Mirrors loyalty.configSource.
type settingsSource interface {
	Get(ctx context.Context) (*settings.Settings, error)
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
	// settings, when set (via WithSettings), enables admin SMS 2FA by letting the
	// service read the AdminSmsTwoFactorEnabled flag at login time. Nil => 2FA off.
	settings settingsSource
	// rolePerms, when set (via WithRolePerms), resolves a custom admin role's
	// permission set at token-issue time. Nil => admins with a custom role get no
	// permissions (fail-closed); super admins (nil AdminRoleID) are unaffected.
	rolePerms func(ctx context.Context, roleID bson.ObjectID) ([]string, error)
	// deletion holds the cross-module ports account deletion depends on (via
	// WithDeletionPorts). Unwired ports => DeleteAccount fails closed.
	deletion DeletionPorts
}

// UserServiceOption configures optional UserService dependencies.
type UserServiceOption func(*UserService)

// WithSettings gives the service a read handle on the app-settings singleton,
// enabling the admin SMS-2FA gate. Without it, admin 2FA is treated as disabled
// (so existing call sites and tests keep their password-only behavior).
func WithSettings(src settingsSource) UserServiceOption {
	return func(s *UserService) { s.settings = src }
}

// WithRolePerms gives the service a resolver from a custom admin role to its
// permission set, consulted when issuing tokens for admins with an assigned
// role. Without it those admins get an empty permission set (fail-closed).
func WithRolePerms(resolve func(ctx context.Context, roleID bson.ObjectID) ([]string, error)) UserServiceOption {
	return func(s *UserService) { s.rolePerms = resolve }
}

// NewUserService constructs a UserService with its dependencies and token config.
func NewUserService(repo Repository, refresh RefreshRepository, otp OTPRepository, sender sms.Sender, otpCfg OTPConfig, secret string, accessTTL, refreshTTL time.Duration, opts ...UserServiceOption) *UserService {
	s := &UserService{
		repo:       repo,
		refresh:    refresh,
		otp:        otp,
		sender:     sender,
		otpCfg:     otpCfg,
		secret:     secret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
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
		SavedPlayerIDs: []SavedPlayerID{},
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
	if isDeleted(user) {
		return nil, apperrors.ErrUnauthorized
	}
	return s.completeLogin(ctx, user)
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
	if isDeleted(user) {
		return nil, apperrors.ErrUnauthorized
	}
	return s.completeLogin(ctx, user)
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

	if err := s.verifyOTPCode(ctx, phone, input.Code); err != nil {
		return nil, err
	}

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
			SavedPlayerIDs: []SavedPlayerID{},
		}
		if err := s.repo.Create(ctx, user); err != nil {
			return nil, err
		}
	}

	if isSuspended(user) {
		return nil, ErrAccountSuspended
	}
	if isDeleted(user) {
		return nil, apperrors.ErrUnauthorized
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
	if isDeleted(user) {
		return nil, apperrors.ErrUnauthorized
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
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// An access token can outlive a self-deletion; never serve the anonymized doc.
	if isDeleted(user) {
		return nil, apperrors.ErrUnauthorized
	}
	return user, nil
}

// UpdateProfile applies the supplied mutable fields and returns the updated user.
func (s *UserService) UpdateProfile(ctx context.Context, id bson.ObjectID, input UpdateProfileInput) (*User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// An access token can outlive a self-deletion; never mutate the anonymized doc.
	if isDeleted(user) {
		return nil, apperrors.ErrUnauthorized
	}
	if input.Locale != nil {
		user.Locale = *input.Locale
	}
	if input.SavedPlayerIDs != nil {
		cleaned := make([]SavedPlayerID, 0, len(input.SavedPlayerIDs))
		for _, p := range input.SavedPlayerIDs {
			label := strings.TrimSpace(p.Label)
			value := strings.TrimSpace(p.Value)
			if label == "" || value == "" {
				return nil, &apperrors.AppError{Code: "INVALID_PLAYER_ID", Message: "each saved player ID needs a label and a value", Err: apperrors.ErrBadRequest}
			}
			cleaned = append(cleaned, SavedPlayerID{Label: label, Value: value})
		}
		user.SavedPlayerIDs = cleaned
	}
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// issueTokens mints an access JWT and a fresh, persisted refresh token for user.
// For admins it resolves the RBAC permission set (from the assigned custom role,
// or the "*" wildcard for super admins) into both the JWT claims and the user
// payload — so role edits take effect on the holder's next refresh/login.
func (s *UserService) issueTokens(ctx context.Context, user *User) (*AuthResult, error) {
	var phone string
	if user.Phone != nil {
		phone = *user.Phone
	}
	perms := s.resolvePerms(ctx, user)
	user.Permissions = perms
	access, err := auth.IssueAccessToken(s.secret, auth.Claims{
		UserID: user.ID.Hex(),
		Email:  user.Email,
		Phone:  phone,
		Role:   string(user.Role),
		Perms:  perms,
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

// resolvePerms returns the RBAC permission set for user: nil for non-admins,
// the "*" wildcard for super admins (no custom role assigned), and the assigned
// role's permissions otherwise. A missing role or resolver failure yields an
// empty set (fail-closed: the admin signs in but can access nothing) rather
// than blocking the login.
func (s *UserService) resolvePerms(ctx context.Context, user *User) []string {
	if user.Role != RoleAdmin {
		return nil
	}
	if user.AdminRoleID == nil {
		return []string{auth.PermAll}
	}
	if s.rolePerms == nil {
		return []string{}
	}
	perms, err := s.rolePerms(ctx, *user.AdminRoleID)
	if err != nil {
		slog.Warn("user: could not resolve admin role permissions; issuing none",
			"userId", user.ID.Hex(), "roleId", user.AdminRoleID.Hex(), "err", err)
		return []string{}
	}
	return perms
}

// ErrAdmin2FANoPhone is returned when admin SMS 2FA is enabled but the admin has
// no phone on file, so no second-factor code can be delivered. It wraps
// ErrForbidden (403): the login fails closed rather than silently bypassing 2FA.
var ErrAdmin2FANoPhone = &apperrors.AppError{
	Code:    "ADMIN_2FA_NO_PHONE",
	Message: "two-factor authentication is enabled but no phone number is set on this admin account",
	Err:     apperrors.ErrForbidden,
}

// completeLogin issues tokens for a password-verified user, first diverting
// admins to an SMS second factor when admin 2FA is enabled. Only the interactive
// password logins (Login/LoginByPhone) route through here; Register, VerifyOTP,
// and Refresh issue tokens directly.
func (s *UserService) completeLogin(ctx context.Context, user *User) (*AuthResult, error) {
	if user.Role == RoleAdmin && s.admin2FAEnabled(ctx) {
		return s.begin2FAChallenge(ctx, user)
	}
	return s.issueTokens(ctx, user)
}

// admin2FAEnabled reports whether admin logins currently require an SMS second
// factor. It fails safe (disabled) when no settings source is wired or the read
// errors, so a settings outage can neither lock admins out nor silently break
// login — the same best-effort posture as loyalty's settings read.
func (s *UserService) admin2FAEnabled(ctx context.Context) bool {
	if s.settings == nil {
		return false
	}
	cfg, err := s.settings.Get(ctx)
	if err != nil {
		slog.Warn("user: could not read settings for admin 2FA; treating as disabled", "err", err)
		return false
	}
	return cfg.AdminSmsTwoFactorEnabled
}

// begin2FAChallenge sends an SMS one-time code to the admin's phone and returns a
// pending challenge (no tokens). The real session is issued later by
// VerifyAdmin2FA once the code is confirmed. Fails closed if the admin has no phone.
func (s *UserService) begin2FAChallenge(ctx context.Context, user *User) (*AuthResult, error) {
	if user.Phone == nil || *user.Phone == "" {
		return nil, ErrAdmin2FANoPhone
	}
	phone := *user.Phone

	code, err := randomNumericCode(s.otpCfg.Length)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.otp.Upsert(ctx, &OtpCode{
		Phone:     phone,
		CodeHash:  hashToken(code),
		ExpiresAt: now.Add(s.otpCfg.TTL),
		CreatedAt: now,
	}); err != nil {
		return nil, err
	}
	if err := s.sender.Send(ctx, phone, fmt.Sprintf("Your SalehCard admin verification code is %s", code)); err != nil {
		return nil, err
	}

	pending, err := auth.Issue2FAToken(s.secret, user.ID.Hex(), s.otpCfg.TTL)
	if err != nil {
		return nil, err
	}
	return &AuthResult{TwoFactorRequired: true, PendingToken: pending, PhoneHint: maskPhone(phone)}, nil
}

// VerifyAdmin2FA completes an admin login: it validates the pending challenge
// token and the SMS code, then issues the real token pair.
func (s *UserService) VerifyAdmin2FA(ctx context.Context, input VerifyTwoFactorInput) (*AuthResult, error) {
	user, err := s.userFromPendingToken(ctx, input.PendingToken)
	if err != nil {
		return nil, err
	}
	if user.Phone == nil {
		return nil, ErrAdmin2FANoPhone
	}
	if err := s.verifyOTPCode(ctx, *user.Phone, input.Code); err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, user)
}

// ResendAdmin2FA re-sends the SMS code for an in-progress admin 2FA challenge,
// honoring the per-number resend interval. It returns a fresh challenge (new
// pending token + masked phone).
func (s *UserService) ResendAdmin2FA(ctx context.Context, input ResendTwoFactorInput) (*AuthResult, error) {
	user, err := s.userFromPendingToken(ctx, input.PendingToken)
	if err != nil {
		return nil, err
	}
	if user.Phone == nil {
		return nil, ErrAdmin2FANoPhone
	}
	if existing, err := s.otp.FindByPhone(ctx, *user.Phone); err == nil {
		if time.Since(existing.CreatedAt) < s.otpCfg.ResendInterval {
			return nil, &apperrors.AppError{Code: "OTP_THROTTLED", Message: "please wait before requesting another code", Err: apperrors.ErrBadRequest}
		}
	}
	return s.begin2FAChallenge(ctx, user)
}

// userFromPendingToken validates a pending-2FA challenge token and loads the bound
// admin account, re-checking role and suspension. An invalid/expired token maps to
// OTP_EXPIRED so the client restarts the login.
func (s *UserService) userFromPendingToken(ctx context.Context, pendingToken string) (*User, error) {
	userID, err := auth.Verify2FAToken(s.secret, pendingToken)
	if err != nil {
		return nil, &apperrors.AppError{Code: "OTP_EXPIRED", Message: "this login attempt has expired, sign in again", Err: apperrors.ErrUnauthorized}
	}
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return nil, apperrors.ErrUnauthorized
		}
		return nil, err
	}
	if user.Role != RoleAdmin {
		return nil, apperrors.ErrUnauthorized
	}
	if isSuspended(user) {
		return nil, ErrAccountSuspended
	}
	if isDeleted(user) {
		return nil, apperrors.ErrUnauthorized
	}
	return user, nil
}

// verifyOTPCode runs the shared one-time-code validation ladder for a phone:
// existence, expiry/consumed, attempt cap, and hash match. On success it consumes
// the code (delete) so it can't be replayed; on a wrong guess it increments the
// attempt counter. It neither creates users nor issues tokens.
func (s *UserService) verifyOTPCode(ctx context.Context, phone, code string) error {
	rec, err := s.otp.FindByPhone(ctx, phone)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return &apperrors.AppError{Code: "OTP_INVALID", Message: "invalid or expired code", Err: apperrors.ErrUnauthorized}
		}
		return err
	}
	if rec.ConsumedAt != nil || time.Now().After(rec.ExpiresAt) {
		return &apperrors.AppError{Code: "OTP_EXPIRED", Message: "this code has expired, request a new one", Err: apperrors.ErrUnauthorized}
	}
	if rec.Attempts >= s.otpCfg.MaxAttempts {
		return &apperrors.AppError{Code: "OTP_LOCKED", Message: "too many attempts, request a new code", Err: apperrors.ErrUnauthorized}
	}
	if hashToken(strings.TrimSpace(code)) != rec.CodeHash {
		_ = s.otp.IncrementAttempts(ctx, phone)
		return &apperrors.AppError{Code: "OTP_INVALID", Message: "invalid or expired code", Err: apperrors.ErrUnauthorized}
	}
	// Code is good — consume it so it can't be replayed.
	_ = s.otp.DeleteByPhone(ctx, phone)
	return nil
}

// maskPhone hides all but the last 3 digits of a phone for display in the 2FA
// challenge, e.g. "+96178991778" -> "•••778".
func maskPhone(phone string) string {
	if len(phone) <= 3 {
		return "•••"
	}
	return "•••" + phone[len(phone)-3:]
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
