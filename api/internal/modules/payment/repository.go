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

// saltCounterID is the counters document behind shared-mode amount salts
// (same atomic $inc pattern as the derivation index).
const saltCounterID = "usdt_shared_amount_salt"

// saltRange bounds the shared-mode salt to 1..saltRange micros (≤ $0.009999 —
// deliberately below the settle path's excessCreditMinMicros so a salt can
// never trip the excess-credit branch). Never 0, so a shared expected amount
// is never round and a stray round-number deposit can't accidentally match.
const saltRange = 9999

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
	// NextAmountSalt hands out the next shared-mode amount salt (1..saltRange
	// micros, cycling).
	NextAmountSalt(ctx context.Context) (int64, error)
	// ReleaseSharedSlots frees the amount slots of shared intents that
	// expired unpaid and are past the late-payment grace — the moment the
	// watcher stops matching them, their amount becomes reusable.
	ReleaseSharedSlots(ctx context.Context, now time.Time, grace time.Duration) error
	// FilterKnownTxHashes reports which of hashes are already recorded on an
	// intent (the shared scan's already-consumed filter).
	FilterKnownTxHashes(ctx context.Context, network string, hashes []string) (map[string]bool, error)

	// Unmatched shared-address deposits (reconciliation queue) — deposits.go.
	RecordUnmatchedDeposit(ctx context.Context, d *Deposit) error
	ListDeposits(ctx context.Context, status string, p pagination.Params) ([]*Deposit, int64, error)
	GetDeposit(ctx context.Context, id bson.ObjectID) (*Deposit, error)
	ClaimDepositAttribution(ctx context.Context, id, userID bson.ObjectID, adminRef, note string) (*Deposit, error)
	RevertDepositAttribution(ctx context.Context, id bson.ObjectID) error
	MarkDepositIgnored(ctx context.Context, id bson.ObjectID, adminRef, note string) error

	ListAll(ctx context.Context, f IntentFilter, p pagination.Params) ([]*Intent, int64, error)
}

// MongoStore is the MongoDB-backed Store over payment_intents + usdt_deposits
// (+ counters).
type MongoStore struct {
	col      *mongo.Collection
	deposits *mongo.Collection
	counters *mongo.Collection
}

// NewMongoStore constructs a MongoStore.
func NewMongoStore(db *mongo.Database) *MongoStore {
	return &MongoStore{
		col:      db.Collection("payment_intents"),
		deposits: db.Collection("usdt_deposits"),
		counters: db.Collection("counters"),
	}
}

// EnsureIndexes creates the payment-intent indexes. The partial-unique
// (network, txHash) index is the double-credit guard: one observed transfer
// can be recorded on exactly one intent, ever.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	col := db.Collection("payment_intents")

	// One-time migration for shared-address mode: the original unconditional
	// unique index on address can't survive every shared intent carrying the
	// same address. Drop it (best-effort — fresh deployments never had it)
	// BEFORE CreateMany, or the options conflict fails the whole batch and
	// silently skips the other new indexes (routes.go only warn-logs).
	_ = col.Indexes().DropOne(ctx, "address_1")

	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "network", Value: 1}, {Key: "txHash", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "txHash", Value: bson.D{{Key: "$exists", Value: true}}}}),
		},
		// Every derived-mode intent owns its address; a duplicate would mean a
		// derivation-counter bug, so fail loudly at insert. Scoped to
		// addressMode:"derived" (new inserts always stamp it) — legacy docs
		// fall outside, which is fine: their uniqueness is historical fact and
		// the index is only a tripwire for future counter bugs. If the target
		// Mongo rejects the equality partial filter, this index is the only
		// casualty (derived uniqueness still holds via the atomic counter).
		{
			Keys: bson.D{{Key: "address", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "addressMode", Value: AddressModeDerived}}),
		},
		// Shared-mode collision guard: no two OPEN shared intents may expect
		// the same amount — exact-amount matching is only unambiguous under
		// this invariant. $exists is the same partial-filter operator the
		// other partial indexes here already rely on.
		{
			Keys: bson.D{{Key: "amountExpectedMicros", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "sharedOpen", Value: bson.D{{Key: "$exists", Value: true}}}}),
		},
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

// MarkConfirmed finalizes a confirming intent, releasing its shared-mode
// amount slot (the $unset is a no-op for derived intents).
func (s *MongoStore) MarkConfirmed(ctx context.Context, id bson.ObjectID, settlement string) error {
	now := time.Now().UTC()
	res, err := s.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "status", Value: StatusConfirming}},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "status", Value: StatusConfirmed},
				{Key: "settlement", Value: settlement},
				{Key: "confirmedAt", Value: now},
				{Key: "updatedAt", Value: now},
			}},
			{Key: "$unset", Value: bson.D{{Key: "sharedOpen", Value: ""}}},
		},
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
	seq, err := s.nextCounter(ctx, derivationCounterID)
	if err != nil {
		return 0, err
	}
	return uint32(seq - 1), nil // post-inc seq 1 → index 0
}

// NextAmountSalt atomically hands out the next shared-mode amount salt,
// cycling 1..saltRange (never 0 — see saltRange).
func (s *MongoStore) NextAmountSalt(ctx context.Context) (int64, error) {
	seq, err := s.nextCounter(ctx, saltCounterID)
	if err != nil {
		return 0, err
	}
	return 1 + ((seq - 1) % saltRange), nil
}

// nextCounter is the shared atomic $inc over one counters document.
func (s *MongoStore) nextCounter(ctx context.Context, id string) (int64, error) {
	var doc struct {
		Seq int64 `bson:"seq"`
	}
	err := s.counters.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "seq", Value: 1}}}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		return 0, err
	}
	return doc.Seq, nil
}

// ReleaseSharedSlots frees the amount slots of shared intents that expired
// unpaid and whose late-payment grace has ended — matching stops and the
// amount becomes reusable at the same moment. Without this, every expired
// $10 intent would burn one of the 9999 salt slots for that base forever.
func (s *MongoStore) ReleaseSharedSlots(ctx context.Context, now time.Time, grace time.Duration) error {
	_, err := s.col.UpdateMany(ctx,
		bson.D{
			{Key: "sharedOpen", Value: bson.D{{Key: "$exists", Value: true}}},
			{Key: "status", Value: StatusExpired},
			{Key: "txHash", Value: bson.D{{Key: "$exists", Value: false}}},
			{Key: "expiresAt", Value: bson.D{{Key: "$lt", Value: now.Add(-grace)}}},
		},
		bson.D{
			{Key: "$unset", Value: bson.D{{Key: "sharedOpen", Value: ""}}},
			{Key: "$set", Value: bson.D{{Key: "updatedAt", Value: time.Now().UTC()}}},
		},
	)
	return err
}

// FilterKnownTxHashes reports which of hashes are already recorded on any
// intent — the shared scan uses it to skip transfers consumed on a prior tick.
func (s *MongoStore) FilterKnownTxHashes(ctx context.Context, network string, hashes []string) (map[string]bool, error) {
	known := make(map[string]bool, len(hashes))
	if len(hashes) == 0 {
		return known, nil
	}
	cur, err := s.col.Find(ctx,
		bson.D{
			{Key: "network", Value: network},
			{Key: "txHash", Value: bson.D{{Key: "$in", Value: hashes}}},
		},
		options.Find().SetProjection(bson.D{{Key: "txHash", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []struct {
		TxHash string `bson:"txHash"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	for _, d := range docs {
		known[d.TxHash] = true
	}
	return known, nil
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
