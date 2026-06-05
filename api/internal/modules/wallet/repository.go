package wallet

import (
	"context"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrInsufficientFunds is returned when a debit would overdraw the wallet.
var ErrInsufficientFunds = &apperrors.AppError{
	Code:    "INSUFFICIENT_FUNDS",
	Message: "wallet balance is insufficient for this charge",
	Err:     apperrors.ErrBadRequest,
}

// Repository defines persistence operations for wallet transactions. The
// authoritative balance lives on the user document (users.walletBalance);
// wallet_transactions is the immutable history/ledger.
type Repository interface {
	Create(ctx context.Context, tx *WalletTransaction) error
	FindByUserID(ctx context.Context, userID bson.ObjectID) ([]*WalletTransaction, error)
	GetBalance(ctx context.Context, userID bson.ObjectID) (float64, error)
	Credit(ctx context.Context, userID bson.ObjectID, amount float64) (float64, error)
	Debit(ctx context.Context, userID bson.ObjectID, amount float64) (float64, error)
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	tx    *mongo.Collection
	users *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository over the wallet_transactions
// and users collections.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		tx:    db.Collection("wallet_transactions"),
		users: db.Collection("users"),
	}
}

// EnsureIndexes creates the index the wallet ledger relies on.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("wallet_transactions").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}},
	})
	return err
}

// Create inserts a ledger row, stamping CreatedAt.
func (r *MongoRepository) Create(ctx context.Context, tx *WalletTransaction) error {
	if tx.ID.IsZero() {
		tx.ID = bson.NewObjectID()
	}
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now().UTC()
	}
	_, err := r.tx.InsertOne(ctx, tx)
	return err
}

// FindByUserID returns a user's ledger rows, newest first.
func (r *MongoRepository) FindByUserID(ctx context.Context, userID bson.ObjectID) ([]*WalletTransaction, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.tx.Find(ctx, bson.D{{Key: "userId", Value: userID}}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*WalletTransaction{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetBalance reads the authoritative balance from the user document.
func (r *MongoRepository) GetBalance(ctx context.Context, userID bson.ObjectID) (float64, error) {
	var doc struct {
		WalletBalance float64 `bson:"walletBalance"`
	}
	err := r.users.FindOne(ctx, bson.D{{Key: "_id", Value: userID}}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, apperrors.ErrNotFound
		}
		return 0, err
	}
	return doc.WalletBalance, nil
}

// Credit atomically increments the user's balance and returns the new balance.
func (r *MongoRepository) Credit(ctx context.Context, userID bson.ObjectID, amount float64) (float64, error) {
	return r.adjust(ctx, bson.D{{Key: "_id", Value: userID}}, amount)
}

// Debit atomically decrements the user's balance, but only when sufficient
// funds exist. A shortfall (no document matched the balance guard) returns
// ErrInsufficientFunds.
func (r *MongoRepository) Debit(ctx context.Context, userID bson.ObjectID, amount float64) (float64, error) {
	filter := bson.D{
		{Key: "_id", Value: userID},
		{Key: "walletBalance", Value: bson.D{{Key: "$gte", Value: amount}}},
	}
	bal, err := r.adjust(ctx, filter, -amount)
	if err == mongo.ErrNoDocuments {
		return 0, ErrInsufficientFunds
	}
	return bal, err
}

// adjust applies a single-document atomic $inc and returns the post-update
// balance. It surfaces mongo.ErrNoDocuments unchanged so callers can classify it.
func (r *MongoRepository) adjust(ctx context.Context, filter bson.D, delta float64) (float64, error) {
	var doc struct {
		WalletBalance float64 `bson:"walletBalance"`
	}
	err := r.users.FindOneAndUpdate(ctx,
		filter,
		bson.D{{Key: "$inc", Value: bson.D{{Key: "walletBalance", Value: delta}}},
			{Key: "$set", Value: bson.D{{Key: "updatedAt", Value: time.Now().UTC()}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		return 0, err
	}
	return doc.WalletBalance, nil
}
