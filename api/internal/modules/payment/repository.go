package payment

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// derivationCounterID is the counters document that hands out HD derivation
// indexes ($inc — document-atomic, so two concurrent intents can't share an
// address even without transactions).
const derivationCounterID = "usdt_trc20_deriv_index"

// IntentFilter narrows the admin listing.
type IntentFilter struct {
	Status  string
	Purpose string
}

// Store defines persistence for payment intents. The service depends on this
// interface, not the Mongo implementation.
type Store interface {
	Insert(ctx context.Context, in *Intent) error
	GetByID(ctx context.Context, id bson.ObjectID) (*Intent, error)
	GetByIdempotencyKey(ctx context.Context, userID bson.ObjectID, key string) (*Intent, error)
	GetActiveByOrder(ctx context.Context, orderID bson.ObjectID) (*Intent, error)
	CountOpenForUser(ctx context.Context, userID bson.ObjectID) (int64, error)

	// ListWatchable returns the intents the watcher should scan the chain
	// for: pending ones, plus expired ones with no payment seen yet whose
	// expiry is within the late-payment grace window. Oldest first, capped.
	ListWatchable(ctx context.Context, now time.Time, grace time.Duration, limit int) ([]*Intent, error)
	// ListSettleRetries returns confirming intents — a payment was claimed
	// but settlement didn't complete (crash / dependency error); the watcher
	// retries them each tick.
	ListSettleRetries(ctx context.Context, limit int) ([]*Intent, error)
	// ListExpiryCandidates returns pending intents past their expiry.
	ListExpiryCandidates(ctx context.Context, now time.Time, limit int) ([]*Intent, error)

	// ClaimPaymentSeen atomically moves a pending/expired intent to
	// confirming, recording the observed transfer. The (network, txHash)
	// unique index plus the status precondition make this the double-credit
	// lock: ErrConflict when another claim won or the tx is already recorded.
	ClaimPaymentSeen(ctx context.Context, id bson.ObjectID, txHash, from string, receivedMicros int64) (*Intent, error)
	// MarkConfirmed finalizes a confirming intent with its settlement outcome.
	MarkConfirmed(ctx context.Context, id bson.ObjectID, settlement string) error
	IncSettleAttempts(ctx context.Context, id bson.ObjectID) error
	// ExpireOne atomically moves a pending intent past its expiry to expired;
	// ErrConflict when the intent moved on concurrently (e.g. a same-tick claim).
	ExpireOne(ctx context.Context, id bson.ObjectID, now time.Time) (*Intent, error)

	// NextDerivationIndex hands out the next HD address index.
	NextDerivationIndex(ctx context.Context) (uint32, error)

	ListAll(ctx context.Context, f IntentFilter, p pagination.Params) ([]*Intent, int64, error)
}

// MongoStore is the MongoDB-backed Store over payment_intents (+ counters).
type MongoStore struct {
	col      *mongo.Collection
	counters *mongo.Collection
}

// NewMongoStore constructs a MongoStore.
func NewMongoStore(db *mongo.Database) *MongoStore {
	return &MongoStore{
		col:      db.Collection("payment_intents"),
		counters: db.Collection("counters"),
	}
}

// EnsureIndexes creates the payment-intent indexes. The partial-unique
// (network, txHash) index is the double-credit guard: one observed transfer
// can be recorded on exactly one intent, ever.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("payment_intents").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "network", Value: 1}, {Key: "txHash", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "txHash", Value: bson.D{{Key: "$exists", Value: true}}}}),
		},
		// Every intent owns its derived address; a duplicate would mean a
		// derivation-counter bug, so fail loudly at insert.
		{Keys: bson.D{{Key: "address", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "expiresAt", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
		{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "idempotencyKey", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "idempotencyKey", Value: bson.D{{Key: "$exists", Value: true}}}}),
		},
		{
			Keys:    bson.D{{Key: "orderId", Value: 1}},
			Options: options.Index().SetPartialFilterExpression(bson.D{{Key: "orderId", Value: bson.D{{Key: "$exists", Value: true}}}}),
		},
	})
	return err
}

// Insert stores a new intent, stamping ID and timestamps. A duplicate
// idempotency key surfaces as ErrConflict so the caller can return the winner.
func (s *MongoStore) Insert(ctx context.Context, in *Intent) error {
	if in.ID.IsZero() {
		in.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	if in.CreatedAt.IsZero() {
		in.CreatedAt = now
	}
	in.UpdatedAt = now
	if _, err := s.col.InsertOne(ctx, in); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return apperrors.ErrConflict
		}
		return err
	}
	return nil
}

// GetByID retrieves one intent, ErrNotFound when absent.
func (s *MongoStore) GetByID(ctx context.Context, id bson.ObjectID) (*Intent, error) {
	var in Intent
	err := s.col.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&in)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &in, nil
}

// GetByIdempotencyKey finds the user's intent created with key, ErrNotFound
// when absent.
func (s *MongoStore) GetByIdempotencyKey(ctx context.Context, userID bson.ObjectID, key string) (*Intent, error) {
	var in Intent
	err := s.col.FindOne(ctx, bson.D{
		{Key: "userId", Value: userID},
		{Key: "idempotencyKey", Value: key},
	}).Decode(&in)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &in, nil
}

// GetActiveByOrder returns the order's live (pending/confirming) intent,
// ErrNotFound when none.
func (s *MongoStore) GetActiveByOrder(ctx context.Context, orderID bson.ObjectID) (*Intent, error) {
	var in Intent
	err := s.col.FindOne(ctx, bson.D{
		{Key: "orderId", Value: orderID},
		{Key: "status", Value: bson.D{{Key: "$in", Value: []IntentStatus{StatusPending, StatusConfirming}}}},
	}, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&in)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &in, nil
}

// CountOpenForUser counts the user's pending intents (the spam guard, like
// the top-up queue's CountPendingForUser).
func (s *MongoStore) CountOpenForUser(ctx context.Context, userID bson.ObjectID) (int64, error) {
	return s.col.CountDocuments(ctx, bson.D{
		{Key: "userId", Value: userID},
		{Key: "status", Value: StatusPending},
	})
}

// ListWatchable implements the watcher's scan set: pending intents plus
// expired-but-unpaid ones still within grace.
func (s *MongoStore) ListWatchable(ctx context.Context, now time.Time, grace time.Duration, limit int) ([]*Intent, error) {
	filter := bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "status", Value: StatusPending}},
		bson.D{
			{Key: "status", Value: StatusExpired},
			{Key: "txHash", Value: bson.D{{Key: "$exists", Value: false}}},
			{Key: "expiresAt", Value: bson.D{{Key: "$gt", Value: now.Add(-grace)}}},
		},
	}}}
	return s.list(ctx, filter, limit, 1)
}

// ListSettleRetries returns confirming intents oldest-first.
func (s *MongoStore) ListSettleRetries(ctx context.Context, limit int) ([]*Intent, error) {
	return s.list(ctx, bson.D{{Key: "status", Value: StatusConfirming}}, limit, 1)
}

// ListExpiryCandidates returns pending intents whose expiry has passed.
func (s *MongoStore) ListExpiryCandidates(ctx context.Context, now time.Time, limit int) ([]*Intent, error) {
	return s.list(ctx, bson.D{
		{Key: "status", Value: StatusPending},
		{Key: "expiresAt", Value: bson.D{{Key: "$lt", Value: now}}},
	}, limit, 1)
}

func (s *MongoStore) list(ctx context.Context, filter bson.D, limit int, sortDir int) ([]*Intent, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: sortDir}}).SetLimit(int64(limit))
	cur, err := s.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*Intent{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ClaimPaymentSeen is the watcher's atomic claim (see Store). The status
// precondition makes concurrent claims race safely; the (network, txHash)
// unique index rejects recording an already-consumed transfer.
func (s *MongoStore) ClaimPaymentSeen(ctx context.Context, id bson.ObjectID, txHash, from string, receivedMicros int64) (*Intent, error) {
	var in Intent
	err := s.col.FindOneAndUpdate(ctx,
		bson.D{
			{Key: "_id", Value: id},
			{Key: "status", Value: bson.D{{Key: "$in", Value: []IntentStatus{StatusPending, StatusExpired}}}},
		},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: StatusConfirming},
			{Key: "txHash", Value: txHash},
			{Key: "fromAddress", Value: from},
			{Key: "amountReceivedMicros", Value: receivedMicros},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&in)
	if err != nil {
		if err == mongo.ErrNoDocuments || mongo.IsDuplicateKeyError(err) {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &in, nil
}

// MarkConfirmed finalizes a confirming intent.
func (s *MongoStore) MarkConfirmed(ctx context.Context, id bson.ObjectID, settlement string) error {
	now := time.Now().UTC()
	res, err := s.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "status", Value: StatusConfirming}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: StatusConfirmed},
			{Key: "settlement", Value: settlement},
			{Key: "confirmedAt", Value: now},
			{Key: "updatedAt", Value: now},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrConflict
	}
	return nil
}

// IncSettleAttempts bumps the retry counter (observability only).
func (s *MongoStore) IncSettleAttempts(ctx context.Context, id bson.ObjectID) error {
	_, err := s.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{
			{Key: "$inc", Value: bson.D{{Key: "settleAttempts", Value: 1}}},
			{Key: "$set", Value: bson.D{{Key: "updatedAt", Value: time.Now().UTC()}}},
		},
	)
	return err
}

// ExpireOne atomically expires one overdue pending intent.
func (s *MongoStore) ExpireOne(ctx context.Context, id bson.ObjectID, now time.Time) (*Intent, error) {
	var in Intent
	err := s.col.FindOneAndUpdate(ctx,
		bson.D{
			{Key: "_id", Value: id},
			{Key: "status", Value: StatusPending},
			{Key: "expiresAt", Value: bson.D{{Key: "$lt", Value: now}}},
		},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: StatusExpired},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&in)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &in, nil
}

// NextDerivationIndex atomically hands out the next HD index (0-based).
func (s *MongoStore) NextDerivationIndex(ctx context.Context) (uint32, error) {
	var doc struct {
		Seq int64 `bson:"seq"`
	}
	err := s.counters.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: derivationCounterID}},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "seq", Value: 1}}}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		return 0, err
	}
	return uint32(doc.Seq - 1), nil // post-inc seq 1 → index 0
}

// ListAll returns the paginated admin view, newest first.
func (s *MongoStore) ListAll(ctx context.Context, f IntentFilter, p pagination.Params) ([]*Intent, int64, error) {
	filter := bson.D{}
	if v := strings.TrimSpace(f.Status); v != "" {
		filter = append(filter, bson.E{Key: "status", Value: v})
	}
	if v := strings.TrimSpace(f.Purpose); v != "" {
		filter = append(filter, bson.E{Key: "purpose", Value: v})
	}
	total, err := s.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := s.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*Intent{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
