package wallet

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

// TopUpStore defines persistence for top-up requests. It is deliberately
// separate from Repository (the balance/ledger surface shared with the order
// module) so consumers of that interface are untouched.
type TopUpStore interface {
	Create(ctx context.Context, req *TopUpRequest) error
	FindByUser(ctx context.Context, userID bson.ObjectID) ([]*TopUpRequest, error)
	CountPendingForUser(ctx context.Context, userID bson.ObjectID) (int64, error)
	List(ctx context.Context, status string, p pagination.Params) ([]*TopUpRequest, int64, error)
	Claim(ctx context.Context, id bson.ObjectID, to TopUpStatus, decidedBy, reason string) (*TopUpRequest, error)
	Revert(ctx context.Context, id bson.ObjectID) error
	SetTxID(ctx context.Context, id bson.ObjectID, txID string) error
}

// TopUpRepo is the MongoDB-backed TopUpStore over topup_requests.
type TopUpRepo struct {
	col *mongo.Collection
}

// NewTopUpRepo constructs a TopUpRepo over topup_requests.
func NewTopUpRepo(db *mongo.Database) *TopUpRepo {
	return &TopUpRepo{col: db.Collection("topup_requests")}
}

// EnsureTopUpIndexes creates the queue and per-user history indexes.
func EnsureTopUpIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("topup_requests").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
	})
	return err
}

// Create inserts a new pending request, stamping ID/status/CreatedAt.
func (r *TopUpRepo) Create(ctx context.Context, req *TopUpRequest) error {
	if req.ID.IsZero() {
		req.ID = bson.NewObjectID()
	}
	req.Status = TopUpPending
	req.CreatedAt = time.Now().UTC()
	_, err := r.col.InsertOne(ctx, req)
	return err
}

// FindByUser returns the user's requests, newest first.
func (r *TopUpRepo) FindByUser(ctx context.Context, userID bson.ObjectID) ([]*TopUpRequest, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(50)
	cur, err := r.col.Find(ctx, bson.D{{Key: "userId", Value: userID}}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*TopUpRequest{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CountPendingForUser counts the user's open requests (the spam guard).
func (r *TopUpRepo) CountPendingForUser(ctx context.Context, userID bson.ObjectID) (int64, error) {
	return r.col.CountDocuments(ctx, bson.D{
		{Key: "userId", Value: userID},
		{Key: "status", Value: TopUpPending},
	})
}

// List returns a paginated admin view, optionally filtered by status, newest
// first (pending queues are typically consumed oldest-first via the UI sort).
func (r *TopUpRepo) List(ctx context.Context, status string, p pagination.Params) ([]*TopUpRequest, int64, error) {
	filter := bson.D{}
	if s := strings.TrimSpace(status); s != "" {
		filter = bson.D{{Key: "status", Value: s}}
	}
	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*TopUpRequest{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// Claim atomically moves a PENDING request to `to`, stamping the decision.
// The pending precondition makes the single document write the double-approve
// lock: of two concurrent decisions exactly one matches. Returns the
// post-decision document; ErrConflict when the request was already decided,
// ErrNotFound when it does not exist.
func (r *TopUpRepo) Claim(ctx context.Context, id bson.ObjectID, to TopUpStatus, decidedBy, reason string) (*TopUpRequest, error) {
	now := time.Now().UTC()
	var req TopUpRequest
	err := r.col.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "status", Value: TopUpPending}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: to},
			{Key: "decidedBy", Value: decidedBy},
			{Key: "decisionReason", Value: reason},
			{Key: "decidedAt", Value: now},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&req)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			if cnt, cerr := r.col.CountDocuments(ctx, bson.D{{Key: "_id", Value: id}}); cerr == nil && cnt == 0 {
				return nil, apperrors.ErrNotFound
			}
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &req, nil
}

// Revert puts a claimed request back to pending (compensation after a failed
// credit/ledger write) so the admin can retry.
func (r *TopUpRepo) Revert(ctx context.Context, id bson.ObjectID) error {
	_, err := r.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{
			{Key: "$set", Value: bson.D{{Key: "status", Value: TopUpPending}}},
			{Key: "$unset", Value: bson.D{
				{Key: "decidedBy", Value: ""},
				{Key: "decisionReason", Value: ""},
				{Key: "decidedAt", Value: ""},
			}},
		},
	)
	return err
}

// SetTxID stamps the ledger transaction id on an approved request (best-effort).
func (r *TopUpRepo) SetTxID(ctx context.Context, id bson.ObjectID, txID string) error {
	_, err := r.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "txId", Value: txID}}}},
	)
	return err
}
