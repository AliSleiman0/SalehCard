package chatbot

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// chatLogTTL is how long AI conversations are kept (for quality review and
// knowledge gap finding). Customers may type personal details, so they
// expire rather than accumulate, and they are never written to the app log.
const chatLogTTL = 30 * 24 * time.Hour

// Outcomes recorded per AI question.
const (
	outcomeAnswered     = "answered"   // grounded answer, clarifying question or small talk
	outcomeUnanswered   = "unanswered" // about the app but not covered (handoff) or unusable output — the gaps
	outcomeOffTopic     = "off_topic"  // not about the app (politely declined)
	outcomeFiltered     = "filtered"   // provider content-safety block, or the iOS output guard
	outcomeError        = "error"      // provider/transport failure
	logFilterFlagged    = "flagged"
	logFilterUnanswered = "unanswered"
)

// ChatLog is one AI question and what the assistant did with it
// (collection chat_logs, TTL-expired after chatLogTTL).
type ChatLog struct {
	ID           bson.ObjectID `bson:"_id"                   json:"id"`
	UserID       string        `bson:"userId"                json:"userId"`
	Lang         string        `bson:"lang"                  json:"lang"`
	Question     string        `bson:"question"              json:"question"`
	Answer       string        `bson:"answer,omitempty"      json:"answer,omitempty"`
	Outcome      string        `bson:"outcome"               json:"outcome"`
	Intent       string        `bson:"intent,omitempty"      json:"intent,omitempty"`  // the model's classification
	Sources      []string      `bson:"sources,omitempty"     json:"sources,omitempty"` // start of each cited knowledge paragraph
	InputTokens  int           `bson:"inputTokens"           json:"inputTokens"`
	CachedTokens int           `bson:"cachedTokens"          json:"cachedTokens"`
	OutputTokens int           `bson:"outputTokens"          json:"outputTokens"`
	LatencyMs    int64         `bson:"latencyMs"             json:"latencyMs"`
	Flagged      bool          `bson:"flagged"               json:"flagged"`
	FlaggedAt    *time.Time    `bson:"flaggedAt,omitempty"   json:"flaggedAt,omitempty"`
	CreatedAt    time.Time     `bson:"createdAt"             json:"createdAt"`
}

// LogRepository persists AI conversation logs.
type LogRepository interface {
	Insert(ctx context.Context, l *ChatLog) error
	// Flag marks the user's own log as reported; errNotFound when it is not
	// theirs (or gone).
	Flag(ctx context.Context, id bson.ObjectID, userID string, at time.Time) error
	// List returns a page of logs, newest first. filter is "", "unanswered" or
	// "flagged".
	List(ctx context.Context, filter string, skip, limit int) ([]ChatLog, bool, error)
}

// MongoLogRepository is the MongoDB-backed LogRepository.
type MongoLogRepository struct{ col *mongo.Collection }

// NewMongoLogRepository builds the repository over chat_logs.
func NewMongoLogRepository(db *mongo.Database) *MongoLogRepository {
	return &MongoLogRepository{col: db.Collection("chat_logs")}
}

// EnsureLogIndexes creates the TTL index and the admin-list index.
func EnsureLogIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("chat_logs").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "createdAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(int32(chatLogTTL.Seconds()))},
		{Keys: bson.D{{Key: "outcome", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "flagged", Value: 1}, {Key: "createdAt", Value: -1}}},
	})
	return err
}

// Insert implements LogRepository.
func (r *MongoLogRepository) Insert(ctx context.Context, l *ChatLog) error {
	_, err := r.col.InsertOne(ctx, l)
	return err
}

// Flag implements LogRepository (owner-scoped, document-atomic).
func (r *MongoLogRepository) Flag(ctx context.Context, id bson.ObjectID, userID string, at time.Time) error {
	res, err := r.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "userId", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "flagged", Value: true}, {Key: "flaggedAt", Value: at}}}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errNotFound
	}
	return nil
}

// List implements LogRepository. It fetches one extra row to report hasMore.
func (r *MongoLogRepository) List(ctx context.Context, filter string, skip, limit int) ([]ChatLog, bool, error) {
	f := bson.D{}
	switch filter {
	case logFilterFlagged:
		f = bson.D{{Key: "flagged", Value: true}}
	case logFilterUnanswered:
		f = bson.D{{Key: "outcome", Value: bson.D{{Key: "$ne", Value: outcomeAnswered}}}}
	case "":
	default:
		return nil, false, errors.New("unknown filter")
	}
	cur, err := r.col.Find(ctx, f, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64(skip)).SetLimit(int64(limit+1)))
	if err != nil {
		return nil, false, err
	}
	defer cur.Close(ctx)
	out := []ChatLog{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, false, err
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, nil
}
