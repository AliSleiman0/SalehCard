package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testSecret = "test-secret"

// optionalProbe runs one request through Optional(testSecret) and returns the
// Claims the wrapped handler observed (nil when anonymous), the response
// recorder, and whether the handler ran.
func optionalProbe(t *testing.T, authorization string) (*Claims, *httptest.ResponseRecorder, bool) {
	t.Helper()
	var seen *Claims
	ran := false
	h := Optional(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ran = true
		if c, ok := ClaimsFromContext(r.Context()); ok {
			seen = c
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return seen, rec, ran
}

func TestOptionalNoHeaderProceedsAnonymously(t *testing.T) {
	seen, rec, ran := optionalProbe(t, "")
	if !ran || rec.Code != http.StatusOK {
		t.Fatalf("handler ran=%v code=%d, want ran with 200", ran, rec.Code)
	}
	if seen != nil {
		t.Fatalf("anonymous request carried claims: %+v", seen)
	}
}

func TestOptionalValidTokenAttachesClaims(t *testing.T) {
	token, err := IssueAccessToken(testSecret, Claims{UserID: "507f1f77bcf86cd799439011", Role: "reseller"}, time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	seen, rec, _ := optionalProbe(t, "Bearer "+token)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	if seen == nil || seen.Role != "reseller" || seen.UserID != "507f1f77bcf86cd799439011" {
		t.Fatalf("claims = %+v, want reseller claims attached", seen)
	}
}

func TestOptionalBadTokensProceedAnonymously(t *testing.T) {
	expired, err := IssueAccessToken(testSecret, Claims{UserID: "u1", Role: "customer"}, -time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	wrongSecret, err := IssueAccessToken("other-secret", Claims{UserID: "u1"}, time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	// A pending-2FA token carries no user_id claim — it must never become an
	// identity (mirrors AuthRequired's empty-UserID rejection).
	noUserID, err := IssueAccessToken(testSecret, Claims{Role: "admin"}, time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	for name, header := range map[string]string{
		"garbage":       "Bearer not-a-jwt",
		"expired":       "Bearer " + expired,
		"wrong secret":  "Bearer " + wrongSecret,
		"empty user id": "Bearer " + noUserID,
		"not bearer":    "Basic dXNlcjpwYXNz",
	} {
		t.Run(name, func(t *testing.T) {
			seen, rec, ran := optionalProbe(t, header)
			if !ran || rec.Code != http.StatusOK {
				t.Fatalf("handler ran=%v code=%d, want anonymous 200", ran, rec.Code)
			}
			if seen != nil {
				t.Fatalf("claims attached from an invalid token: %+v", seen)
			}
		})
	}
}

func TestOptionalSetsVaryAuthorization(t *testing.T) {
	_, rec, _ := optionalProbe(t, "")
	for _, v := range rec.Header().Values("Vary") {
		if v == "Authorization" {
			return
		}
	}
	t.Fatalf("Vary header = %q, want to include Authorization", rec.Header().Values("Vary"))
}
