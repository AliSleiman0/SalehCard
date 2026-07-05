package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ctxKey is an unexported type for context keys defined in this package.
type ctxKey int

const claimsKey ctxKey = iota

// devWarnOnce ensures the dev-bypass warning is logged at most once.
var devWarnOnce sync.Once

// AdminOnly returns middleware that authorizes requests carrying a JWT whose
// role claim is "admin", attaching the verified Claims to the request context.
//
// Development bypass: only when devBypass is true (the caller confirms
// ENV=development) AND secret is empty does the middleware let requests through
// with a synthetic admin identity, so the admin console works locally without a
// token-issuing setup. It logs a one-time warning. In every other case —
// including an empty secret outside development — requests without a valid
// admin token are rejected (an empty secret verifies no token, so all requests
// get 401; config.Validate additionally refuses to boot in that state).
func AdminOnly(secret string, devBypass bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if devBypass && secret == "" {
				devWarnOnce.Do(func() {
					slog.Warn("AdminOnly: JWT_SECRET is empty — admin auth is BYPASSED (development only)")
				})
				ctx := context.WithValue(r.Context(), claimsKey, &Claims{
					UserID: "dev-admin",
					Email:  "dev@salehcard.local",
					Role:   "admin",
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				response.Unauthorized(w, "missing bearer token")
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")

			claims, err := VerifyToken(secret, token)
			if err != nil {
				response.Unauthorized(w, "invalid or expired token")
				return
			}
			if claims.Role != "admin" {
				response.Forbidden(w, "admin role required")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthRequired returns middleware that authorizes any request carrying a valid
// (non-expired) JWT, attaching the verified Claims to the request context. Unlike
// AdminOnly it has no dev-bypass and no role restriction — it is the gate for
// customer-facing authenticated routes (e.g. /api/v1/users/me).
func AuthRequired(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				response.Unauthorized(w, "missing bearer token")
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")

			claims, err := VerifyToken(secret, token)
			if err != nil {
				response.Unauthorized(w, "invalid or expired token")
				return
			}
			// Reject tokens without a user id (e.g. a pending 2FA challenge token
			// replayed here — it carries no user_id claim) so they can't act as a
			// bearer credential.
			if claims.UserID == "" {
				response.Unauthorized(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext returns the Claims attached by AdminOnly, if present.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}

// ContextWithClaims returns a copy of ctx carrying c under the same key the auth
// middleware uses, so handler tests can exercise claims-dependent logic without
// wiring up the full middleware chain.
func ContextWithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

// UserIDFromContext returns the authenticated user's ObjectID from the Claims
// attached by AuthRequired/AdminOnly. The bool is false when no valid claims are
// present or the UserID is not a valid ObjectID.
func UserIDFromContext(ctx context.Context) (bson.ObjectID, bool) {
	c, ok := ClaimsFromContext(ctx)
	if !ok {
		return bson.NilObjectID, false
	}
	id, err := bson.ObjectIDFromHex(c.UserID)
	if err != nil {
		return bson.NilObjectID, false
	}
	return id, true
}
