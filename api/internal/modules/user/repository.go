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

	// TouchLastSeen stamps the account's last-seen time (best-effort activity
	// heartbeat, written on every auth). CountActiveSince counts accounts seen
	// at or after `since` — the "active users" dashboard metric.
	TouchLastSeen(ctx context.Context, id bson.ObjectID, t time.Time) error
	CountActiveSince(ctx context.Context, since time.Time) (int64, error)

	// Admin operations.
	ListAll(ctx context.Context, f UserFilter, p pagination.Params) ([]*User, int64, error)
	// UpdateRole sets a user's role and RBAC role assignment together. A nil
	// adminRoleID (or a non-admin role) clears the assignment — for an admin
	// that means built-in Super Admin.
	UpdateRole(ctx context.Context, id bson.ObjectID, role Role, adminRoleID *bson.ObjectID) (*User, error)
	UpdateStatus(ctx context.Context, id bson.ObjectID, status Status) (*User, error)
	// BulkUpdateStatus sets status on every user in ids, returning the modified count.
	BulkUpdateStatus(ctx context.Context, ids []bson.ObjectID, status Status) (int64, error)
	// SoftDelete anonymizes an account in place (status=deleted, PII unset).
	SoftDelete(ctx context.Context, id bson.ObjectID) (*User, error)
	UpdateResellerTier(ctx context.Context, id bson.ObjectID, tier string) (*User, error)
	// CountActiveSuperAdmins counts non-suspended super admins — admins with no
	// custom RBAC role (the last-super-admin demotion/suspension guard).
	CountActiveSuperAdmins(ctx context.Context) (int64, error)
}

// UserFilter narrows an admin user listing. Zero-valued fields are ignored.
type UserFilter struct {
	Role         Role
	Status       Status
	ResellerTier string // exact tier name (reseller listing)
	Search       string // matches an ObjectID hex, or email/phone (case-insensitive)
}

// build assembles the MongoDB filter document for f.
func (f UserFilter) build() bson.D {
	filter := bson.D{}
	if f.Role != "" {
		filter = append(filter, bson.E{Key: "role", Value: f.Role})
	}
	switch f.Status {
	case StatusActive:
		// "active" must also match legacy accounts that predate the status field
		// (no status stored) — treat a missing status as active.
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "status", Value: StatusActive}},
			bson.D{{Key: "status", Value: bson.D{{Key: "$exists", Value: false}}}},
		}})
	case "":
		// Default listing hides soft-deleted accounts (a missing status is a
		// legacy-active account, which $ne keeps included).
		filter = append(filter, bson.E{Key: "status", Value: bson.D{{Key: "$ne", Value: StatusDeleted}}})
	default:
		// suspended / deleted: exact match.
		filter = append(filter, bson.E{Key: "status", Value: f.Status})
	}
	if f.ResellerTier != "" {
		filter = append(filter, bson.E{Key: "resellerTier", Value: f.ResellerTier})
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
		{Keys: bson.D{{Key: "lastSeen", Value: -1}}},
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
		user.SavedPlayerIDs = []SavedPlayerID{}
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

// TouchLastSeen stamps the account's lastSeen time. It is best-effort and does
// not bump updatedAt (which tracks profile edits, not activity). A missing user
// is silently ignored.
func (r *MongoRepository) TouchLastSeen(ctx context.Context, id bson.ObjectID, t time.Time) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "lastSeen", Value: t.UTC()}}}},
	)
	return err
}

// CountActiveSince counts accounts whose lastSeen is at or after `since`.
func (r *MongoRepository) CountActiveSince(ctx context.Context, since time.Time) (int64, error) {
	return r.collection.CountDocuments(ctx,
		bson.D{{Key: "lastSeen", Value: bson.D{{Key: "$gte", Value: since.UTC()}}}})
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

// UpdateRole atomically sets a user's role and RBAC role assignment and returns
// the updated document, returning ErrNotFound when no user matches. The
// assignment is cleared (unset) when adminRoleID is nil or the role is not
// admin, so a demoted account never keeps a stale role reference.
func (r *MongoRepository) UpdateRole(ctx context.Context, id bson.ObjectID, role Role, adminRoleID *bson.ObjectID) (*User, error) {
	set := bson.D{
		{Key: "role", Value: role},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}
	update := bson.D{}
	if role == RoleAdmin && adminRoleID != nil {
		set = append(set, bson.E{Key: "adminRoleId", Value: *adminRoleID})
	} else {
		update = append(update, bson.E{Key: "$unset", Value: bson.D{{Key: "adminRoleId", Value: ""}}})
	}
	update = append(update, bson.E{Key: "$set", Value: set})

	var u User
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		update,
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

// UpdateStatus atomically sets a user's status and returns the updated document,
// returning ErrNotFound when no user matches.
func (r *MongoRepository) UpdateStatus(ctx context.Context, id bson.ObjectID, status Status) (*User, error) {
	return r.setFields(ctx, id, bson.D{{Key: "status", Value: status}})
}

// BulkUpdateStatus sets the status on every user whose id is in ids (a single
// UpdateMany) and returns the number of documents modified. An empty id list is
// a no-op returning 0.
func (r *MongoRepository) BulkUpdateStatus(ctx context.Context, ids []bson.ObjectID, status Status) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res, err := r.collection.UpdateMany(ctx,
		bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: status},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// SoftDelete anonymizes a user in place: it marks the account deleted, stamps
// deletedAt, and unsets the email/phone/credential identifiers (freeing the
// sparse-unique indexes so the person can re-register) while leaving orders and
// ledger rows intact. Returns ErrNotFound when no user matches.
func (r *MongoRepository) SoftDelete(ctx context.Context, id bson.ObjectID) (*User, error) {
	now := time.Now().UTC()
	var u User
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "status", Value: StatusDeleted},
				{Key: "deletedAt", Value: now},
				{Key: "updatedAt", Value: now},
			}},
			{Key: "$unset", Value: bson.D{
				{Key: "email", Value: ""},
				{Key: "phone", Value: ""},
				{Key: "passwordHash", Value: ""},
				{Key: "googleId", Value: ""},
				// Sever any RBAC role assignment: a dead, anonymized account must
				// not keep an adminRoleId, or role.CountAssigned would count it and
				// block deletion of an otherwise-unused role forever.
				{Key: "adminRoleId", Value: ""},
			}},
		},
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

// CountActiveSuperAdmins counts usable super admins (admins without a custom
// RBAC role — a nil adminRoleId matches both missing and explicit null). It
// excludes both suspended AND soft-deleted accounts: a deleted admin keeps
// role=admin but can never sign in, so counting it would let the last usable
// super admin be removed and permanently lock out role/admin management. The
// $nin (rather than == active) keeps legacy documents with no status field
// counted as active.
func (r *MongoRepository) CountActiveSuperAdmins(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.D{
		{Key: "role", Value: RoleAdmin},
		{Key: "adminRoleId", Value: nil},
		{Key: "status", Value: bson.D{{Key: "$nin", Value: bson.A{StatusSuspended, StatusDeleted}}}},
	})
}

// UpdateResellerTier atomically sets a user's reseller tier name and returns the
// updated document, returning ErrNotFound when no user matches.
func (r *MongoRepository) UpdateResellerTier(ctx context.Context, id bson.ObjectID, tier string) (*User, error) {
	return r.setFields(ctx, id, bson.D{{Key: "resellerTier", Value: tier}})
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
