package user

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// maxNameLen bounds an admin-edited display name.
const maxNameLen = 80

// ChangePasswordInput is the self-service password change payload.
type ChangePasswordInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

var errWrongPassword = &apperrors.AppError{
	Code: "WRONG_PASSWORD", Message: "current password is incorrect",
	// 400, not 401: a 401 makes the clients drop the session.
	Err: apperrors.ErrBadRequest,
}

// maxPasswordBytes is bcrypt's input limit; longer passwords make
// GenerateFromPassword fail, which would otherwise surface as a 500.
const maxPasswordBytes = 72

var errNoPasswordSet = &apperrors.AppError{
	Code:    "NO_PASSWORD_SET",
	Message: "this account has no password yet — set one by signing in with an SMS code",
	Err:     apperrors.ErrBadRequest,
}

// validateNewPassword enforces the password policy for any new password:
// at least 8 characters and at most 72 bytes (bcrypt's limit).
func validateNewPassword(pw string) error {
	if len(pw) < minPasswordLen {
		return &apperrors.AppError{Code: "WEAK_PASSWORD", Message: "password must be at least 8 characters", Err: apperrors.ErrBadRequest}
	}
	if len(pw) > maxPasswordBytes {
		return &apperrors.AppError{Code: "PASSWORD_TOO_LONG", Message: "password must be at most 72 characters", Err: apperrors.ErrBadRequest}
	}
	return nil
}

// ChangePassword lets a signed-in user (customer or admin) replace their own
// password; the current one is always required. A phone-OTP account that never
// set a password gets NO_PASSWORD_SET — it sets its first one through the SMS
// sign-in (VerifyOTP with a password), so a stolen session alone can't.
// Other sessions are left alone — the caller's own refresh token can't be told
// apart here, and revoking everything would sign the caller out too.
func (s *UserService) ChangePassword(ctx context.Context, id bson.ObjectID, in ChangePasswordInput) error {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if isDeleted(u) {
		return apperrors.ErrUnauthorized
	}
	if err := validateNewPassword(in.NewPassword); err != nil {
		return err
	}
	if u.PasswordHash == nil {
		return errNoPasswordSet
	}
	if bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(in.CurrentPassword)) != nil {
		return errWrongPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.SetPassword(ctx, id, string(hash))
}

// ChangePassword handles POST /api/v1/users/me/password.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	var in ChangePasswordInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := h.service.ChangePassword(r.Context(), id, in); err != nil {
		writeAuthError(w, err)
		return
	}
	if h.rec != nil {
		h.rec.Record(r.Context(), audit.Entry{Action: audit.ActionPasswordChange, TargetType: "user", TargetID: id.Hex()})
	}
	response.OK(w, map[string]bool{"changed": true})
}

// guardAdminTarget enforces that only a super admin may edit another admin
// account (an admin with users.manage must not be able to take over a super
// admin by resetting their password).
func guardAdminTarget(w http.ResponseWriter, r *http.Request, target *User) bool {
	if target.Role != RoleAdmin {
		return true
	}
	if claims, _ := auth.ClaimsFromContext(r.Context()); claims == nil || !claims.IsSuperAdmin() {
		response.Forbidden(w, "only a super admin can edit an admin account")
		return false
	}
	return true
}

// updateProfile handles PUT /api/admin/users/{id}/profile — an admin edits a
// customer's display name.
func (a *adminHandler) updateProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || len([]rune(name)) > maxNameLen {
		response.BadRequest(w, "name must be 1–80 characters")
		return
	}
	current, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		a.writeRepoError(w, err)
		return
	}
	if isDeleted(current) {
		response.BadRequest(w, "this account has been deleted")
		return
	}
	if !guardAdminTarget(w, r, current) {
		return
	}
	if _, err := a.users.UpdateOne(r.Context(), bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: name}}}}); err != nil {
		response.InternalError(w)
		return
	}
	a.rec.Record(r.Context(), audit.Entry{
		Action:     audit.ActionUserProfileEdit,
		TargetType: "user",
		TargetID:   id.Hex(),
		Summary:    map[string]any{"from": current.Name, "to": name},
	})
	current.Name = name
	response.OK(w, current)
}

// setPassword handles POST /api/admin/users/{id}/password — an admin sets a new
// password for a customer (e.g. one who lost access). Every session the user
// has is revoked so the old credentials stop working immediately.
func (a *adminHandler) setPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := validateNewPassword(body.NewPassword); err != nil {
		writeAuthError(w, err)
		return
	}
	current, err := a.repo.FindByID(r.Context(), id)
	if err != nil {
		a.writeRepoError(w, err)
		return
	}
	if isDeleted(current) {
		response.BadRequest(w, "this account has been deleted")
		return
	}
	if !guardAdminTarget(w, r, current) {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		response.InternalError(w)
		return
	}
	if err := a.repo.SetPassword(r.Context(), id, string(hash)); err != nil {
		a.writeRepoError(w, err)
		return
	}
	if err := a.refresh.RevokeAllForUser(r.Context(), id); err != nil {
		slog.Error("user: failed to revoke sessions after admin password reset", "userId", id.Hex(), "error", err)
	}
	a.rec.Record(r.Context(), audit.Entry{Action: audit.ActionUserPasswordReset, TargetType: "user", TargetID: id.Hex()})
	response.OK(w, map[string]bool{"changed": true})
}
