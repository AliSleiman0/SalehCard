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

// Deposit statuses. A deposit is born unmatched (a transfer to the shared
// address that matched no open intent); an admin either attributes it to a
// customer (credited — wallet top-up) or ignores it (dust/spam/unknown).
const (
	DepositUnmatched = "unmatched"
	DepositCredited  = "credited"
	DepositIgnored   = "ignored"
)

// Deposit is one on-chain transfer into the shared deposit address that the
// watcher could not match to any open intent — the admin reconciliation queue.
// The unique (network, txHash) index makes recording idempotent across ticks.
type Deposit struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Network     string        `bson:"network"       json:"network"`
	TxHash      string        `bson:"txHash"        json:"txHash"`
	FromAddress string        `bson:"fromAddress"   json:"fromAddress"`
	ToAddress   string        `bson:"toAddress"     json:"toAddress"`
	// AmountMicros is integer micro-USDT, like Intent amounts.
	AmountMicros int64     `bson:"amountMicros" json:"-"`
	BlockTime    time.Time `bson:"blockTime"    json:"blockTime"`
	SeenAt       time.Time `bson:"seenAt"       json:"seenAt"`

	Status string `bson:"status" json:"status"`
	// Attribution audit trail (set on credited/ignored).
	AttributedUserID *bson.ObjectID `bson:"attributedUserId,omitempty" json:"attributedUserId,omitempty"`
	AttributedBy     string         `bson:"attributedBy,omitempty"     json:"attributedBy,omitempty"`
	AttributedAt     *time.Time     `bson:"attributedAt,omitempty"     json:"attributedAt,omitempty"`
	Note             string         `bson:"note,omitempty"             json:"note,omitempty"`
}

// EnsureDepositIndexes creates the usdt_deposits indexes: the unique
// (network, txHash) recording guard and the admin listing sort.
func EnsureDepositIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("usdt_deposits").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "network", Value: 1}, {Key: "txHash", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "seenAt", Value: -1}}},
	})
	return err
}

// RecordUnmatchedDeposit stores an observed-but-unmatched transfer. Upsert on
// (network, txHash) with $setOnInsert so a transfer re-seen on every tick is
// recorded exactly once and a later attribution is never clobbered.
func (s *MongoStore) RecordUnmatchedDeposit(ctx context.Context, d *Deposit) error {
	if d.SeenAt.IsZero() {
		d.SeenAt = time.Now().UTC()
	}
	if d.Status == "" {
		d.Status = DepositUnmatched
	}
	_, err := s.deposits.UpdateOne(ctx,
		bson.D{{Key: "network", Value: d.Network}, {Key: "txHash", Value: d.TxHash}},
		bson.D{{Key: "$setOnInsert", Value: bson.D{
			{Key: "fromAddress", Value: d.FromAddress},
			{Key: "toAddress", Value: d.ToAddress},
			{Key: "amountMicros", Value: d.AmountMicros},
			{Key: "blockTime", Value: d.BlockTime},
			{Key: "seenAt", Value: d.SeenAt},
			{Key: "status", Value: d.Status},
		}}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// ListDeposits returns the paginated reconciliation queue, newest first.
func (s *MongoStore) ListDeposits(ctx context.Context, status string, p pagination.Params) ([]*Deposit, int64, error) {
	filter := bson.D{}
	if v := strings.TrimSpace(status); v != "" {
		filter = append(filter, bson.E{Key: "status", Value: v})
	}
	total, err := s.deposits.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "seenAt", Value: -1}})
	cur, err := s.deposits.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*Deposit{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// GetDeposit retrieves one deposit, ErrNotFound when absent.
func (s *MongoStore) GetDeposit(ctx context.Context, id bson.ObjectID) (*Deposit, error) {
	var d Deposit
	err := s.deposits.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&d)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

// ClaimDepositAttribution atomically moves an unmatched deposit to credited,
// recording who it was attributed to and by whom. The status precondition is
// the two-admins guard: the loser gets ErrConflict, so one deposit can never
// credit two different customers.
func (s *MongoStore) ClaimDepositAttribution(ctx context.Context, id, userID bson.ObjectID, adminRef, note string) (*Deposit, error) {
	var d Deposit
	err := s.deposits.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "status", Value: DepositUnmatched}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: DepositCredited},
			{Key: "attributedUserId", Value: userID},
			{Key: "attributedBy", Value: adminRef},
			{Key: "attributedAt", Value: time.Now().UTC()},
			{Key: "note", Value: note},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&d)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &d, nil
}

// RevertDepositAttribution is the best-effort compensation when the wallet
// credit fails after a successful claim: put the deposit back in the queue.
func (s *MongoStore) RevertDepositAttribution(ctx context.Context, id bson.ObjectID) error {
	_, err := s.deposits.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "status", Value: DepositCredited}},
		bson.D{
			{Key: "$set", Value: bson.D{{Key: "status", Value: DepositUnmatched}}},
			{Key: "$unset", Value: bson.D{
				{Key: "attributedUserId", Value: ""},
				{Key: "attributedBy", Value: ""},
				{Key: "attributedAt", Value: ""},
			}},
		},
	)
	return err
}

// MarkDepositIgnored moves an unmatched deposit to ignored (dust/spam).
func (s *MongoStore) MarkDepositIgnored(ctx context.Context, id bson.ObjectID, adminRef, note string) error {
	res, err := s.deposits.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "status", Value: DepositUnmatched}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: DepositIgnored},
			{Key: "attributedBy", Value: adminRef},
			{Key: "attributedAt", Value: time.Now().UTC()},
			{Key: "note", Value: note},
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
