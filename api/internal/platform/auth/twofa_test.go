package auth

import (
	"testing"
	"time"
)

func TestIssueVerify2FAToken_RoundTrip(t *testing.T) {
	tok, err := Issue2FAToken("secret", "user-123", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	uid, err := Verify2FAToken("secret", tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if uid != "user-123" {
		t.Fatalf("uid = %q, want user-123", uid)
	}
}

func TestVerify2FAToken_RejectsExpired(t *testing.T) {
	tok, err := Issue2FAToken("secret", "user-123", -time.Minute) // already expired
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := Verify2FAToken("secret", tok); err == nil {
		t.Fatal("expected expired 2FA token to be rejected")
	}
}

func TestVerify2FAToken_RejectsWrongSecret(t *testing.T) {
	tok, _ := Issue2FAToken("secret", "user-123", time.Minute)
	if _, err := Verify2FAToken("other", tok); err == nil {
		t.Fatal("expected wrong-secret 2FA token to be rejected")
	}
}

// A normal access token must not be accepted by Verify2FAToken (wrong purpose),
// and a 2FA token must not verify as an access token (no user_id/role claims).
func TestTokenTypesDoNotCross(t *testing.T) {
	access, err := IssueAccessToken("secret", Claims{UserID: "u1", Role: "admin"}, time.Minute)
	if err != nil {
		t.Fatalf("issue access: %v", err)
	}
	if _, err := Verify2FAToken("secret", access); err == nil {
		t.Fatal("access token must not pass Verify2FAToken")
	}

	pending, _ := Issue2FAToken("secret", "u1", time.Minute)
	claims, err := VerifyToken("secret", pending)
	if err != nil {
		// Signature is valid (same secret), so it parses — but it must carry no
		// usable identity/role.
		t.Fatalf("unexpected parse error: %v", err)
	}
	if claims.UserID != "" || claims.Role != "" {
		t.Fatalf("2FA token leaked identity as access claims: uid=%q role=%q", claims.UserID, claims.Role)
	}
}
