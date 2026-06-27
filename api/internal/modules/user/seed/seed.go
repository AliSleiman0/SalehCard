package seed

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"

	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
)

// devPassword is the known password for the seeded accounts (development only).
const devPassword = "password123"

// seedUser is the shape of an account to seed.
type seedUser struct {
	Email   string
	Role    user.Role
	Status  user.Status
	Balance float64
	Loyalty int
	DaysAgo int // how long ago the account "joined" (for varied Joined dates)
}

// Seed inserts development accounts when they are missing. The two core accounts
// (customer + admin) are created when the collection is empty; a handful of demo
// accounts are additionally ensured so the admin users list/filters have signal.
// Safe to call repeatedly: each account is inserted only if its email is absent.
func Seed(ctx context.Context, db *mongo.Database) error {
	if err := user.EnsureIndexes(ctx, db); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(devPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hashed := string(hash)

	col := db.Collection("users")
	accounts := []seedUser{
		{Email: "customer@salehcard.local", Role: user.RoleCustomer, Status: user.StatusActive, Balance: 142.5, Loyalty: 320, DaysAgo: 180},
		{Email: "admin@salehcard.local", Role: user.RoleAdmin, Status: user.StatusActive, DaysAgo: 365},
		{Email: "gamehub.store@salehcard.local", Role: user.RoleReseller, Status: user.StatusActive, Balance: 4820, Loyalty: 5400, DaysAgo: 300},
		{Email: "sara.nasser@salehcard.local", Role: user.RoleCustomer, Status: user.StatusSuspended, Balance: 58.75, Loyalty: 90, DaysAgo: 45},
		{Email: "omar.haddad@salehcard.local", Role: user.RoleCustomer, Status: user.StatusActive, Balance: 12, Loyalty: 40, DaysAgo: 12},
		{Email: "topup.pro@salehcard.local", Role: user.RoleReseller, Status: user.StatusActive, Balance: 1290, Loyalty: 2100, DaysAgo: 90},
		{Email: "lina.khoury@salehcard.local", Role: user.RoleCustomer, Status: user.StatusActive, Balance: 0, Loyalty: 10, DaysAgo: 5},
		{Email: "blocked.reseller@salehcard.local", Role: user.RoleReseller, Status: user.StatusSuspended, Balance: 0, Loyalty: 0, DaysAgo: 60},
	}

	now := time.Now().UTC()
	for _, a := range accounts {
		exists, err := col.CountDocuments(ctx, bson.D{{Key: "email", Value: a.Email}})
		if err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		created := now.AddDate(0, 0, -a.DaysAgo)
		u := &user.User{
			ID:             bson.NewObjectID(),
			Email:          a.Email,
			PasswordHash:   &hashed,
			Role:           a.Role,
			Status:         a.Status,
			Locale:         "en",
			SavedPlayerIDs: []string{},
			WalletBalance:  a.Balance,
			LoyaltyPoints:  a.Loyalty,
			CreatedAt:      created,
			UpdatedAt:      created,
		}
		if _, err := col.InsertOne(ctx, u); err != nil {
			return err
		}
	}
	return nil
}
