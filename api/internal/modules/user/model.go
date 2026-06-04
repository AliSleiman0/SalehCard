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
