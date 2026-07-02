package notification

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// notificationTTL caps inbox growth: rows expire server-side after ~6 months.
const notificationTTL = 180 * 24 * time.Hour

// Repository defines persistence for notifications and device tokens.
type Repository interface {
	Insert(ctx context.Context, n *Notification) error
	ListByUser(ctx context.Context, userID bson.ObjectID, p pagination.Params) ([]*Notification, int64, error)
	CountUnread(ctx context.Context, userID bson.ObjectID) (int64, error)
	MarkAllRead(ctx context.Context, userID bson.ObjectID) (int64, error)
	UpsertToken(ctx context.Context, userID bson.ObjectID, token, platform string) error
	DeleteToken(ctx context.Context, userID bson.ObjectID, token string) error
	TokensForUser(ctx context.Context, userID bson.ObjectID) ([]string, error)
	DeleteTokenValue(ctx context.Context, token string) error
}

// MongoRepository persists notifications and device_tokens.
type MongoRepository struct {
	notifications *mongo.Collection
	tokens        *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository over the two collections.
func NewMongoRepository(notifications, tokens *mongo.Collection) *MongoRepository {
	return &MongoRepository{notifications: notifications, tokens: tokens}
}

// EnsureIndexes creates the inbox indexes (newest-first listing, unread count,
// TTL expiry) and the device-token indexes (globally-unique token, per-user
// fan-out lookup).
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("notifications").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "readAt", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(int32(notificationTTL.Seconds()))},
	})
	if err != nil {
		return err
	}
	_, err = db.Collection("device_tokens").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "token", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}}},
	})
	return err
}

// Insert stores one notification row.
func (r *MongoRepository) Insert(ctx context.Context, n *Notification) error {
	if n.ID.IsZero() {
		n.ID = bson.NewObjectID()
	}
	_, err := r.notifications.InsertOne(ctx, n)
	return err
}

// ListByUser returns a paginated slice of the user's notifications, newest first.
func (r *MongoRepository) ListByUser(ctx context.Context, userID bson.ObjectID, p pagination.Params) ([]*Notification, int64, error) {
	filter := bson.D{{Key: "userId", Value: userID}}
	total, err := r.notifications.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.notifications.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*Notification{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// CountUnread counts the user's unread notifications (readAt absent or null —
// the nil filter matches both).
func (r *MongoRepository) CountUnread(ctx context.Context, userID bson.ObjectID) (int64, error) {
	return r.notifications.CountDocuments(ctx, bson.D{
		{Key: "userId", Value: userID},
		{Key: "readAt", Value: nil},
	})
}

// MarkAllRead stamps every unread notification of the user as read, returning
// the number of rows updated. One atomic UpdateMany — no select-then-update.
func (r *MongoRepository) MarkAllRead(ctx context.Context, userID bson.ObjectID) (int64, error) {
	res, err := r.notifications.UpdateMany(ctx,
		bson.D{{Key: "userId", Value: userID}, {Key: "readAt", Value: nil}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "readAt", Value: time.Now().UTC()}}}},
	)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// UpsertToken registers (or reassigns) a device token to userID. Keyed on the
// globally-unique token so a shared device always pushes to the latest login.
func (r *MongoRepository) UpsertToken(ctx context.Context, userID bson.ObjectID, token, platform string) error {
	_, err := r.tokens.UpdateOne(ctx,
		bson.D{{Key: "token", Value: token}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "userId", Value: userID},
			{Key: "platform", Value: platform},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// DeleteToken removes the token registration, scoped to the owning user so one
// account cannot unregister another's device.
func (r *MongoRepository) DeleteToken(ctx context.Context, userID bson.ObjectID, token string) error {
	_, err := r.tokens.DeleteOne(ctx, bson.D{
		{Key: "token", Value: token},
		{Key: "userId", Value: userID},
	})
	return err
}

// TokensForUser returns every device token registered to userID.
func (r *MongoRepository) TokensForUser(ctx context.Context, userID bson.ObjectID) ([]string, error) {
	cur, err := r.tokens.Find(ctx, bson.D{{Key: "userId", Value: userID}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []DeviceToken
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = row.Token
	}
	return out, nil
}

// DeleteTokenValue removes a token regardless of owner — the prune path when
// the push provider reports it unregistered.
func (r *MongoRepository) DeleteTokenValue(ctx context.Context, token string) error {
	_, err := r.tokens.DeleteOne(ctx, bson.D{{Key: "token", Value: token}})
	return err
}
