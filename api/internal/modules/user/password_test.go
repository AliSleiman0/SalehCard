package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

func TestChangePassword(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestService()
	res, err := svc.Register(ctx, RegisterInput{Email: "pw@b.com", Password: "password123"})
	require.NoError(t, err)
	id := res.User.ID

	t.Run("wrong current password is a 400, not a 401", func(t *testing.T) {
		err := svc.ChangePassword(ctx, id, ChangePasswordInput{CurrentPassword: "nope-nope", NewPassword: "newpass123"})
		var ae *apperrors.AppError
		require.True(t, errors.As(err, &ae))
		assert.Equal(t, "WRONG_PASSWORD", ae.Code)
		assert.True(t, errors.Is(err, apperrors.ErrBadRequest))
	})

	t.Run("too short new password", func(t *testing.T) {
		err := svc.ChangePassword(ctx, id, ChangePasswordInput{CurrentPassword: "password123", NewPassword: "short"})
		var ae *apperrors.AppError
		require.True(t, errors.As(err, &ae))
		assert.Equal(t, "WEAK_PASSWORD", ae.Code)
	})

	t.Run("changes it: old login fails, new login works", func(t *testing.T) {
		require.NoError(t, svc.ChangePassword(ctx, id, ChangePasswordInput{CurrentPassword: "password123", NewPassword: "newpass123"}))
		_, err := svc.Login(ctx, LoginInput{Email: "pw@b.com", Password: "password123"})
		assert.Error(t, err)
		_, err = svc.Login(ctx, LoginInput{Email: "pw@b.com", Password: "newpass123"})
		assert.NoError(t, err)
	})
}

func TestValidateNewPassword(t *testing.T) {
	code := func(err error) string {
		var ae *apperrors.AppError
		if errors.As(err, &ae) {
			return ae.Code
		}
		return ""
	}
	assert.Equal(t, "WEAK_PASSWORD", code(validateNewPassword("short")))
	assert.Equal(t, "PASSWORD_TOO_LONG", code(validateNewPassword(strings.Repeat("x", 73))))
	assert.NoError(t, validateNewPassword(strings.Repeat("x", 72)))
	assert.NoError(t, validateNewPassword("password123"))
}

func TestChangePassword_PasswordlessAccountMustUseSMS(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newTestService()
	phone := "+96171000000"
	u := &User{Phone: &phone, Role: RoleCustomer, Status: StatusActive}
	require.NoError(t, repo.Create(ctx, u))

	err := svc.ChangePassword(ctx, u.ID, ChangePasswordInput{NewPassword: "newpass123"})
	var ae *apperrors.AppError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, "NO_PASSWORD_SET", ae.Code)
	assert.True(t, errors.Is(err, apperrors.ErrBadRequest))
}

func TestRegister_RejectsOverlongPassword(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Register(context.Background(), RegisterInput{Email: "long@b.com", Password: strings.Repeat("p", 80)})
	var ae *apperrors.AppError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, "PASSWORD_TOO_LONG", ae.Code)
}
