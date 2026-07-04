package idcheck

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestVerifier points a rapidAPIVerifier at a stub server.
func newTestVerifier(baseURL string) *rapidAPIVerifier {
	return &rapidAPIVerifier{key: "test-key", baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
}

func TestRapidAPIVerify_Found(t *testing.T) {
	// Mirrors the real PUBG Mobile Global response (includes is_ban).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-rapidapi-key"); got != "test-key" {
			t.Errorf("missing/wrong x-rapidapi-key header: %q", got)
		}
		if got := r.Header.Get("x-rapidapi-host"); got != rapidAPIHost {
			t.Errorf("wrong x-rapidapi-host: %q", got)
		}
		if r.URL.Path != "/pubgm-global/5204837417" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":false,"status":200,"msg":"id_found","data":{"id":"5204837417","username":"Agus haji","is_ban":false}}`))
	}))
	defer srv.Close()

	acct, err := newTestVerifier(srv.URL).Verify(context.Background(), "pubgm-global", "5204837417")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acct.Username != "Agus haji" {
		t.Errorf("username = %q, want %q", acct.Username, "Agus haji")
	}
	if acct.Banned {
		t.Error("Banned = true, want false")
	}
}

func TestRapidAPIVerify_FoundWithoutBanField(t *testing.T) {
	// Mirrors the real Delta Force Garena response (no is_ban field → Banned false).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":false,"status":200,"msg":"id_found","data":{"id":"182200303107200135203","username":"SoyBlaze"}}`))
	}))
	defer srv.Close()

	acct, err := newTestVerifier(srv.URL).Verify(context.Background(), "dfm-garena", "182200303107200135203")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acct.Username != "SoyBlaze" || acct.Banned {
		t.Errorf("got %+v, want {SoyBlaze false}", acct)
	}
}

func TestRapidAPIVerify_NotFound(t *testing.T) {
	// The real API returns id_not_found over HTTP 200 — the body, not the status,
	// is authoritative. Must map to ErrIDNotFound (a clean "bad ID").
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":true,"status":404,"msg":"id_not_found"}`))
	}))
	defer srv.Close()

	_, err := newTestVerifier(srv.URL).Verify(context.Background(), "pubgm-global", "0000000000")
	if !errors.Is(err, ErrIDNotFound) {
		t.Fatalf("error = %v, want ErrIDNotFound", err)
	}
}

func TestRapidAPIVerify_UpstreamErrorIsNotNotFound(t *testing.T) {
	// A real transport/auth/quota failure (non-2xx) must NOT look like a bad ID —
	// the caller fails open on it, so it must be a generic error, not ErrIDNotFound.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"You are not subscribed to this API."}`))
	}))
	defer srv.Close()

	_, err := newTestVerifier(srv.URL).Verify(context.Background(), "pubgm-global", "5204837417")
	if err == nil {
		t.Fatal("expected an error on HTTP 403")
	}
	if errors.Is(err, ErrIDNotFound) {
		t.Fatal("HTTP 403 must not map to ErrIDNotFound (would wrongly block the sale)")
	}
}

func TestNew_Selection(t *testing.T) {
	if v, err := New(Config{}); err != nil || v == nil {
		t.Fatalf("empty provider should default to stub: v=%v err=%v", v, err)
	}
	if _, err := New(Config{Provider: "stub"}); err != nil {
		t.Fatalf("stub: %v", err)
	}
	if _, err := New(Config{Provider: "rapidapi"}); err == nil {
		t.Fatal("rapidapi without a key should error")
	}
	if _, err := New(Config{Provider: "rapidapi", RapidAPIKey: "k"}); err != nil {
		t.Fatalf("rapidapi with a key: %v", err)
	}
	if _, err := New(Config{Provider: "bogus"}); err == nil {
		t.Fatal("unknown provider should error")
	}
}

func TestStubVerifier(t *testing.T) {
	acct, err := StubVerifier{}.Verify(context.Background(), "pubgm-global", "123")
	if err != nil || acct.Username == "" {
		t.Fatalf("stub should resolve any non-empty id: acct=%+v err=%v", acct, err)
	}
	if _, err := (StubVerifier{}).Verify(context.Background(), "pubgm-global", ""); !errors.Is(err, ErrIDNotFound) {
		t.Fatalf("stub empty id: err=%v, want ErrIDNotFound", err)
	}
}
