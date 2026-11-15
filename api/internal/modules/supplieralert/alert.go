// Package supplieralert stores the supplier alerts the admin works through on
// the Suppliers page (and counts on the dashboard): a supplier's cost moved, a
// link was paused because it sells at a loss, the sync hid or re-showed a
// product, an exchange rate is missing, an order ran out of suppliers to fall
// back to. It is a leaf package (mongo only) so both the supplier sync job and
// the order engine can raise alerts without importing each other.
package supplieralert

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Alert kinds.
const (
	KindCostChanged       = "cost_changed"
	KindLossPaused        = "loss_paused"
	KindLossRecovered     = "loss_recovered"
	KindProductHidden     = "product_hidden"
	KindProductShown      = "product_shown"
	KindFxMissing         = "fx_missing"
	KindFallbackExhausted = "fallback_exhausted"
)

// Collection is the alerts collection name (the dashboard counts it directly).
const Collection = "supplier_alerts"

// Alert is one supplier alert. Key deduplicates: while an alert with the same
// key is open, raising it again only refreshes it (message, data, lastAt,
// count) instead of adding a row.
type Alert struct {
	ID          bson.ObjectID  `bson:"_id,omitempty"         json:"id"`
	Kind        string         `bson:"kind"                  json:"kind"`
	Key         string         `bson:"key"                   json:"key"`
	Open        bool           `bson:"open"                  json:"open"`
	Message     string         `bson:"message"               json:"message"`
	Provider    int            `bson:"provider,omitempty"    json:"provider,omitempty"`
	ProductID   string         `bson:"productId,omitempty"   json:"productId,omitempty"`
	ProductName string         `bson:"productName,omitempty" json:"productName,omitempty"`
	LinkID      string         `bson:"linkId,omitempty"      json:"linkId,omitempty"`
	OrderID     string         `bson:"orderId,omitempty"     json:"orderId,omitempty"`
	Data        map[string]any `bson:"data,omitempty"        json:"data,omitempty"`
	// DryRun: raised by a dry-run sync pass (nothing was changed).
	DryRun  bool       `bson:"dryRun,omitempty"  json:"dryRun,omitempty"`
	Count   int        `bson:"count"             json:"count"`
	FirstAt time.Time  `bson:"firstAt"           json:"firstAt"`
	LastAt  time.Time  `bson:"lastAt"            json:"lastAt"`
	AckAt   *time.Time `bson:"ackAt,omitempty"   json:"ackAt,omitempty"`
	AckBy   string     `bson:"ackBy,omitempty"   json:"ackBy,omitempty"`
}

// Raiser is the narrow port the sync job and the order engine raise through.
type Raiser interface {
	Raise(ctx context.Context, a Alert) error
	// Resolve closes the open alert with key (if any), e.g. a loss alert once
	// the cost recovered. by is recorded as the acknowledger.
	Resolve(ctx context.Context, key, by string) error
}

// Repo persists alerts.
type Repo struct {
	col *mongo.Collection
	now func() time.Time
}

// NewRepo builds the alerts repository.
func NewRepo(db *mongo.Database) *Repo {
	return &Repo{col: db.Collection(Collection), now: func() time.Time { return time.Now().UTC() }}
}

// EnsureIndexes creates the open-key dedupe index (unique among OPEN alerts
// only, so a key can be raised again after it was acknowledged) and the list
// index.
func (r *Repo) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "key", Value: 1}},
			Options: options.Index().
				SetName("open_key_unique").
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "open", Value: true}}),
		},
		{Keys: bson.D{{Key: "open", Value: 1}, {Key: "lastAt", Value: -1}}},
	})
	return err
}

// Raise opens an alert, or refreshes the open one with the same key.
func (r *Repo) Raise(ctx context.Context, a Alert) error {
	if a.Key == "" || a.Kind == "" {
		return errors.New("supplieralert: kind and key are required")
	}
	now := r.now()
	set := bson.D{
		{Key: "message", Value: a.Message},
		{Key: "lastAt", Value: now},
		{Key: "dryRun", Value: a.DryRun},
	}
	if a.Data != nil {
		set = append(set, bson.E{Key: "data", Value: a.Data})
	}
	if a.ProductName != "" {
		set = append(set, bson.E{Key: "productName", Value: a.ProductName})
	}
	onInsert := bson.D{
		{Key: "kind", Value: a.Kind},
		{Key: "firstAt", Value: now},
	}
	for _, f := range []struct {
		k string
		v any
		z bool
	}{
		{"provider", a.Provider, a.Provider == 0},
		{"productId", a.ProductID, a.ProductID == ""},
		{"linkId", a.LinkID, a.LinkID == ""},
		{"orderId", a.OrderID, a.OrderID == ""},
	} {
		if !f.z {
			onInsert = append(onInsert, bson.E{Key: f.k, Value: f.v})
		}
	}
	_, err := r.col.UpdateOne(ctx,
		bson.D{{Key: "key", Value: a.Key}, {Key: "open", Value: true}},
		bson.D{
			{Key: "$set", Value: set},
			{Key: "$setOnInsert", Value: onInsert},
			{Key: "$inc", Value: bson.D{{Key: "count", Value: 1}}},
		},
		options.UpdateOne().SetUpsert(true),
	)
	if mongo.IsDuplicateKeyError(err) {
		// Two raisers upserted the same open key at once: the other one won,
		// which is the dedupe working.
		return nil
	}
	return err
}

// Resolve closes the open alert with key (no-op when none is open).
func (r *Repo) Resolve(ctx context.Context, key, by string) error {
	now := r.now()
	_, err := r.col.UpdateOne(ctx,
		bson.D{{Key: "key", Value: key}, {Key: "open", Value: true}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "open", Value: false},
			{Key: "ackAt", Value: now},
			{Key: "ackBy", Value: by},
		}}},
	)
	return err
}

// ErrNotFound is returned by Ack for an unknown or already-closed alert.
var ErrNotFound = errors.New("supplieralert: no open alert with that id")

// Ack closes one open alert.
func (r *Repo) Ack(ctx context.Context, id bson.ObjectID, by string) (*Alert, error) {
	var out Alert
	err := r.col.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "open", Value: true}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "open", Value: false},
			{Key: "ackAt", Value: r.now()},
			{Key: "ackBy", Value: by},
		}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// AckAll closes every open alert, returning how many were closed.
func (r *Repo) AckAll(ctx context.Context, by string) (int64, error) {
	res, err := r.col.UpdateMany(ctx,
		bson.D{{Key: "open", Value: true}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "open", Value: false},
			{Key: "ackAt", Value: r.now()},
			{Key: "ackBy", Value: by},
		}}},
	)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// List returns alerts newest first: only open ones when open is true, else
// everything (capped by limit).
func (r *Repo) List(ctx context.Context, open bool, limit int64) ([]Alert, error) {
	filter := bson.D{}
	if open {
		filter = bson.D{{Key: "open", Value: true}}
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	cur, err := r.col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "lastAt", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	out := []Alert{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CountOpen counts the open alerts.
func (r *Repo) CountOpen(ctx context.Context) (int64, error) {
	return r.col.CountDocuments(ctx, bson.D{{Key: "open", Value: true}})
}
