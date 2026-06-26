package user

import (
	"context"
	"strings"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository defines persistence operations for the User entity.
type Repository interface {
	FindByID(ctx context.Context, id bson.ObjectID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByPhone(ctx context.Context, phone string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	SetPassword(ctx context.Context, id bson.ObjectID, passwordHash string) error
	Delete(ctx context.Context, id bson.ObjectID) error
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository over the "users" collection.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{collection: db.Collection("users")}
}

// EnsureIndexes creates the indexes the user module relies on: sparse-unique
// indexes on email and phone (so an account may carry just one of them) and a
// secondary index on role. A pre-existing non-sparse email index is dropped and
// recreated so phone-only accounts (no email) don't collide on a null email.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	users := db.Collection("users")
	dropIndexIfNotSparse(ctx, users, "email_1")
	_, err := users.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys:    bson.D{{Key: "phone", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{Keys: bson.D{{Key: "role", Value: 1}}},
	})
	if err != nil {
		return err
	}
	if err := EnsureRefreshIndexes(ctx, db); err != nil {
		return err
	}
	return EnsureOTPIndexes(ctx, db)
}

// dropIndexIfNotSparse drops the named index when it exists without the sparse
// flag, so it can be recreated sparse. Best-effort: errors are ignored (the
// subsequent CreateMany surfaces any real problem).
func dropIndexIfNotSparse(ctx context.Context, coll *mongo.Collection, name string) {
	cur, err := coll.Indexes().List(ctx)
	if err != nil {
		return
	}
	var idxs []bson.M
	if err := cur.All(ctx, &idxs); err != nil {
		return
	}
	for _, idx := range idxs {
		if idx["name"] != name {
			continue
		}
		if sparse, _ := idx["sparse"].(bool); !sparse {
			_ = coll.Indexes().DropOne(ctx, name)
		}
		return
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// FindByID retrieves a single user by ObjectID, returning ErrNotFound when absent.
func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*User, error) {
	var u User
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// FindByEmail retrieves a user by (normalized) email, returning ErrNotFound when absent.
func (r *MongoRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.collection.FindOne(ctx, bson.D{{Key: "email", Value: normalizeEmail(email)}}).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// FindByPhone retrieves a user by phone (E.164), returning ErrNotFound when absent.
func (r *MongoRepository) FindByPhone(ctx context.Context, phone string) (*User, error) {
	var u User
	err := r.collection.FindOne(ctx, bson.D{{Key: "phone", Value: phone}}).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// Create inserts a new user, stamping CreatedAt/UpdatedAt and normalizing email.
// A duplicate email or phone (unique index violation) is surfaced as ErrConflict.
func (r *MongoRepository) Create(ctx context.Context, user *User) error {
	now := time.Now().UTC()
	if user.ID.IsZero() {
		user.ID = bson.NewObjectID()
	}
	user.Email = normalizeEmail(user.Email)
	user.CreatedAt = now
	user.UpdatedAt = now
	if user.SavedPlayerIDs == nil {
		user.SavedPlayerIDs = []string{}
	}

	if _, err := r.collection.InsertOne(ctx, user); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return apperrors.ErrConflict
		}
		return err
	}
	return nil
}

// Update writes mutable profile fields (locale, saved IDs) and refreshes UpdatedAt.
func (r *MongoRepository) Update(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now().UTC()
	res, err := r.collection.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: user.ID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "locale", Value: user.Locale},
			{Key: "savedPlayerIds", Value: user.SavedPlayerIDs},
			{Key: "updatedAt", Value: user.UpdatedAt},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// SetPassword sets (or replaces) the bcrypt password hash for a user.
func (r *MongoRepository) SetPassword(ctx context.Context, id bson.ObjectID, passwordHash string) error {
	res, err := r.collection.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "passwordHash", Value: passwordHash},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// Delete removes a user by ObjectID.
func (r *MongoRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}
