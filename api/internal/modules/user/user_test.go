package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/platform/sms"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// --- in-memory fakes -------------------------------------------------------

type fakeUserRepo struct {
	byID    map[bson.ObjectID]*User
	byEmail map[string]*User
	byPhone map[string]*User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byID: map[bson.ObjectID]*User{}, byEmail: map[string]*User{}, byPhone: map[string]*User{}}
}

func (f *fakeUserRepo) FindByID(_ context.Context, id bson.ObjectID) (*User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeUserRepo) FindByIDs(_ context.Context, ids []bson.ObjectID) ([]*User, error) {
	out := []*User{}
	for _, id := range ids {
		if u, ok := f.byID[id]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}

func (f *fakeUserRepo) FindByEmail(_ context.Context, email string) (*User, error) {
	if u, ok := f.byEmail[normalizeEmail(email)]; ok {
		return u, nil
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeUserRepo) FindByPhone(_ context.Context, phone string) (*User, error) {
	if u, ok := f.byPhone[phone]; ok {
		return u, nil
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeUserRepo) Create(_ context.Context, u *User) error {
	u.Email = normalizeEmail(u.Email)
	if u.Email != "" {
		if _, exists := f.byEmail[u.Email]; exists {
			return apperrors.ErrConflict
		}
	}
	if u.Phone != nil {
		if _, exists := f.byPhone[*u.Phone]; exists {
			return apperrors.ErrConflict
		}
	}
	if u.ID.IsZero() {
		u.ID = bson.NewObjectID()
	}
	f.byID[u.ID] = u
	if u.Email != "" {
		f.byEmail[u.Email] = u
	}
	if u.Phone != nil {
		f.byPhone[*u.Phone] = u
	}
	return nil
}

func (f *fakeUserRepo) Update(_ context.Context, u *User) error {
	if _, ok := f.byID[u.ID]; !ok {
		return apperrors.ErrNotFound
	}
	f.byID[u.ID] = u
	return nil
}

func (f *fakeUserRepo) SetPassword(_ context.Context, id bson.ObjectID, hash string) error {
	u, ok := f.byID[id]
	if !ok {
		return apperrors.ErrNotFound
	}
	u.PasswordHash = &hash
	return nil
}

func (f *fakeUserRepo) Delete(_ context.Context, id bson.ObjectID) error {
	delete(f.byID, id)
	return nil
}

func (f *fakeUserRepo) TouchLastSeen(_ context.Context, id bson.ObjectID, t time.Time) error {
	if u, ok := f.byID[id]; ok {
		u.LastSeen = t
	}
	return nil
}

func (f *fakeUserRepo) CountActiveSince(_ context.Context, since time.Time) (int64, error) {
	var n int64
	for _, u := range f.byID {
		if !u.LastSeen.Before(since) && !u.LastSeen.IsZero() {
			n++
		}
	}
	return n, nil
}

func (f *fakeUserRepo) CountActiveSuperAdmins(_ context.Context) (int64, error) {
	var n int64
	for _, u := range f.byID {
		if u.Role == RoleAdmin && u.AdminRoleID == nil &&
			u.Status != StatusSuspended && u.Status != StatusDeleted {
			n++
		}
	}
	return n, nil
}

func (f *fakeUserRepo) ListAll(_ context.Context, _ UserFilter, _ pagination.Params) ([]*User, int64, error) {
	out := make([]*User, 0, len(f.byID))
	for _, u := range f.byID {
		out = append(out, u)
	}
	return out, int64(len(out)), nil
}

func (f *fakeUserRepo) UpdateRole(_ context.Context, id bson.ObjectID, role Role, adminRoleID *bson.ObjectID) (*User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	u.Role = role
	if role == RoleAdmin {
		u.AdminRoleID = adminRoleID
	} else {
		u.AdminRoleID = nil
	}
	return u, nil
}

func (f *fakeUserRepo) UpdateStatus(_ context.Context, id bson.ObjectID, status Status) (*User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	u.Status = status
	return u, nil
}

func (f *fakeUserRepo) UpdateResellerTier(_ context.Context, id bson.ObjectID, tier string) (*User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	u.ResellerTier = tier
	return u, nil
}

func (f *fakeUserRepo) BulkUpdateStatus(_ context.Context, ids []bson.ObjectID, status Status) (int64, error) {
	var n int64
	for _, id := range ids {
		if u, ok := f.byID[id]; ok {
			u.Status = status
			n++
		}
	}
	return n, nil
}

func (f *fakeUserRepo) SoftDelete(_ context.Context, id bson.ObjectID) (*User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	u.Status = StatusDeleted
	now := time.Now().UTC()
	u.DeletedAt = &now
	u.Email = ""
	u.Phone = nil
	return u, nil
}

func (f *fakeUserRepo) SoftDeleteSelf(_ context.Context, id bson.ObjectID) (*User, error) {
	u, ok := f.byID[id]
	if !ok || u.Status == StatusDeleted || u.Status == StatusSuspended || u.WalletBalance > 0 {
		return nil, apperrors.ErrNotFound // mirror the Mongo filter miss
	}
	u.Status = StatusDeleted
	now := time.Now().UTC()
	u.DeletedAt = &now
	u.Email = ""
	u.Phone = nil
	u.PasswordHash = nil
	u.Name = ""
	u.SavedPlayerIDs = nil
	return u, nil
}

// fakeOTPRepo is a single-record-per-phone in-memory OTPRepository.
type fakeOTPRepo struct {
	byPhone map[string]*OtpCode
}

func newFakeOTPRepo() *fakeOTPRepo { return &fakeOTPRepo{byPhone: map[string]*OtpCode{}} }

func (f *fakeOTPRepo) Upsert(_ context.Context, c *OtpCode) error {
	if c.ID.IsZero() {
		c.ID = bson.NewObjectID()
	}
	f.byPhone[c.Phone] = c
	return nil
}

func (f *fakeOTPRepo) FindByPhone(_ context.Context, phone string) (*OtpCode, error) {
	if c, ok := f.byPhone[phone]; ok {
		return c, nil
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeOTPRepo) IncrementAttempts(_ context.Context, phone string) error {
	if c, ok := f.byPhone[phone]; ok {
		c.Attempts++
	}
	return nil
}

func (f *fakeOTPRepo) DeleteByPhone(_ context.Context, phone string) error {
	delete(f.byPhone, phone)
	return nil
}

// captureSender records the last message sent, for asserting the OTP code.
type captureSender struct{ lastMessage string }

func (c *captureSender) Send(_ context.Context, _, message string) error {
	c.lastMessage = message
	return nil
}

func testOTPConfig() OTPConfig {
	return OTPConfig{CountryCode: "+961", Length: 6, TTL: 5 * time.Minute, ResendInterval: time.Minute, MaxAttempts: 5}
}

type fakeRefreshRepo struct {
	byHash map[string]*RefreshToken
}

func newFakeRefreshRepo() *fakeRefreshRepo {
	return &fakeRefreshRepo{byHash: map[string]*RefreshToken{}}
}

func (f *fakeRefreshRepo) Create(_ context.Context, t *RefreshToken) error {
	if t.ID.IsZero() {
		t.ID = bson.NewObjectID()
	}
	f.byHash[t.TokenHash] = t
	return nil
}

func (f *fakeRefreshRepo) FindByHash(_ context.Context, hash string) (*RefreshToken, error) {
	if t, ok := f.byHash[hash]; ok {
		return t, nil
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeRefreshRepo) Revoke(_ context.Context, id bson.ObjectID) error {
	now := time.Now().UTC()
	for _, t := range f.byHash {
		if t.ID == id {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (f *fakeRefreshRepo) RevokeAllForUser(_ context.Context, userID bson.ObjectID) error {
	now := time.Now().UTC()
	for _, t := range f.byHash {
		if t.UserID == userID {
			t.RevokedAt = &now
		}
	}
	return nil
}

func newTestService() (*UserService, *fakeUserRepo, *fakeRefreshRepo) {
	repo := newFakeUserRepo()
	refresh := newFakeRefreshRepo()
	svc := NewUserService(repo, refresh, newFakeOTPRepo(), sms.LogSender{}, testOTPConfig(), "test-secret", 15*time.Minute, 24*time.Hour)
	return svc, repo, refresh
}

// --- tests -----------------------------------------------------------------

func TestRegister_HashesAndIssuesTokens(t *testing.T) {
	svc, repo, _ := newTestService()

	res, err := svc.Register(context.Background(), RegisterInput{Email: "A@B.com", Password: "password123"})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)
	assert.Equal(t, RoleCustomer, res.User.Role)
	assert.Equal(t, "a@b.com", res.User.Email) // normalized

	stored := repo.byEmail["a@b.com"]
	require.NotNil(t, stored.PasswordHash)
	assert.NotEqual(t, "password123", *stored.PasswordHash) // hashed, not plaintext
}

func TestRegister_RejectsShortPassword(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "short"})
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrBadRequest)
}

func TestRegister_RejectsDuplicate(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "password123"})
	require.NoError(t, err)
	_, err = svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "password123"})
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrConflict)
}

func TestLogin_Succeeds(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "password123"})
	require.NoError(t, err)

	res, err := svc.Login(context.Background(), LoginInput{Email: "a@b.com", Password: "password123"})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
}

func TestLogin_RejectsBadPassword(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "password123"})
	require.NoError(t, err)

	_, err = svc.Login(context.Background(), LoginInput{Email: "a@b.com", Password: "wrongpass"})
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestLogin_UnknownEmailIsUnauthorized(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Login(context.Background(), LoginInput{Email: "nobody@b.com", Password: "password123"})
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestLogin_SuspendedIsForbidden(t *testing.T) {
	svc, repo, _ := newTestService()
	_, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "password123"})
	require.NoError(t, err)

	// Suspend the account, then a correct-credential login must be rejected as
	// forbidden (not unauthorized) — the credentials are valid, the account is not.
	repo.byEmail["a@b.com"].Status = StatusSuspended

	_, err = svc.Login(context.Background(), LoginInput{Email: "a@b.com", Password: "password123"})
	assert.ErrorIs(t, err, apperrors.ErrForbidden)
	assert.ErrorIs(t, err, ErrAccountSuspended)
}

func TestRefresh_SuspendedIsForbidden(t *testing.T) {
	svc, repo, _ := newTestService()
	reg, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "password123"})
	require.NoError(t, err)

	repo.byEmail["a@b.com"].Status = StatusSuspended

	_, err = svc.Refresh(context.Background(), reg.RefreshToken)
	assert.ErrorIs(t, err, apperrors.ErrForbidden)
}

func TestRefresh_RotatesAndRevokesOld(t *testing.T) {
	svc, _, _ := newTestService()
	reg, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "password123"})
	require.NoError(t, err)

	rotated, err := svc.Refresh(context.Background(), reg.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, rotated.AccessToken)
	assert.NotEqual(t, reg.RefreshToken, rotated.RefreshToken)

	// The old token must now be rejected (rotation revoked it).
	_, err = svc.Refresh(context.Background(), reg.RefreshToken)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestRefresh_RejectsExpired(t *testing.T) {
	repo := newFakeUserRepo()
	refresh := newFakeRefreshRepo()
	svc := NewUserService(repo, refresh, newFakeOTPRepo(), sms.LogSender{}, testOTPConfig(), "test-secret", 15*time.Minute, -time.Hour) // already-expired refresh TTL

	reg, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "password123"})
	require.NoError(t, err)
	_, err = svc.Refresh(context.Background(), reg.RefreshToken)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestLogout_RevokesToken(t *testing.T) {
	svc, _, _ := newTestService()
	reg, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "password123"})
	require.NoError(t, err)

	require.NoError(t, svc.Logout(context.Background(), reg.RefreshToken))
	_, err = svc.Refresh(context.Background(), reg.RefreshToken)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestUpdateProfile(t *testing.T) {
	svc, _, _ := newTestService()
	reg, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "password123"})
	require.NoError(t, err)

	tr := "tr"
	updated, err := svc.UpdateProfile(context.Background(), reg.User.ID, UpdateProfileInput{
		Locale:         &tr,
		SavedPlayerIDs: []SavedPlayerID{{Label: " PUBG main ", Value: " player-1 "}},
	})
	require.NoError(t, err)
	assert.Equal(t, "tr", updated.Locale)
	// Label and value are trimmed on save.
	assert.Equal(t, []SavedPlayerID{{Label: "PUBG main", Value: "player-1"}}, updated.SavedPlayerIDs)

	// A saved player ID missing a label (or value) is rejected.
	_, err = svc.UpdateProfile(context.Background(), reg.User.ID, UpdateProfileInput{
		SavedPlayerIDs: []SavedPlayerID{{Label: "", Value: "player-2"}},
	})
	assert.ErrorIs(t, err, apperrors.ErrBadRequest)
}

// TestSavedPlayerIDLegacyDecode verifies the tolerant BSON decoder: documents
// written before labels existed stored the list as bare strings, and must still
// decode (mirrored into both Label and Value) alongside the new document shape.
func TestSavedPlayerIDLegacyDecode(t *testing.T) {
	// A mixed array: one legacy string entry, one new {label, value} document.
	doc := bson.D{{Key: "savedPlayerIds", Value: bson.A{
		"legacy-123",
		bson.D{{Key: "label", Value: "PUBG main"}, {Key: "value", Value: "999"}},
	}}}
	raw, err := bson.Marshal(doc)
	require.NoError(t, err)

	var out struct {
		SavedPlayerIDs []SavedPlayerID `bson:"savedPlayerIds"`
	}
	require.NoError(t, bson.Unmarshal(raw, &out))
	assert.Equal(t, []SavedPlayerID{
		{Label: "legacy-123", Value: "legacy-123"},
		{Label: "PUBG main", Value: "999"},
	}, out.SavedPlayerIDs)
}

// otpServiceWith builds a service with explicit OTP fakes for OTP-flow tests.
func otpServiceWith() (*UserService, *fakeUserRepo, *fakeOTPRepo, *captureSender) {
	repo := newFakeUserRepo()
	otp := newFakeOTPRepo()
	sender := &captureSender{}
	svc := NewUserService(repo, newFakeRefreshRepo(), otp, sender, testOTPConfig(), "test-secret", 15*time.Minute, 24*time.Hour)
	return svc, repo, otp, sender
}

func TestRequestOTP_NormalizesPhoneAndSendsCode(t *testing.T) {
	svc, _, otp, sender := otpServiceWith()

	// Local Lebanese number with a leading 0 → +961 E.164.
	require.NoError(t, svc.RequestOTP(context.Background(), RequestOTPInput{Phone: "070 123 456"}))
	rec, err := otp.FindByPhone(context.Background(), "+96170123456")
	require.NoError(t, err)
	assert.NotEmpty(t, rec.CodeHash)
	assert.Contains(t, sender.lastMessage, "SalehCard")
}

func TestRequestOTP_RejectsInvalidPhone(t *testing.T) {
	svc, _, _, _ := otpServiceWith()
	err := svc.RequestOTP(context.Background(), RequestOTPInput{Phone: "123"})
	assert.ErrorIs(t, err, apperrors.ErrBadRequest)
}

func TestRequestOTP_ThrottlesRepeat(t *testing.T) {
	svc, _, _, _ := otpServiceWith()
	require.NoError(t, svc.RequestOTP(context.Background(), RequestOTPInput{Phone: "+96170123456"}))
	err := svc.RequestOTP(context.Background(), RequestOTPInput{Phone: "+96170123456"})
	assert.ErrorIs(t, err, apperrors.ErrBadRequest)
}

func TestVerifyOTP_CreatesAccountAndIssuesTokens(t *testing.T) {
	svc, repo, otp, _ := otpServiceWith()
	require.NoError(t, svc.RequestOTP(context.Background(), RequestOTPInput{Phone: "+96170123456"}))

	// Pull the raw code by replacing the stored record with a known one.
	known := "654321"
	rec, _ := otp.FindByPhone(context.Background(), "+96170123456")
	rec.CodeHash = hashToken(known)

	res, err := svc.VerifyOTP(context.Background(), VerifyOTPInput{Phone: "+96170123456", Code: known})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
	require.NotNil(t, res.User.Phone)
	assert.Equal(t, "+96170123456", *res.User.Phone)

	stored, err := repo.FindByPhone(context.Background(), "+96170123456")
	require.NoError(t, err)
	assert.Equal(t, RoleCustomer, stored.Role)

	// Code is consumed → second verify fails.
	_, err = svc.VerifyOTP(context.Background(), VerifyOTPInput{Phone: "+96170123456", Code: known})
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestVerifyOTP_WrongCodeIsUnauthorized(t *testing.T) {
	svc, _, _, _ := otpServiceWith()
	require.NoError(t, svc.RequestOTP(context.Background(), RequestOTPInput{Phone: "+96170123456"}))
	_, err := svc.VerifyOTP(context.Background(), VerifyOTPInput{Phone: "+96170123456", Code: "000000"})
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestVerifyOTP_SetsPasswordThenPhoneLoginWorks(t *testing.T) {
	svc, _, otp, _ := otpServiceWith()
	require.NoError(t, svc.RequestOTP(context.Background(), RequestOTPInput{Phone: "+96170123456"}))
	known := "112233"
	rec, _ := otp.FindByPhone(context.Background(), "+96170123456")
	rec.CodeHash = hashToken(known)

	_, err := svc.VerifyOTP(context.Background(), VerifyOTPInput{Phone: "+96170123456", Code: known, Password: "password123"})
	require.NoError(t, err)

	res, err := svc.LoginByPhone(context.Background(), PhoneLoginInput{Phone: "+96170123456", Password: "password123"})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)

	_, err = svc.LoginByPhone(context.Background(), PhoneLoginInput{Phone: "+96170123456", Password: "wrong"})
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestHandler_GetProfile_Unauthenticated(t *testing.T) {
	svc, _, _ := newTestService()
	h := NewHandler(svc, false, 15*time.Minute)

	// No claims in context → 401.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()
	h.GetProfile(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
