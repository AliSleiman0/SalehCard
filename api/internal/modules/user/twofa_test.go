package user

import (
	"context"
	"testing"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// fakeSettings is a static settingsSource for the admin-2FA gate.
type fakeSettings struct{ enabled bool }

func (f fakeSettings) Get(_ context.Context) (*settings.Settings, error) {
	return &settings.Settings{AdminSmsTwoFactorEnabled: f.enabled}, nil
}

// twoFAService builds a service whose admin-2FA gate is on/off per `enabled`.
func twoFAService(enabled bool) (*UserService, *fakeUserRepo, *fakeOTPRepo, *captureSender) {
	repo := newFakeUserRepo()
	otp := newFakeOTPRepo()
	sender := &captureSender{}
	svc := NewUserService(repo, newFakeRefreshRepo(), otp, sender, testOTPConfig(), "test-secret", 15*time.Minute, 24*time.Hour,
		WithSettings(fakeSettings{enabled: enabled}))
	return svc, repo, otp, sender
}

// seedAdmin inserts an admin account (password "password123") with the given
// phone (pass "" for a phoneless admin) into the fake repo.
func seedAdmin(t *testing.T, repo *fakeUserRepo, email, phone string) *User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)
	h := string(hash)
	u := &User{Email: email, PasswordHash: &h, Role: RoleAdmin, Status: StatusActive}
	if phone != "" {
		p := phone
		u.Phone = &p
	}
	require.NoError(t, repo.Create(context.Background(), u))
	return u
}

// setKnownCode overwrites the stored OTP hash so a test knows the raw code.
func setKnownCode(t *testing.T, otp *fakeOTPRepo, phone, code string) {
	t.Helper()
	rec, err := otp.FindByPhone(context.Background(), phone)
	require.NoError(t, err)
	rec.CodeHash = hashToken(code)
}

func TestLogin_AdminWith2FA_ReturnsChallengeNotTokens(t *testing.T) {
	svc, repo, otp, sender := twoFAService(true)
	seedAdmin(t, repo, "admin@x.com", "+96170123456")

	res, err := svc.Login(context.Background(), LoginInput{Email: "admin@x.com", Password: "password123"})
	require.NoError(t, err)

	assert.True(t, res.TwoFactorRequired)
	assert.Empty(t, res.AccessToken)
	assert.Empty(t, res.RefreshToken)
	assert.NotEmpty(t, res.PendingToken)
	assert.Equal(t, "•••456", res.PhoneHint)

	// A code was stored for the admin's phone and an SMS was sent.
	_, err = otp.FindByPhone(context.Background(), "+96170123456")
	require.NoError(t, err)
	assert.Contains(t, sender.lastMessage, "admin verification code")
}

func TestLogin_AdminWith2FA_NoPhone_FailsClosed(t *testing.T) {
	svc, repo, _, sender := twoFAService(true)
	seedAdmin(t, repo, "admin@x.com", "") // no phone

	_, err := svc.Login(context.Background(), LoginInput{Email: "admin@x.com", Password: "password123"})
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrForbidden)
	assert.ErrorIs(t, err, ErrAdmin2FANoPhone)
	assert.Empty(t, sender.lastMessage) // nothing sent
}

func TestLogin_CustomerWith2FAOn_IsNotChallenged(t *testing.T) {
	svc, _, _, _ := twoFAService(true)
	_, err := svc.Register(context.Background(), RegisterInput{Email: "c@x.com", Password: "password123"})
	require.NoError(t, err)

	res, err := svc.Login(context.Background(), LoginInput{Email: "c@x.com", Password: "password123"})
	require.NoError(t, err)
	assert.False(t, res.TwoFactorRequired)
	assert.NotEmpty(t, res.AccessToken) // customers log in directly
}

func TestLogin_Admin2FADisabled_IssuesTokensDirectly(t *testing.T) {
	svc, repo, _, _ := twoFAService(false)
	seedAdmin(t, repo, "admin@x.com", "+96170123456")

	res, err := svc.Login(context.Background(), LoginInput{Email: "admin@x.com", Password: "password123"})
	require.NoError(t, err)
	assert.False(t, res.TwoFactorRequired)
	assert.NotEmpty(t, res.AccessToken)
}

func TestLogin_AdminNoSettingsPort_IssuesTokensDirectly(t *testing.T) {
	// A service built without WithSettings treats admin 2FA as off.
	svc, repo, _ := newTestService()
	seedAdmin(t, repo, "admin@x.com", "+96170123456")

	res, err := svc.Login(context.Background(), LoginInput{Email: "admin@x.com", Password: "password123"})
	require.NoError(t, err)
	assert.False(t, res.TwoFactorRequired)
	assert.NotEmpty(t, res.AccessToken)
}

func TestVerifyAdmin2FA_HappyPathIssuesTokens(t *testing.T) {
	svc, repo, otp, _ := twoFAService(true)
	seedAdmin(t, repo, "admin@x.com", "+96170123456")

	ch, err := svc.Login(context.Background(), LoginInput{Email: "admin@x.com", Password: "password123"})
	require.NoError(t, err)
	require.True(t, ch.TwoFactorRequired)

	setKnownCode(t, otp, "+96170123456", "654321")

	res, err := svc.VerifyAdmin2FA(context.Background(), VerifyTwoFactorInput{PendingToken: ch.PendingToken, Code: "654321"})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)
	assert.Equal(t, RoleAdmin, res.User.Role)

	// Code is consumed → a replay of the same challenge fails.
	_, err = svc.VerifyAdmin2FA(context.Background(), VerifyTwoFactorInput{PendingToken: ch.PendingToken, Code: "654321"})
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestVerifyAdmin2FA_WrongCodeIsUnauthorizedAndCounts(t *testing.T) {
	svc, repo, otp, _ := twoFAService(true)
	seedAdmin(t, repo, "admin@x.com", "+96170123456")

	ch, err := svc.Login(context.Background(), LoginInput{Email: "admin@x.com", Password: "password123"})
	require.NoError(t, err)
	setKnownCode(t, otp, "+96170123456", "654321")

	_, err = svc.VerifyAdmin2FA(context.Background(), VerifyTwoFactorInput{PendingToken: ch.PendingToken, Code: "000000"})
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)

	rec, err := otp.FindByPhone(context.Background(), "+96170123456")
	require.NoError(t, err)
	assert.Equal(t, 1, rec.Attempts) // wrong guess bumped the counter
}

func TestVerifyAdmin2FA_ExpiredCode(t *testing.T) {
	svc, repo, otp, _ := twoFAService(true)
	seedAdmin(t, repo, "admin@x.com", "+96170123456")

	ch, err := svc.Login(context.Background(), LoginInput{Email: "admin@x.com", Password: "password123"})
	require.NoError(t, err)

	// Force the stored code to be expired.
	rec, err := otp.FindByPhone(context.Background(), "+96170123456")
	require.NoError(t, err)
	rec.ExpiresAt = time.Now().Add(-time.Minute)
	rec.CodeHash = hashToken("654321")

	_, err = svc.VerifyAdmin2FA(context.Background(), VerifyTwoFactorInput{PendingToken: ch.PendingToken, Code: "654321"})
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestVerifyAdmin2FA_InvalidPendingTokenIsRejected(t *testing.T) {
	svc, repo, _, _ := twoFAService(true)
	seedAdmin(t, repo, "admin@x.com", "+96170123456")

	_, err := svc.VerifyAdmin2FA(context.Background(), VerifyTwoFactorInput{PendingToken: "garbage.token.value", Code: "654321"})
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestVerifyAdmin2FA_PendingTokenForNonAdminRejected(t *testing.T) {
	// A pending token minted for a non-admin id must not complete an admin session.
	svc, _, _, _ := twoFAService(true)
	// Seed a customer, mint a pending token bound to them — a non-admin must not
	// be able to complete an admin 2FA challenge even with a valid code.
	reg, err := svc.Register(context.Background(), RegisterInput{Email: "c@x.com", Password: "password123"})
	require.NoError(t, err)

	pending, err := auth.Issue2FAToken(svc.secret, reg.User.ID.Hex(), time.Minute)
	require.NoError(t, err)
	_, err = svc.VerifyAdmin2FA(context.Background(), VerifyTwoFactorInput{PendingToken: pending, Code: "654321"})
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestResendAdmin2FA_ThrottledImmediately(t *testing.T) {
	svc, repo, _, _ := twoFAService(true)
	seedAdmin(t, repo, "admin@x.com", "+96170123456")

	ch, err := svc.Login(context.Background(), LoginInput{Email: "admin@x.com", Password: "password123"})
	require.NoError(t, err)

	// A resend right away is inside the resend interval → throttled.
	_, err = svc.ResendAdmin2FA(context.Background(), ResendTwoFactorInput{PendingToken: ch.PendingToken})
	assert.ErrorIs(t, err, apperrors.ErrBadRequest)
}
