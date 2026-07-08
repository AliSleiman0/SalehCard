package user

import (
	"fmt"
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

// Status represents the account state. An empty status (on accounts created
// before this field existed) is treated as active everywhere it is read.
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	// StatusDeleted marks a soft-deleted (anonymized) account. The row is kept
	// so orders and ledger rows keep resolving; the account can no longer sign
	// in and is hidden from the default admin listing.
	StatusDeleted Status = "deleted"
)

// User is the primary account entity. Either Email (email/password signup) or
// Phone (phone-OTP signup) identifies the account; both carry a sparse-unique
// index so phone-only accounts may omit the email and vice versa.
type User struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name           string        `bson:"name,omitempty" json:"name"`
	Email          string        `bson:"email,omitempty" json:"email"`
	Phone          *string       `bson:"phone,omitempty" json:"phone,omitempty"`
	PasswordHash   *string       `bson:"passwordHash"  json:"-"`
	GoogleID       *string       `bson:"googleId,omitempty" json:"googleId,omitempty"`
	Role           Role          `bson:"role"          json:"role"`
	// AdminRoleID references the custom RBAC role (roles collection) assigned to
	// an admin account. Nil on an admin means built-in Super Admin (all
	// permissions). Meaningless for customers/resellers.
	AdminRoleID *bson.ObjectID `bson:"adminRoleId,omitempty" json:"adminRoleId,omitempty"`
	// Permissions is the resolved RBAC permission set for an admin ("*" = super
	// admin). Never persisted — populated at token-issue time so auth responses
	// carry it to the admin console.
	Permissions []string `bson:"-" json:"permissions,omitempty"`
	Status      Status   `bson:"status,omitempty" json:"status"`
	// ResellerTier is the name of the reseller tier (Bronze/Silver/Gold) this
	// account belongs to. Empty for non-resellers and unassigned resellers.
	ResellerTier   string        `bson:"resellerTier,omitempty" json:"resellerTier,omitempty"`
	Locale         string          `bson:"locale"        json:"locale"`
	SavedPlayerIDs []SavedPlayerID `bson:"savedPlayerIds" json:"savedPlayerIds"`
	WalletBalance  float64         `bson:"walletBalance" json:"walletBalance"`
	LoyaltyPoints  int           `bson:"loyaltyPoints" json:"loyaltyPoints"`
	CreatedAt      time.Time     `bson:"createdAt"     json:"createdAt"`
	UpdatedAt      time.Time     `bson:"updatedAt"     json:"updatedAt"`
	// LastSeen is refreshed whenever the account authenticates (login / OTP /
	// token refresh). It powers the "active users" dashboard metric; absent on
	// accounts that have not signed in since the field was introduced.
	LastSeen time.Time `bson:"lastSeen,omitempty" json:"lastSeen,omitempty"`
	// DeletedAt stamps when the account was soft-deleted (anonymized). Nil for
	// live accounts.
	DeletedAt *time.Time `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
}

// SavedPlayerID is a player/account ID the customer has saved for faster
// checkout, tagged with a human label (e.g. "PUBG main"). Both fields are
// required when written.
type SavedPlayerID struct {
	Label string `bson:"label" json:"label"`
	Value string `bson:"value" json:"value"`
}

// UnmarshalBSONValue decodes a saved player ID tolerantly: legacy documents
// stored the list as bare strings (before labels existed), so a BSON string is
// accepted and mirrored into both Label and Value. New documents decode from
// the embedded {label, value} document. Legacy rows are rewritten in the new
// shape on the user's next profile save.
func (p *SavedPlayerID) UnmarshalBSONValue(typ byte, data []byte) error {
	rv := bson.RawValue{Type: bson.Type(typ), Value: data}
	switch rv.Type {
	case bson.TypeString:
		s := rv.StringValue()
		p.Label, p.Value = s, s
		return nil
	case bson.TypeEmbeddedDocument:
		// Decode into an alias so this method is not called recursively.
		var doc struct {
			Label string `bson:"label"`
			Value string `bson:"value"`
		}
		if err := rv.Unmarshal(&doc); err != nil {
			return err
		}
		p.Label, p.Value = doc.Label, doc.Value
		return nil
	case bson.TypeNull, bson.TypeUndefined:
		return nil
	default:
		return fmt.Errorf("user: cannot decode BSON %s into SavedPlayerID", rv.Type)
	}
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
	Name     string `json:"name"`
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
	// Name is captured at signup (first sign-in) and set on the new account; it
	// is ignored when the phone already maps to an existing user (OTP login).
	Name string `json:"name"`
}

// PhoneLoginInput holds the credentials for a phone+password login.
type PhoneLoginInput struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// UpdateProfileInput carries the fields a user may change on their own account.
type UpdateProfileInput struct {
	Locale         *string         `json:"locale,omitempty"`
	SavedPlayerIDs []SavedPlayerID `json:"savedPlayerIds,omitempty"`
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
	// TwoFactorRequired is set on an admin login that needs an SMS second factor.
	// When true, AccessToken/RefreshToken are empty and the caller must complete
	// the challenge via POST /auth/2fa/verify using PendingToken. PhoneHint is a
	// masked phone (e.g. "•••778") for display.
	TwoFactorRequired bool
	PendingToken      string
	PhoneHint         string
}

// AuthResponse is the public JSON body returned by register/login/refresh.
// RefreshToken is populated only for native clients (X-Client: mobile), which
// cannot rely on the httpOnly refresh cookie; browser clients receive it solely
// via that cookie and the field is omitted.
type AuthResponse struct {
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	User         *User  `json:"user,omitempty"`
	// Two-factor challenge (admin login): when TwoFactorRequired is true the tokens
	// above are absent and the client must post the SMS code to /auth/2fa/verify
	// with PendingToken. PhoneHint is a masked phone number for display.
	TwoFactorRequired bool   `json:"twoFactorRequired,omitempty"`
	PendingToken      string `json:"pendingToken,omitempty"`
	PhoneHint         string `json:"phoneHint,omitempty"`
}

// VerifyTwoFactorInput is the body of POST /auth/2fa/verify: the pending-challenge
// token returned by the login response plus the SMS code the admin received.
type VerifyTwoFactorInput struct {
	PendingToken string `json:"pendingToken"`
	Code         string `json:"code"`
}

// ResendTwoFactorInput is the body of POST /auth/2fa/resend: the pending-challenge
// token identifying the in-progress admin login.
type ResendTwoFactorInput struct {
	PendingToken string `json:"pendingToken"`
}

// RefreshInput is the optional JSON body accepted by POST /auth/refresh. Native
// clients present the refresh token here; browser clients omit it and the token
// is read from the httpOnly cookie instead.
type RefreshInput struct {
	RefreshToken string `json:"refreshToken"`
}
