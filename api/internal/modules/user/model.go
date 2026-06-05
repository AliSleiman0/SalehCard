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

// User is the primary account entity.
type User struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email          string        `bson:"email"         json:"email"`
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
type AuthResponse struct {
	AccessToken string `json:"accessToken"`
	User        *User  `json:"user"`
}
