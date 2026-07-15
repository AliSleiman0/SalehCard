package supplier

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// settingsRepo persists per-supplier operational settings in supplier_settings,
// one document per provider id (int _id). The default _id index suffices — no
// EnsureIndexes needed.
type settingsRepo struct {
	col *mongo.Collection
}

func newSettingsRepo(db *mongo.Database) *settingsRepo {
	return &settingsRepo{col: db.Collection("supplier_settings")}
}

// Get returns the settings for one supplier, or zero-value defaults when none
// have been saved yet.
func (r *settingsRepo) Get(ctx context.Context, id int) (Settings, error) {
	var s Settings
	err := r.col.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&s)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Settings{ID: id}, nil
		}
		return Settings{}, err
	}
	return s, nil
}

// All returns every saved supplier-settings row keyed by provider id (missing
// ids simply won't appear — the caller layers defaults).
func (r *settingsRepo) All(ctx context.Context) (map[int]Settings, error) {
	cur, err := r.col.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := map[int]Settings{}
	for cur.Next(ctx) {
		var s Settings
		if err := cur.Decode(&s); err != nil {
			return nil, err
		}
		out[s.ID] = s
	}
	return out, cur.Err()
}

// Update applies the non-nil fields of in over the current settings and upserts
// the single document, returning the post-update value. Read-modify-write keeps
// the merge simple (mirrors the settings module), avoiding $set/$setOnInsert
// conflicts on a single-document upsert.
func (r *settingsRepo) Update(ctx context.Context, id int, in SettingsInput, actor string) (Settings, error) {
	cur, err := r.Get(ctx, id)
	if err != nil {
		return Settings{}, err
	}
	if in.LowBalanceThreshold != nil {
		cur.LowBalanceThreshold = *in.LowBalanceThreshold
	}
	if in.MarkupPercent != nil {
		cur.MarkupPercent = *in.MarkupPercent
	}
	cur.ID = id
	cur.UpdatedAt = time.Now().UTC()
	cur.UpdatedBy = actor
	if _, err := r.col.ReplaceOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		cur,
		options.Replace().SetUpsert(true),
	); err != nil {
		return Settings{}, err
	}
	return cur, nil
}
