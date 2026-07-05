package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the JWT registered claims plus application-specific fields.
type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Role   string `json:"role"`
}

// ActorLabel is a never-blank identifier for the claim holder, for attributing
// admin actions in audit logs and top-up decisions. It prefers email, then phone
// (phone-only admins have no email), then the user id as a last resort.
func ActorLabel(c *Claims) string {
	if c == nil {
		return ""
	}
	if c.Email != "" {
		return c.Email
	}
	if c.Phone != "" {
		return c.Phone
	}
	return c.UserID
}

// IssueAccessToken signs a new JWT with HS256 and the given expiry duration.
func IssueAccessToken(secret string, c Claims, expiry time.Duration) (string, error) {
	c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(expiry))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(secret))
}

// twoFAPurpose tags a pending two-factor challenge token, distinguishing it from
// a normal access token so it can never be replayed as a bearer credential.
const twoFAPurpose = "2fa"

// twoFAClaims is the short-lived token minted after an admin's password is
// verified but before the SMS second factor. Its shape is deliberately distinct
// from Claims (no role/user_id) so that even if presented as a bearer token it
// yields empty Claims and fails AdminOnly/AuthRequired.
type twoFAClaims struct {
	jwt.RegisteredClaims
	UserID  string `json:"uid"`
	Purpose string `json:"purpose"`
}

// Issue2FAToken signs a short-lived pending-2FA token binding userID.
func Issue2FAToken(secret, userID string, ttl time.Duration) (string, error) {
	claims := twoFAClaims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl))},
		UserID:           userID,
		Purpose:          twoFAPurpose,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Verify2FAToken parses a pending-2FA token and returns the bound userID. It
// rejects tokens whose purpose is not "2fa" (e.g. a regular access token) and
// tokens with no userID.
func Verify2FAToken(secret, tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &twoFAClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*twoFAClaims)
	if !ok || !token.Valid || claims.Purpose != twoFAPurpose || claims.UserID == "" {
		return "", fmt.Errorf("invalid 2fa token")
	}
	return claims.UserID, nil
}

// VerifyToken parses and validates a JWT string, returning the typed Claims on success.
func VerifyToken(secret, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
