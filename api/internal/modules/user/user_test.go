package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// --- in-memory fakes -------------------------------------------------------

type fakeUserRepo struct {
	byID    map[bson.ObjectID]*User
	byEmail map[string]*User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byID: map[bson.ObjectID]*User{}, byEmail: map[string]*User{}}
}

func (f *fakeUserRepo) FindByID(_ context.Context, id bson.ObjectID) (*User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeUserRepo) FindByEmail(_ context.Context, email string) (*User, error) {
	if u, ok := f.byEmail[normalizeEmail(email)]; ok {
		return u, nil
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeUserRepo) Create(_ context.Context, u *User) error {
	u.Email = normalizeEmail(u.Email)
	if _, exists := f.byEmail[u.Email]; exists {
		return apperrors.ErrConflict
	}
	if u.ID.IsZero() {
		u.ID = bson.NewObjectID()
	}
	f.byID[u.ID] = u
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepo) Update(_ context.Context, u *User) error {
	if _, ok := f.byID[u.ID]; !ok {
		return apperrors.ErrNotFound
	}
	f.byID[u.ID] = u
	return nil
}

func (f *fakeUserRepo) Delete(_ context.Context, id bson.ObjectID) error {
	delete(f.byID, id)
	return nil
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
	svc := NewUserService(repo, refresh, "test-secret", 15*time.Minute, 24*time.Hour)
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
	svc := NewUserService(repo, refresh, "test-secret", 15*time.Minute, -time.Hour) // already-expired refresh TTL

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
		SavedPlayerIDs: []string{"player-1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "tr", updated.Locale)
	assert.Equal(t, []string{"player-1"}, updated.SavedPlayerIDs)
}

func TestHandler_GetProfile_Unauthenticated(t *testing.T) {
	svc, _, _ := newTestService()
	h := NewHandler(svc, false)

	// No claims in context → 401.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()
	h.GetProfile(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
