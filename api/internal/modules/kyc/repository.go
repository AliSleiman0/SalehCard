package kyc

import (
	"context"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// Row is a submission enriched with the account's contact (email/phone) for the
// admin moderation queue, so the list renders the account behind each submission.
type Row struct {
	Submission  `bson:",inline"`
	UserContact string `bson:"userContact" json:"userContact"`
}

// Repository defines persistence operations for KYC submissions.
type Repository interface {
	FindByUserID(ctx context.Context, userID bson.ObjectID) (*Submission, error)
	Upsert(ctx context.Context, s *Submission) (*Submission, error)
	FindByID(ctx context.Context, id bson.ObjectID) (*Submission, error)
	List(ctx context.Context, f Filter, p pagination.Params) ([]Row, int64, error)
	UpdateStatus(ctx context.Context, id bson.ObjectID, status, reason, reviewedBy string) (*Submission, error)
	CountPending(ctx context.Context) (int64, error)
}

// Filter narrows an admin submission listing. Zero-valued fields are ignored.
type Filter struct {
	Status string
	Search string
}

// build assembles the MongoDB match document for f (status bucket + a name /
// document-number search).
func (f Filter) build() bson.D {
	and := bson.A{}
	switch f.Status {
	case StatusPending, StatusApproved, StatusRejected:
		and = append(and, bson.D{{Key: "status", Value: f.Status}})
	}
	if f.Search != "" {
		rx := bson.D{{Key: "$regex", Value: regexp.QuoteMeta(f.Search)}, {Key: "$options", Value: "i"}}
		and = append(and, bson.D{{Key: "$or", Value: bson.A{
			bson.D{{Key: "fullName", Value: rx}},
			bson.D{{Key: "documentNumber", Value: rx}},
		}}})
	}
	if len(and) == 0 {
		return bson.D{}
	}
	return bson.D{{Key: "$and", Value: and}}
}

// MongoRepository is a MongoDB-backed implementation of Repository. It also holds
// the users collection to join the account contact into the admin list.
type MongoRepository struct {
	collection *mongo.Collection
	users      *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository over the kyc_submissions
// collection, using users for the admin-list contact join.
func NewMongoRepository(submissions, users *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: submissions, users: users}
}

// EnsureIndexes creates the supporting indexes: a unique index on userId (one
// submission per user, enabling the upsert) and a status+createdAt index for the
// moderation queue.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("kyc_submissions").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: -1}}},
	})
	return err
}

// FindByUserID retrieves a user's submission, ErrNotFound when none exists.
func (r *MongoRepository) FindByUserID(ctx context.Context, userID bson.ObjectID) (*Submission, error) {
	var s Submission
	err := r.collection.FindOne(ctx, bson.D{{Key: "userId", Value: userID}}).Decode(&s)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// FindByID retrieves a submission by ObjectID, ErrNotFound when absent.
func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*Submission, error) {
	var s Submission
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&s)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// Upsert creates or replaces the user's submission, resetting it to pending and
// clearing any prior moderation outcome (re-submitting re-enters the queue).
func (r *MongoRepository) Upsert(ctx context.Context, s *Submission) (*Submission, error) {
	now := time.Now().UTC()
	var out Submission
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "userId", Value: s.UserID}},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "fullName", Value: s.FullName},
				{Key: "dateOfBirth", Value: s.DateOfBirth},
				{Key: "placeOfBirth", Value: s.PlaceOfBirth},
				{Key: "placeOfResidence", Value: s.PlaceOfResidence},
				{Key: "documentType", Value: s.DocumentType},
				{Key: "documentNumber", Value: s.DocumentNumber},
				{Key: "status", Value: StatusPending},
				{Key: "updatedAt", Value: now},
			}},
			{Key: "$unset", Value: bson.D{
				{Key: "rejectionReason", Value: ""},
				{Key: "reviewedBy", Value: ""},
				{Key: "reviewedAt", Value: ""},
			}},
			{Key: "$setOnInsert", Value: bson.D{
				{Key: "userId", Value: s.UserID},
				{Key: "createdAt", Value: now},
			}},
		},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns a paginated slice of submissions matching f, newest first, each
// joined to its account's contact (email/phone).
func (r *MongoRepository) List(ctx context.Context, f Filter, p pagination.Params) ([]Row, int64, error) {
	match := f.build()
	total, err := r.collection.CountDocuments(ctx, match)
	if err != nil {
		return nil, 0, err
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$sort", Value: bson.D{{Key: "createdAt", Value: -1}}}},
		{{Key: "$skip", Value: pagination.Skip(p)}},
		{{Key: "$limit", Value: int64(p.Limit)}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "userId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "userDoc"},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$userDoc"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
	}
	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var rows []struct {
		Submission `bson:",inline"`
		UserDoc    struct {
			Email string `bson:"email"`
			Phone string `bson:"phone"`
		} `bson:"userDoc"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, 0, err
	}

	out := make([]Row, len(rows))
	for i, row := range rows {
		out[i] = Row{Submission: row.Submission, UserContact: contactOf(row.UserDoc.Email, row.UserDoc.Phone)}
	}
	return out, total, nil
}

// UpdateStatus sets a submission's moderation status (stamping the reviewer +
// time, and the rejection reason on reject) and returns the updated document.
func (r *MongoRepository) UpdateStatus(ctx context.Context, id bson.ObjectID, status, reason, reviewedBy string) (*Submission, error) {
	now := time.Now().UTC()
	set := bson.D{
		{Key: "status", Value: status},
		{Key: "reviewedBy", Value: reviewedBy},
		{Key: "reviewedAt", Value: now},
		{Key: "updatedAt", Value: now},
	}
	update := bson.D{{Key: "$set", Value: set}}
	if status == StatusRejected {
		set = append(set, bson.E{Key: "rejectionReason", Value: reason})
		update = bson.D{{Key: "$set", Value: set}}
	} else {
		update = bson.D{
			{Key: "$set", Value: set},
			{Key: "$unset", Value: bson.D{{Key: "rejectionReason", Value: ""}}},
		}
	}
	var s Submission
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&s)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// CountPending counts submissions awaiting moderation (dashboard metric).
func (r *MongoRepository) CountPending(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.D{{Key: "status", Value: StatusPending}})
}

// contactOf derives a display contact from an account's email or phone.
func contactOf(email, phone string) string {
	if email != "" {
		if at := strings.IndexByte(email, '@'); at > 0 {
			return email[:at]
		}
		return email
	}
	if phone != "" {
		return phone
	}
	return "Unknown"
}
