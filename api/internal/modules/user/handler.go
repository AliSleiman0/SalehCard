package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// refreshCookieName is the name of the httpOnly refresh-token cookie.
const refreshCookieName = "refresh_token"

// mobileClientHeader and mobileClientValue identify native clients that receive
// the refresh token in the response body instead of (only) the httpOnly cookie.
const mobileClientHeader = "X-Client"
const mobileClientValue = "mobile"

// isMobile reports whether the request comes from a native client that needs the
// refresh token in the JSON body (it cannot read the httpOnly cookie).
func isMobile(r *http.Request) bool {
	return r.Header.Get(mobileClientHeader) == mobileClientValue
}

// refreshCookiePath scopes the cookie to the auth endpoints that consume it.
const refreshCookiePath = "/api/v1/auth"

// Handler exposes user domain operations over HTTP.
type Handler struct {
	service    Service
	secure     bool
	refreshTTL time.Duration
	// rec records customer self-deletions to the admin audit log (the actor is
	// the customer, from the request claims). Nil-safe: nil skips recording.
	rec audit.Recorder
}

// NewHandler constructs a Handler. secure marks the refresh cookie Secure
// (enabled outside development); refreshTTL sets the cookie's Max-Age so the
// refresh cookie persists across browser restarts (it must match the server-side
// refresh-token lifetime).
func NewHandler(service Service, secure bool, refreshTTL time.Duration) *Handler {
	return &Handler{service: service, secure: secure, refreshTTL: refreshTTL}
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	res, err := h.service.Register(r.Context(), in)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeAuth(w, r, res)
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in LoginInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	res, err := h.service.Login(r.Context(), in)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeAuth(w, r, res)
}

// LoginPhone handles POST /api/v1/auth/login-phone (phone + password).
func (h *Handler) LoginPhone(w http.ResponseWriter, r *http.Request) {
	var in PhoneLoginInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	res, err := h.service.LoginByPhone(r.Context(), in)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeAuth(w, r, res)
}

// RequestOTP handles POST /api/v1/auth/otp/request, sending a one-time code.
func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var in RequestOTPInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := h.service.RequestOTP(r.Context(), in); err != nil {
		writeAuthError(w, err)
		return
	}
	response.OK(w, map[string]bool{"sent": true})
}

// VerifyOTP handles POST /api/v1/auth/otp/verify, authenticating on a valid code.
func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var in VerifyOTPInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	res, err := h.service.VerifyOTP(r.Context(), in)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeAuth(w, r, res)
}

// VerifyTwoFactor handles POST /api/v1/auth/2fa/verify, completing an admin login
// by validating the pending challenge token + SMS code and issuing the session.
func (h *Handler) VerifyTwoFactor(w http.ResponseWriter, r *http.Request) {
	var in VerifyTwoFactorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	res, err := h.service.VerifyAdmin2FA(r.Context(), in)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeAuth(w, r, res)
}

// ResendTwoFactor handles POST /api/v1/auth/2fa/resend, re-sending the SMS code
// for an in-progress admin login. It returns a fresh challenge (new pending
// token + masked phone), not a session.
func (h *Handler) ResendTwoFactor(w http.ResponseWriter, r *http.Request) {
	var in ResendTwoFactorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	res, err := h.service.ResendAdmin2FA(r.Context(), in)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeAuth(w, r, res)
}

// Refresh handles POST /api/v1/auth/refresh, rotating the refresh token. Native
// clients present the token in the JSON body; browser clients omit it and the
// token is read from the httpOnly cookie.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	token := h.refreshTokenFromRequest(r)
	if token == "" {
		response.Unauthorized(w, "missing refresh token")
		return
	}
	res, err := h.service.Refresh(r.Context(), token)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeAuth(w, r, res)
}

// refreshTokenFromRequest extracts the refresh token, preferring an explicit
// JSON body (native clients) and falling back to the httpOnly cookie (browsers).
func (h *Handler) refreshTokenFromRequest(r *http.Request) string {
	if r.Body != nil {
		var in RefreshInput
		if err := json.NewDecoder(r.Body).Decode(&in); err == nil && in.RefreshToken != "" {
			return in.RefreshToken
		}
	}
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		return cookie.Value
	}
	return ""
}

// Logout handles POST /api/v1/auth/logout, revoking and clearing the cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		_ = h.service.Logout(r.Context(), cookie.Value)
	}
	h.clearRefreshCookie(w)
	response.OK(w, map[string]bool{"success": true})
}

// GetProfile handles GET /api/v1/users/me.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	user, err := h.service.GetProfile(r.Context(), id)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	response.OK(w, user)
}

// UpdateProfile handles PATCH /api/v1/users/me.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	var in UpdateProfileInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	user, err := h.service.UpdateProfile(r.Context(), id, in)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	response.OK(w, user)
}

// DeleteMe handles DELETE /api/v1/users/me — customer self-service account
// deletion (Play Store requirement). Guard refusals surface as 409/403
// AppErrors via writeAuthError; success is idempotent.
func (h *Handler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "authentication required")
		return
	}
	hadKyc, err := h.service.DeleteAccount(r.Context(), id)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if h.rec != nil {
		// Actor pre-set to the bare user id: the recorder must NOT fall back to
		// the claims, which still carry the email/phone this request just erased
		// — an audit row is forever. No PII in the summary either.
		h.rec.Record(r.Context(), audit.Entry{
			ActorID:    id.Hex(),
			ActorEmail: "customer (self-deleted)",
			Action:     audit.ActionUserSelfDelete,
			TargetType: "user",
			TargetID:   id.Hex(),
			Summary:    map[string]any{"hadKyc": hadKyc},
		})
	}
	response.OK(w, map[string]bool{"deleted": true})
}

// writeAuth sets the rotated refresh cookie and writes the AuthResponse body.
// For native clients (X-Client: mobile) the refresh token is also returned in the
// body, since they cannot read the httpOnly cookie.
func (h *Handler) writeAuth(w http.ResponseWriter, r *http.Request, res *AuthResult) {
	if res.TwoFactorRequired {
		// Second factor pending: no session yet, so issue no refresh cookie/tokens —
		// only the challenge the client completes via /auth/2fa/verify.
		response.OK(w, AuthResponse{
			TwoFactorRequired: true,
			PendingToken:      res.PendingToken,
			PhoneHint:         res.PhoneHint,
		})
		return
	}
	h.setRefreshCookie(w, res.RefreshToken)
	body := AuthResponse{AccessToken: res.AccessToken, User: res.User}
	if isMobile(r) {
		body.RefreshToken = res.RefreshToken
	}
	response.OK(w, body)
}

// refreshCookieSameSite picks the SameSite policy for the refresh cookie. In
// production the storefront/admin SPAs and the API are served from different
// sites (*.azurestaticapps.net vs *.azurewebsites.net), so the cookie must be
// SameSite=None (always paired with Secure) to be sent on the SPA's cross-site
// /auth/refresh and authenticated requests. In local dev everything is
// same-site on localhost and Secure is off — where browsers reject None — so
// fall back to Lax.
func (h *Handler) refreshCookieSameSite() http.SameSite {
	if h.secure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: h.refreshCookieSameSite(),
		// Persist across browser restarts; without Max-Age this is a session
		// cookie and the admin SPA loses its session when the browser closes.
		MaxAge: int(h.refreshTTL.Seconds()),
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: h.refreshCookieSameSite(),
		MaxAge:   -1,
	})
}

// writeAuthError maps domain errors to HTTP responses.
func writeAuthError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		// AppError wraps a sentinel; classify by the wrapped cause.
		switch {
		case errors.Is(appErr, apperrors.ErrBadRequest):
			response.Error(w, http.StatusBadRequest, appErr.Code, appErr.Message)
		case errors.Is(appErr, apperrors.ErrConflict):
			response.Error(w, http.StatusConflict, appErr.Code, appErr.Message)
		case errors.Is(appErr, apperrors.ErrUnauthorized):
			response.Error(w, http.StatusUnauthorized, appErr.Code, appErr.Message)
		case errors.Is(appErr, apperrors.ErrForbidden):
			response.Error(w, http.StatusForbidden, appErr.Code, appErr.Message)
		default:
			response.InternalError(w)
		}
		return
	}

	switch {
	case errors.Is(err, apperrors.ErrConflict):
		response.Error(w, http.StatusConflict, "EMAIL_TAKEN", "an account with this email already exists")
	case errors.Is(err, apperrors.ErrUnauthorized):
		response.Unauthorized(w, "invalid credentials")
	case errors.Is(err, apperrors.ErrBadRequest):
		response.BadRequest(w, err.Error())
	case errors.Is(err, apperrors.ErrNotFound):
		response.NotFound(w)
	default:
		response.InternalError(w)
	}
}
