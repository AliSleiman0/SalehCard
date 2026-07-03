// Package ratelimit is a small hexagonal rate limiter: a Limiter port with a
// Mongo TTL-counter adapter (default) and a noop adapter, plus a chi middleware
// enforcing a per-key fixed-window cap. It fails OPEN — a backend error allows
// the request (logged) so an infrastructure blip never locks users out.
package ratelimit

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// Limiter reports whether an action under key is within its fixed-window cap,
// counting the attempt. Implementations fail open (return true) on backend error.
type Limiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// Config selects the limiter adapter: "mongo" (default) or "noop"/"off".
type Config struct {
	Provider string
}

// New builds the configured limiter. "noop"/"off" disables limiting; any other
// value uses the Mongo adapter.
func New(cfg Config, db *mongo.Database) Limiter {
	switch cfg.Provider {
	case "noop", "off":
		return NoopLimiter{}
	default:
		return NewMongoLimiter(db)
	}
}

// NoopLimiter allows everything (limiting disabled).
type NoopLimiter struct{}

// Allow always permits the request.
func (NoopLimiter) Allow(context.Context, string, int, time.Duration) (bool, error) {
	return true, nil
}

// ClientIP extracts the caller IP for use as a rate-limit key. middleware.RealIP
// upstream has already normalized r.RemoteAddr to the true client IP.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Middleware enforces `limit` requests per `window` for the key returned by
// keyFn, responding 429 RATE_LIMITED when exceeded. A limiter error fails open.
func Middleware(l Limiter, limit int, window time.Duration, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, err := l.Allow(r.Context(), keyFn(r), limit, window)
			if err != nil {
				slog.Warn("ratelimit: backend error, allowing request", "error", err)
			} else if !ok {
				response.Error(w, http.StatusTooManyRequests, "RATE_LIMITED",
					"too many requests — please slow down and try again shortly")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
