package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// Tokens signs and verifies the HMAC tokens appended to Whish's unsigned
// callback URLs. Whish fires the success/failure callback as a bare GET with no
// proof of origin, so the token in the query string (SignExternalID(externalId))
// is the authentication for the public webhook route — see handler.WhishCallback.
type Tokens struct {
	secret []byte
}

// NewTokens constructs a Tokens over the raw HMAC secret (PAYMENTS_HMAC_SECRET).
func NewTokens(secret string) *Tokens {
	return &Tokens{secret: []byte(secret)}
}

// SignExternalID returns the lowercase-hex HMAC-SHA256 of the externalId's
// decimal string.
func (t *Tokens) SignExternalID(externalID int64) string {
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(strconv.FormatInt(externalID, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyExternalID reports whether token is a valid signature for externalId,
// using a constant-time comparison.
func (t *Tokens) VerifyExternalID(externalID int64, token string) bool {
	expected, err := hex.DecodeString(token)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(strconv.FormatInt(externalID, 10)))
	return hmac.Equal(expected, mac.Sum(nil))
}
