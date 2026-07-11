package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/internal/platform/sms"
)

// --- deletion-port fakes ----------------------------------------------------

type fakeOrderGuard struct{ inFlight int64 }

func (f *fakeOrderGuard) CountInFlightByUser(context.Context, bson.ObjectID) (int64, error) {
	return f.inFlight, nil
}

type fakePaymentGuard struct{ open int64 }

func (f *fakePaymentGuard) CountOpenByUser(context.Context, bson.ObjectID) (int64, error) {
	return f.open, nil
}

type fakeTopUpGuard struct{ pending int64 }

func (f *fakeTopUpGuard) CountPendingForUser(context.Context, bson.ObjectID) (int64, error) {
	return f.pending, nil
}

type fakeKYCPurger struct {
	called bool
	hadKyc bool
	err    error
}

func (f *fakeKYCPurger) PurgeByUser(context.Context, bson.ObjectID) (bool, error) {
	f.called = true
	return f.hadKyc, f.err
}

type fakeTokenPurger struct{ called bool }

func (f *fakeTokenPurger) DeleteAllForUser(context.Context, bson.ObjectID) (int64, error) {
	f.called = true
	return 1, nil
}

// deletionFixture wires a service whose deletion ports are all fakes with
// permissive defaults; tests tighten individual fakes per scenario.
type deletionFixture struct {
	svc     *UserService
	repo    *fakeUserRepo
	refresh *fakeRefreshRepo
	orders  *fakeOrderGuard
	pays    *fakePaymentGuard
	topups  *fakeTopUpGuard
	kyc     *fakeKYCPurger
	tokens  *fakeTokenPurger
}

func newDeletionFixture() *deletionFixture {
	f := &deletionFixture{
		repo:    newFakeUserRepo(),
		refresh: newFakeRefreshRepo(),
		orders:  &fakeOrderGuard{},
		pays:    &fakePaymentGuard{},
		topups:  &fakeTopUpGuard{},
		kyc:     &fakeKYCPurger{hadKyc: true},
		tokens:  &fakeTokenPurger{},
	}
	f.svc = NewUserService(f.repo, f.refresh, newFakeOTPRepo(), sms.LogSender{}, testOTPConfig(),
		"test-secret", 15*time.Minute, 24*time.Hour,
		WithDeletionPorts(DeletionPorts{Orders: f.orders, Payments: f.pays, TopUps: f.topups, KYC: f.kyc, Tokens: f.tokens}))
	return f
}

// seedCustomer registers a customer and returns it (with a live refresh token).
func (f *deletionFixture) seedCustomer(t *testing.T) (*User, *AuthResult) {
	t.Helper()
	res, err := f.svc.Register(context.Background(), RegisterInput{Email: "c@d.com", Password: "password123", Name: "Cee Dee"})
	require.NoError(t, err)
	u := f.repo.byEmail["c@d.com"]
	require.NotNil(t, u)
	return u, res
}

// --- service tests ----------------------------------------------------------

func TestDeleteAccount_HappyPath(t *testing.T) {
	f := newDeletionFixture()
	u, reg := f.seedCustomer(t)
	u.SavedPlayerIDs = []SavedPlayerID{{Label: "PUBG", Value: "12345"}}

	hadKyc, err := f.svc.DeleteAccount(context.Background(), u.ID)
	require.NoError(t, err)
	assert.True(t, hadKyc)

	assert.Equal(t, StatusDeleted, u.Status)
	assert.NotNil(t, u.DeletedAt)
	assert.Empty(t, u.Email)
	assert.Nil(t, u.Phone)
	assert.Nil(t, u.PasswordHash)
	assert.Empty(t, u.Name)
	assert.Empty(t, u.SavedPlayerIDs)

	assert.True(t, f.kyc.called, "KYC purger must run")
	assert.True(t, f.tokens.called, "device tokens must be purged")

	// Every session is dead: the pre-deletion refresh token no longer rotates.
	_, err = f.svc.Refresh(context.Background(), reg.RefreshToken)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestDeleteAccount_WalletNotEmpty(t *testing.T) {
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	u.WalletBalance = 12.5

	_, err := f.svc.DeleteAccount(context.Background(), u.ID)
	assert.ErrorIs(t, err, apperrors.ErrConflict)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "WALLET_NOT_EMPTY", appErr.Code)
	assert.False(t, f.kyc.called, "KYC must survive a refused deletion")
	assert.NotEqual(t, StatusDeleted, u.Status)
}

func TestDeleteAccount_OrdersInFlight(t *testing.T) {
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	f.orders.inFlight = 1

	_, err := f.svc.DeleteAccount(context.Background(), u.ID)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ORDERS_IN_FLIGHT", appErr.Code)
	assert.False(t, f.kyc.called)
}

func TestDeleteAccount_PaymentsPending(t *testing.T) {
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	f.pays.open = 1

	_, err := f.svc.DeleteAccount(context.Background(), u.ID)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "PAYMENTS_PENDING", appErr.Code)
	assert.False(t, f.kyc.called)
}

func TestDeleteAccount_TopUpsPending(t *testing.T) {
	// A pending top-up is money already paid out-of-band; deleting before the
	// admin decision would silently forfeit it.
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	f.topups.pending = 1

	_, err := f.svc.DeleteAccount(context.Background(), u.ID)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "TOPUPS_PENDING", appErr.Code)
	assert.False(t, f.kyc.called)
}

func TestDeleteAccount_SuspendedRefused(t *testing.T) {
	// Self-deletion frees the unique email/phone indexes; a suspended (banned)
	// account must not use it to shed the suspension and re-register clean.
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	u.Status = StatusSuspended

	_, err := f.svc.DeleteAccount(context.Background(), u.ID)
	assert.ErrorIs(t, err, apperrors.ErrForbidden)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ACCOUNT_SUSPENDED", appErr.Code)
	assert.False(t, f.kyc.called)
	assert.Equal(t, StatusSuspended, u.Status)
}

func TestDeleteAccount_GuardRaceDuringPurgeRejected(t *testing.T) {
	// A top-up filed while the KYC purge's blob I/O runs must still block the
	// flip — the guards re-run between the purge and SoftDeleteSelf.
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	f.svc.deletion.KYC = &purgeRacer{topups: f.topups}

	_, err := f.svc.DeleteAccount(context.Background(), u.ID)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "TOPUPS_PENDING", appErr.Code)
	assert.NotEqual(t, StatusDeleted, u.Status)
}

// purgeRacer files a pending top-up during the purge — after the first guard
// pass — to exercise the post-purge guard re-check.
type purgeRacer struct{ topups *fakeTopUpGuard }

func (r *purgeRacer) PurgeByUser(context.Context, bson.ObjectID) (bool, error) {
	r.topups.pending = 1
	return false, nil
}

func TestDeleteAccount_AdminRefused(t *testing.T) {
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	u.Role = RoleAdmin

	_, err := f.svc.DeleteAccount(context.Background(), u.ID)
	assert.ErrorIs(t, err, apperrors.ErrForbidden)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ADMIN_ACCOUNT", appErr.Code)
	assert.False(t, f.kyc.called)
}

func TestDeleteAccount_AlreadyDeletedIsIdempotent(t *testing.T) {
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	u.Status = StatusDeleted
	f.kyc.hadKyc = false

	hadKyc, err := f.svc.DeleteAccount(context.Background(), u.ID)
	require.NoError(t, err)
	assert.False(t, hadKyc)
	// The double-tap still re-runs the KYC purge: a submission that raced in
	// around the original deletion must not survive the account.
	assert.True(t, f.kyc.called, "double-tap must re-purge KYC")
}

func TestDeleteAccount_KycPurgeFailureAborts(t *testing.T) {
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	f.kyc.err = errors.New("blob storage down")

	_, err := f.svc.DeleteAccount(context.Background(), u.ID)
	require.Error(t, err)
	assert.NotEqual(t, StatusDeleted, u.Status, "account must stay intact for a retry")
	assert.False(t, f.tokens.called, "no cleanup on an aborted deletion")
}

func TestDeleteAccount_BalanceRaceRejected(t *testing.T) {
	// A credit lands between the guard check and the flip: the guarded update
	// misses, the re-fetch sees the balance, and the deletion is refused.
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	f.kyc.hadKyc = false

	// Sneak the credit in "after the guard" by making the purger the racer.
	f.kyc.err = nil
	raceKyc := &racingKYCPurger{user: u}
	f.svc.deletion.KYC = raceKyc

	_, err := f.svc.DeleteAccount(context.Background(), u.ID)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "WALLET_NOT_EMPTY", appErr.Code)
	assert.NotEqual(t, StatusDeleted, u.Status)
}

// racingKYCPurger credits the wallet during the purge — after the service's
// balance guard has already passed — to exercise SoftDeleteSelf's filter.
type racingKYCPurger struct{ user *User }

func (r *racingKYCPurger) PurgeByUser(context.Context, bson.ObjectID) (bool, error) {
	r.user.WalletBalance = 5
	return false, nil
}

func TestDeleteAccount_UnwiredPortsFailClosed(t *testing.T) {
	svc, repo, _ := newTestService() // no WithDeletionPorts
	res, err := svc.Register(context.Background(), RegisterInput{Email: "c@d.com", Password: "password123"})
	require.NoError(t, err)
	_ = res
	u := repo.byEmail["c@d.com"]

	_, err = svc.DeleteAccount(context.Background(), u.ID)
	assert.ErrorIs(t, err, apperrors.ErrInternal)
	assert.NotEqual(t, StatusDeleted, u.Status)
}

func TestRefresh_DeletedIsRejected(t *testing.T) {
	svc, repo, _ := newTestService()
	reg, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "password123"})
	require.NoError(t, err)

	repo.byEmail["a@b.com"].Status = StatusDeleted

	// Deleted reads as a plain 401 — indistinguishable from a bad token, unlike
	// suspended's explicit 403.
	_, err = svc.Refresh(context.Background(), reg.RefreshToken)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
	assert.NotErrorIs(t, err, apperrors.ErrForbidden)
}

func TestGetProfile_DeletedIsRejected(t *testing.T) {
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	require.NoError(t, errFrom(f.svc.DeleteAccount(context.Background(), u.ID)))

	_, err := f.svc.GetProfile(context.Background(), u.ID)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

// errFrom drops a leading value from a (T, error) return for require.NoError.
func errFrom(_ bool, err error) error { return err }

// --- handler tests ----------------------------------------------------------

// newDeleteMeRequest builds an authenticated DELETE /users/me request whose
// context carries claims for id, mirroring what AuthRequired attaches.
func newDeleteMeRequest(t *testing.T, id bson.ObjectID) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
	claims := &auth.Claims{UserID: id.Hex(), Role: string(RoleCustomer)}
	return req.WithContext(auth.ContextWithClaims(req.Context(), claims))
}

func TestDeleteMe_Handler(t *testing.T) {
	f := newDeletionFixture()
	u, _ := f.seedCustomer(t)
	h := NewHandler(f.svc, false, 24*time.Hour)

	t.Run("no claims → 401", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.DeleteMe(rr, httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil))
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("wallet guard → 409 with code", func(t *testing.T) {
		u.WalletBalance = 3
		rr := httptest.NewRecorder()
		h.DeleteMe(rr, newDeleteMeRequest(t, u.ID))
		assert.Equal(t, http.StatusConflict, rr.Code)
		var body struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
		assert.Equal(t, "WALLET_NOT_EMPTY", body.Error.Code)
	})

	t.Run("happy path → 200", func(t *testing.T) {
		u.WalletBalance = 0
		rr := httptest.NewRecorder()
		h.DeleteMe(rr, newDeleteMeRequest(t, u.ID))
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, StatusDeleted, u.Status)
	})
}
