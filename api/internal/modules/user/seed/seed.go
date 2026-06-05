package seed

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"

	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
)

// devPassword is the known password for both seeded accounts (development only).
const devPassword = "password123"

// seedUser is the shape of an account to seed.
type seedUser struct {
	Email string
	Role  user.Role
}

// Seed inserts development accounts (one customer, one admin) when the users
// collection is empty. Safe to call repeatedly; a no-op once data exists.
func Seed(ctx context.Context, db *mongo.Database) error {
	if err := user.EnsureIndexes(ctx, db); err != nil {
		return err
	}

	col := db.Collection("users")
	count, err := col.CountDocuments(ctx, bson.D{})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(devPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	repo := user.NewMongoRepository(db)
	accounts := []seedUser{
		{Email: "customer@salehcard.local", Role: user.RoleCustomer},
		{Email: "admin@salehcard.local", Role: user.RoleAdmin},
	}
	for _, a := range accounts {
		hashed := string(hash)
		u := &user.User{
			Email:          a.Email,
			PasswordHash:   &hashed,
			Role:           a.Role,
			Locale:         "en",
			SavedPlayerIDs: []string{},
		}
		if err := repo.Create(ctx, u); err != nil {
			return err
		}
	}
	return nil
}
