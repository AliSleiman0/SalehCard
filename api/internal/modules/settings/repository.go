package settings

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository persists the single app-settings document.
type Repository interface {
	Get(ctx context.Context) (*Settings, error)
	Update(ctx context.Context, in UpdateInput, actor string) (*Settings, error)
}

// MongoRepository is a MongoDB-backed Repository over the app_settings
// collection (a one-row collection keyed on the fixed settingsID).
type MongoRepository struct {
	col *mongo.Collection
}

// NewMongoRepository constructs a MongoRepository over the app_settings collection.
func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{col: db.Collection("app_settings")}
}

// Get returns the saved settings, or the baseline defaults when none exist yet.
func (r *MongoRepository) Get(ctx context.Context) (*Settings, error) {
	var s Settings
	err := r.col.FindOne(ctx, bson.D{{Key: "_id", Value: settingsID}}).Decode(&s)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return defaults(), nil
		}
		return nil, err
	}
	return &s, nil
}

// Update applies the non-nil fields of in over the current settings (defaults on
// first write) and upserts the singleton, returning the post-update document.
// Read-modify-write in the app keeps the merge simple and avoids $set/$setOnInsert
// field conflicts on the single-document upsert.
func (r *MongoRepository) Update(ctx context.Context, in UpdateInput, actor string) (*Settings, error) {
	cur, err := r.Get(ctx)
	if err != nil {
		return nil, err
	}
	if in.StoreName != nil {
		cur.StoreName = strings.TrimSpace(*in.StoreName)
	}
	if in.SupportEmail != nil {
		cur.SupportEmail = strings.TrimSpace(*in.SupportEmail)
	}
	if in.SupportPhone != nil {
		cur.SupportPhone = strings.TrimSpace(*in.SupportPhone)
	}
	if in.DefaultLanguage != nil {
		cur.DefaultLanguage = *in.DefaultLanguage
	}
	if in.DefaultCurrency != nil {
		cur.DefaultCurrency = *in.DefaultCurrency
	}
	if in.LowStockThreshold != nil {
		cur.LowStockThreshold = *in.LowStockThreshold
	}
	if in.MaintenanceMode != nil {
		cur.MaintenanceMode = *in.MaintenanceMode
	}
	if in.LoyaltyEnabled != nil {
		cur.LoyaltyEnabled = *in.LoyaltyEnabled
	}
	if in.LoyaltyEarnUsdPerPoint != nil {
		cur.LoyaltyEarnUsdPerPoint = *in.LoyaltyEarnUsdPerPoint
	}
	if in.AdminSmsTwoFactorEnabled != nil {
		cur.AdminSmsTwoFactorEnabled = *in.AdminSmsTwoFactorEnabled
	}
	cur.ID = settingsID
	cur.UpdatedAt = time.Now().UTC()
	cur.UpdatedBy = actor

	if _, err := r.col.ReplaceOne(ctx,
		bson.D{{Key: "_id", Value: settingsID}},
		cur,
		options.Replace().SetUpsert(true),
	); err != nil {
		return nil, err
	}
	return cur, nil
}
