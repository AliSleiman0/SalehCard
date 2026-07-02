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
