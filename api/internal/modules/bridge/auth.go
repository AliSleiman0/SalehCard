package bridge

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// tokenPrefix marks a bridge device token so it's recognizable in logs/support.
const tokenPrefix = "bd_"

// ctxKey is the private context key type for the authenticated device.
type ctxKey int

const deviceCtxKey ctxKey = 0

// GenerateToken mints a new device token: a 256-bit random secret returned in
// plaintext (shown to the admin ONCE) alongside its storage hash. Only the hash
// is ever persisted, so a database leak cannot recover a usable token.
func GenerateToken() (plain, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plain = tokenPrefix + base64.RawURLEncoding.EncodeToString(b)
	return plain, HashToken(plain), nil
}

// HashToken returns the hex SHA-256 of a token — the value stored and looked up.
// A plain hash (not bcrypt) is appropriate here: the token is high-entropy
// random, so there is nothing to brute-force, and the constant-length hex output
// makes the indexed equality lookup a non-secret comparison.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

// DeviceAuth authenticates device requests by their bearer token, attaching the
// resolved *Device to the request context. A disabled or unknown device is 401.
func DeviceAuth(store DeviceStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				response.Unauthorized(w, "device token required")
				return
			}
			d, err := store.FindDeviceByTokenHash(r.Context(), HashToken(token))
			if err != nil || d == nil || !d.Enabled {
				response.Unauthorized(w, "invalid or disabled device token")
				return
			}
			ctx := context.WithValue(r.Context(), deviceCtxKey, d)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DeviceFromContext returns the authenticated device attached by DeviceAuth.
func DeviceFromContext(ctx context.Context) (*Device, bool) {
	d, ok := ctx.Value(deviceCtxKey).(*Device)
	return d, ok
}

// bearerToken extracts a token from the Authorization: Bearer header.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
