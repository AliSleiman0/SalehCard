package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubLimiter struct {
	allow bool
	err   error
}

func (s stubLimiter) Allow(context.Context, string, int, time.Duration) (bool, error) {
	return s.allow, s.err
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func serve(l Limiter) int {
	h := Middleware(l, 1, time.Minute, ClientIP)(okHandler())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", nil))
	return rec.Code
}

func TestMiddleware_BlocksWhenOverLimit(t *testing.T) {
	if code := serve(stubLimiter{allow: false}); code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", code)
	}
}

func TestMiddleware_AllowsWhenUnderLimit(t *testing.T) {
	if code := serve(stubLimiter{allow: true}); code != http.StatusOK {
		t.Fatalf("got %d, want 200", code)
	}
}

func TestMiddleware_FailsOpenOnBackendError(t *testing.T) {
	// A backend error must not block the request (fail open).
	if code := serve(stubLimiter{allow: false, err: context.DeadlineExceeded}); code != http.StatusOK {
		t.Fatalf("got %d, want 200 (fail-open)", code)
	}
}

func TestNoopLimiter_AlwaysAllows(t *testing.T) {
	ok, err := NoopLimiter{}.Allow(context.Background(), "k", 1, time.Minute)
	if !ok || err != nil {
		t.Fatalf("noop should allow: ok=%v err=%v", ok, err)
	}
}
