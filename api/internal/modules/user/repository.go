package user

import (
	"context"
	"regexp"
	"strings"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository defines persistence operations for the User entity.
type Repository interface {
	FindByID(ctx context.Context, id bson.ObjectID) (*User, error)
	FindByIDs(ctx context.Context, ids []bson.ObjectID) ([]*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByPhone(ctx context.Context, phone string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	SetPassword(ctx context.Context, id bson.ObjectID, passwordHash string) error
	Delete(ctx context.Context, id bson.ObjectID) error

	// Admin operations.
	ListAll(ctx context.Context, f UserFilter, p pagination.Params) ([]*User, int64, error)
	UpdateRole(ctx context.Context, id bson.ObjectID, role Role) (*User, error)
	UpdateStatus(ctx context.Context, id bson.ObjectID, status Status) (*User, error)
}

// UserFilter narrows an admin user listing. Zero-valued fields are ignored.
type UserFilter struct {
	Role   Role
	Status Status
	Search string // matches an ObjectID hex, or email/phone (case-insensitive)
}

// build assembles the MongoDB filter document for f.
func (f UserFilter) build() bson.D {
	filter := bson.D{}
	if f.Role != "" {
		filter = append(filter, bson.E{Key: "role", Value: f.Role})
	}
	if f.Status != "" {
		// "active" must also match legacy accounts that predate the status field
		// (no status stored) — treat a missing status as active.
		if f.Status == StatusActive {
			filter = append(filter, bson.E{Key: "$or", Value: bson.A{
				bson.D{{Key: "status", Value: StatusActive}},
				bson.D{{Key: "status", Value: bson.D{{Key: "$exists", Value: false}}}},
			}})
		} else {
			filter = append(filter, bson.E{Key: "status", Value: f.Status})
		}
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		if id, err := bson.ObjectIDFromHex(s); err == nil {
			filter = append(filter, bson.E{Key: "_id", Value: id})
		} else {
			pattern := regexp.QuoteMeta(s)
			rx := bson.D{{Key: "$regex", Value: pattern}, {Key: "$options", Value: "i"}}
			filter = append(filter, bson.E{Key: "$or", Value: bson.A{
				bson.D{{Key: "email", Value: rx}},
				bson.D{{Key: "phone", Value: rx}},
			}})
		}
	}
	return filter
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

// FindByIDs retrieves users for the given ObjectIDs in a single query. Missing
// IDs are simply omitted from the result; the order is not guaranteed. An empty
// input returns an empty slice without touching the database.
func (r *MongoRepository) FindByIDs(ctx context.Context, ids []bson.ObjectID) ([]*User, error) {
	if len(ids) == 0 {
		return []*User{}, nil
	}
	cur, err := r.collection.Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*User{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FindByEmailLike returns users whose email contains q (case-insensitive). Used
// by admin order search. An empty/whitespace term returns an empty slice.
func (r *MongoRepository) FindByEmailLike(ctx context.Context, q string) ([]*User, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []*User{}, nil
	}
	pattern := regexp.QuoteMeta(strings.ToLower(q))
	cur, err := r.collection.Find(ctx, bson.D{{Key: "email", Value: bson.D{
		{Key: "$regex", Value: pattern},
		{Key: "$options", Value: "i"},
	}}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*User{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
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
	if user.Status == "" {
		user.Status = StatusActive
	}
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

// ListAll returns a paginated, newest-first page of users matching f, plus the
// total count of matches (for pagination meta).
func (r *MongoRepository) ListAll(ctx context.Context, f UserFilter, p pagination.Params) ([]*User, int64, error) {
	filter := f.build()
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*User{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// UpdateRole atomically sets a user's role and returns the updated document,
// returning ErrNotFound when no user matches.
func (r *MongoRepository) UpdateRole(ctx context.Context, id bson.ObjectID, role Role) (*User, error) {
	return r.setFields(ctx, id, bson.D{{Key: "role", Value: role}})
}

// UpdateStatus atomically sets a user's status and returns the updated document,
// returning ErrNotFound when no user matches.
func (r *MongoRepository) UpdateStatus(ctx context.Context, id bson.ObjectID, status Status) (*User, error) {
	return r.setFields(ctx, id, bson.D{{Key: "status", Value: status}})
}

// setFields applies a single-document $set (plus updatedAt) and returns the
// post-update user.
func (r *MongoRepository) setFields(ctx context.Context, id bson.ObjectID, fields bson.D) (*User, error) {
	fields = append(fields, bson.E{Key: "updatedAt", Value: time.Now().UTC()})
	var u User
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: fields}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
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
