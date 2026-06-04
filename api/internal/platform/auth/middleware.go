package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// ctxKey is an unexported type for context keys defined in this package.
type ctxKey int

const claimsKey ctxKey = iota

// devWarnOnce ensures the dev-bypass warning is logged at most once.
var devWarnOnce sync.Once

// AdminOnly returns middleware that authorizes requests carrying a JWT whose
// role claim is "admin", attaching the verified Claims to the request context.
//
// Development bypass: when secret is empty (no JWT_SECRET configured, as in the
// default local setup) the middleware lets requests through with a synthetic
// admin identity so the wired product/inventory pages work end-to-end without a
// token-issuing auth module. It logs a one-time warning. In any environment
// where a secret IS set, the middleware strictly enforces a valid admin token.
func AdminOnly(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
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

// ClaimsFromContext returns the Claims attached by AdminOnly, if present.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}
