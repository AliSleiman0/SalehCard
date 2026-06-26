package user

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Role represents the access level of a user.
type Role string

const (
	RoleCustomer Role = "customer"
	RoleReseller Role = "reseller"
	RoleAdmin    Role = "admin"
)

// User is the primary account entity. Either Email (email/password signup) or
// Phone (phone-OTP signup) identifies the account; both carry a sparse-unique
// index so phone-only accounts may omit the email and vice versa.
type User struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email          string        `bson:"email,omitempty" json:"email"`
	Phone          *string       `bson:"phone,omitempty" json:"phone,omitempty"`
	PasswordHash   *string       `bson:"passwordHash"  json:"-"`
	GoogleID       *string       `bson:"googleId,omitempty" json:"googleId,omitempty"`
	Role           Role          `bson:"role"          json:"role"`
	Locale         string        `bson:"locale"        json:"locale"`
	SavedPlayerIDs []string      `bson:"savedPlayerIds" json:"savedPlayerIds"`
	WalletBalance  float64       `bson:"walletBalance" json:"walletBalance"`
	LoyaltyPoints  int           `bson:"loyaltyPoints" json:"loyaltyPoints"`
	CreatedAt      time.Time     `bson:"createdAt"     json:"createdAt"`
	UpdatedAt      time.Time     `bson:"updatedAt"     json:"updatedAt"`
}

// OtpCode is a pending one-time passcode for a phone number. Only the SHA-256
// hash of the code is stored. One active record per phone (upserted on request).
type OtpCode struct {
	ID         bson.ObjectID `bson:"_id,omitempty"`
	Phone      string        `bson:"phone"`
	CodeHash   string        `bson:"codeHash"`
	ExpiresAt  time.Time     `bson:"expiresAt"`
	Attempts   int           `bson:"attempts"`
	CreatedAt  time.Time     `bson:"createdAt"`
	ConsumedAt *time.Time    `bson:"consumedAt,omitempty"`
}

// RegisterInput holds the data required to create a new account.
type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Locale   string `json:"locale"`
}

// LoginInput holds the credentials for an email/password login.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RequestOTPInput is the body of POST /auth/otp/request.
type RequestOTPInput struct {
	Phone string `json:"phone"`
}

// VerifyOTPInput is the body of POST /auth/otp/verify. Password is optional: when
// present on a brand-new account it sets the user's password (signup flow), so
// the account can later log in by phone+password as well as by OTP.
type VerifyOTPInput struct {
	Phone    string `json:"phone"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

// PhoneLoginInput holds the credentials for a phone+password login.
type PhoneLoginInput struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// UpdateProfileInput carries the fields a user may change on their own account.
type UpdateProfileInput struct {
	Locale         *string  `json:"locale,omitempty"`
	SavedPlayerIDs []string `json:"savedPlayerIds,omitempty"`
}

// RefreshToken is a server-side record of an issued refresh token. Only the
// SHA-256 hash of the opaque token is stored; the raw value lives solely in the
// client's httpOnly cookie. Tokens are rotated on every use.
type RefreshToken struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"-"`
	UserID    bson.ObjectID `bson:"userId"        json:"-"`
	TokenHash string        `bson:"tokenHash"     json:"-"`
	ExpiresAt time.Time     `bson:"expiresAt"     json:"-"`
	CreatedAt time.Time     `bson:"createdAt"     json:"-"`
	RevokedAt *time.Time    `bson:"revokedAt,omitempty" json:"-"`
}

// AuthResult is the internal result of an authentication operation. The raw
// RefreshToken is handed to the handler so it can set the httpOnly cookie; it is
// never serialized to JSON.
type AuthResult struct {
	User         *User
	AccessToken  string
	RefreshToken string
}

// AuthResponse is the public JSON body returned by register/login/refresh.
// RefreshToken is populated only for native clients (X-Client: mobile), which
// cannot rely on the httpOnly refresh cookie; browser clients receive it solely
// via that cookie and the field is omitted.
type AuthResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	User         *User  `json:"user"`
}

// RefreshInput is the optional JSON body accepted by POST /auth/refresh. Native
// clients present the refresh token here; browser clients omit it and the token
// is read from the httpOnly cookie instead.
type RefreshInput struct {
	RefreshToken string `json:"refreshToken"`
}
