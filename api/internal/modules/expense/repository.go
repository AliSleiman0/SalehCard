package expense

import (
	"context"
	"time"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository defines persistence operations for Expense entities.
type Repository interface {
	FindByID(ctx context.Context, id bson.ObjectID) (*Expense, error)
	List(ctx context.Context, f ExpenseFilter, p pagination.Params) ([]*Expense, int64, error)
	Summarize(ctx context.Context, f ExpenseFilter) (Summary, error)
	Create(ctx context.Context, e *Expense) error
	Update(ctx context.Context, id bson.ObjectID, fields ExpenseUpdate) (*Expense, error)
	Delete(ctx context.Context, id bson.ObjectID) error
}

// ExpenseUpdate carries the editable fields of an expense (admin edit).
type ExpenseUpdate struct {
	Amount        float64
	Currency      string
	Category      ExpenseCategory
	CategoryOther string
	Note          string
	IncurredAt    time.Time
}

// ExpenseFilter narrows an admin expense listing/summary. Zero-valued fields
// are ignored. From/To bound the incurredAt date range (inclusive).
type ExpenseFilter struct {
	Category string
	Currency string
	From     *time.Time
	To       *time.Time
	Search   string
}

// build assembles the MongoDB filter document for f.
func (f ExpenseFilter) build() bson.D {
	and := bson.A{}
	if f.Category != "" {
		and = append(and, bson.D{{Key: "category", Value: f.Category}})
	}
	if f.Currency != "" {
		and = append(and, bson.D{{Key: "currency", Value: f.Currency}})
	}
	if f.From != nil {
		and = append(and, bson.D{{Key: "incurredAt", Value: bson.D{{Key: "$gte", Value: *f.From}}}})
	}
	if f.To != nil {
		and = append(and, bson.D{{Key: "incurredAt", Value: bson.D{{Key: "$lte", Value: *f.To}}}})
	}
	if f.Search != "" {
		and = append(and, bson.D{{Key: "note", Value: bson.D{
			{Key: "$regex", Value: f.Search},
			{Key: "$options", Value: "i"},
		}}})
	}
	if len(and) == 0 {
		return bson.D{}
	}
	return bson.D{{Key: "$and", Value: and}}
}

// CategoryTotal is the spend within a (currency, category) bucket.
type CategoryTotal struct {
	Category string  `json:"category"`
	Currency string  `json:"currency"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
}

// CurrencyTotal is the grand total of spend in a single currency.
type CurrencyTotal struct {
	Currency string  `json:"currency"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
}

// Summary is the aggregate spend for a filter: grand totals per currency plus a
// per-currency/category breakdown. Totals are never summed across currencies.
type Summary struct {
	Totals     []CurrencyTotal `json:"totals"`
	ByCategory []CategoryTotal `json:"byCategory"`
}

// MongoRepository is a MongoDB-backed implementation of Repository.
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository using the given collection.
func NewMongoRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: col}
}

// EnsureIndexes creates the expense indexes: a date index for the default sort
// and a compound index supporting category-filtered date listings.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("expenses").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "incurredAt", Value: -1}}},
		{Keys: bson.D{{Key: "category", Value: 1}, {Key: "incurredAt", Value: -1}}},
	})
	return err
}

// FindByID retrieves an expense by ObjectID, ErrNotFound when absent.
func (r *MongoRepository) FindByID(ctx context.Context, id bson.ObjectID) (*Expense, error) {
	var e Expense
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&e)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

// List returns a paginated slice of expenses matching f, newest incurred first.
func (r *MongoRepository) List(ctx context.Context, f ExpenseFilter, p pagination.Params) ([]*Expense, int64, error) {
	filter := f.build()
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "incurredAt", Value: -1}, {Key: "createdAt", Value: -1}})
	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*Expense{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// Summarize aggregates total spend for f across all matching rows (ignoring
// pagination): grand totals per currency and a per-currency/category breakdown.
func (r *MongoRepository) Summarize(ctx context.Context, f ExpenseFilter) (Summary, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: f.build()}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "currency", Value: "$currency"},
				{Key: "category", Value: "$category"},
			}},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return Summary{}, err
	}
	defer cur.Close(ctx)

	var rows []struct {
		ID struct {
			Currency string `bson:"currency"`
			Category string `bson:"category"`
		} `bson:"_id"`
		Total float64 `bson:"total"`
		Count int     `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return Summary{}, err
	}

	byCategory := make([]CategoryTotal, 0, len(rows))
	totalsByCurrency := map[string]*CurrencyTotal{}
	for _, row := range rows {
		byCategory = append(byCategory, CategoryTotal{
			Category: row.ID.Category,
			Currency: row.ID.Currency,
			Total:    row.Total,
			Count:    row.Count,
		})
		ct, ok := totalsByCurrency[row.ID.Currency]
		if !ok {
			ct = &CurrencyTotal{Currency: row.ID.Currency}
			totalsByCurrency[row.ID.Currency] = ct
		}
		ct.Total += row.Total
		ct.Count += row.Count
	}

	totals := make([]CurrencyTotal, 0, len(totalsByCurrency))
	for _, ct := range totalsByCurrency {
		totals = append(totals, *ct)
	}
	return Summary{Totals: totals, ByCategory: byCategory}, nil
}

// Create inserts a new expense, stamping CreatedAt/UpdatedAt.
func (r *MongoRepository) Create(ctx context.Context, e *Expense) error {
	if e.ID.IsZero() {
		e.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now
	_, err := r.collection.InsertOne(ctx, e)
	return err
}

// Update atomically sets an expense's editable fields and returns the
// post-update document. ErrNotFound when no expense matches.
func (r *MongoRepository) Update(ctx context.Context, id bson.ObjectID, fields ExpenseUpdate) (*Expense, error) {
	var e Expense
	err := r.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "amount", Value: fields.Amount},
			{Key: "currency", Value: fields.Currency},
			{Key: "category", Value: fields.Category},
			{Key: "categoryOther", Value: fields.CategoryOther},
			{Key: "note", Value: fields.Note},
			{Key: "incurredAt", Value: fields.IncurredAt},
			{Key: "updatedAt", Value: time.Now().UTC()},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&e)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

// Delete removes an expense by ObjectID, ErrNotFound when absent.
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
