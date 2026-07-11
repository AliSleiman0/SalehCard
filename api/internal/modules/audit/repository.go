package audit

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// Filter narrows an audit-log listing. Zero-valued fields are ignored.
type Filter struct {
	Action     string
	TargetType string
	TargetID   string
	ActorID    string
	From       *time.Time
	To         *time.Time
}

// build assembles the MongoDB filter document for f.
func (f Filter) build() bson.D {
	and := bson.A{}
	if f.Action != "" {
		and = append(and, bson.D{{Key: "action", Value: f.Action}})
	}
	if f.TargetType != "" {
		and = append(and, bson.D{{Key: "targetType", Value: f.TargetType}})
	}
	if f.TargetID != "" {
		and = append(and, bson.D{{Key: "targetId", Value: f.TargetID}})
	}
	if f.ActorID != "" {
		and = append(and, bson.D{{Key: "actorId", Value: f.ActorID}})
	}
	if f.From != nil {
		and = append(and, bson.D{{Key: "at", Value: bson.D{{Key: "$gte", Value: *f.From}}}})
	}
	if f.To != nil {
		and = append(and, bson.D{{Key: "at", Value: bson.D{{Key: "$lte", Value: *f.To}}}})
	}
	if len(and) == 0 {
		return bson.D{}
	}
	return bson.D{{Key: "$and", Value: and}}
}

// MongoRepository persists audit entries in the admin_audit_log collection.
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository over col.
func NewMongoRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: col}
}

// EnsureIndexes creates the audit-log indexes: newest-first listing, per-actor
// history, and per-target history.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("admin_audit_log").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "at", Value: -1}}},
		{Keys: bson.D{{Key: "actorId", Value: 1}, {Key: "at", Value: -1}}},
		{Keys: bson.D{{Key: "targetType", Value: 1}, {Key: "targetId", Value: 1}, {Key: "at", Value: -1}}},
	})
	return err
}

// Insert stores one audit entry.
func (r *MongoRepository) Insert(ctx context.Context, e *Entry) error {
	if e.ID.IsZero() {
		e.ID = bson.NewObjectID()
	}
	_, err := r.collection.InsertOne(ctx, e)
	return err
}

// List returns a paginated slice of audit entries matching f, newest first.
func (r *MongoRepository) List(ctx context.Context, f Filter, p pagination.Params) ([]*Entry, int64, error) {
	filter := f.build()
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "at", Value: -1}})
	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*Entry{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// recorder is the Recorder implementation backed by MongoRepository.
type recorder struct {
	repo *MongoRepository
}

// NewRecorder returns the Recorder the server wires into each admin module.
func NewRecorder(db *mongo.Database) Recorder {
	return &recorder{repo: NewMongoRepository(db.Collection("admin_audit_log"))}
}

// Record fills the actor (from the request JWT claims) and timestamp, then
// persists the entry. Failures are logged, never propagated — see Recorder.
// A pre-set actor wins over the claims: customer self-deletion uses this so
// the permanent audit row never stores the email/phone the same request just
// erased from the user document.
func (rec *recorder) Record(ctx context.Context, e Entry) {
	if claims, ok := auth.ClaimsFromContext(ctx); ok && e.ActorID == "" && e.ActorEmail == "" {
		e.ActorID = claims.UserID
		// Keep real emails intact; fall back to phone/user-id only when empty so a
		// phone-only admin never records a blank actor.
		e.ActorEmail = claims.Email
		if e.ActorEmail == "" {
			e.ActorEmail = auth.ActorLabel(claims)
		}
	}
	e.At = time.Now().UTC()
	if err := rec.repo.Insert(ctx, &e); err != nil {
		slog.Error("audit: failed to record admin action",
			"action", e.Action, "targetType", e.TargetType, "targetId", e.TargetID,
			"actor", e.ActorEmail, "error", err)
	}
}
