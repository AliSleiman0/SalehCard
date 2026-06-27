package offer

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
)

// RegisterRoutes wires the customer-facing offer route onto r. GET /api/v1/offers
// is guarded by AuthRequired — the storefront Offers tab is an authenticated
// screen (the mobile client sends its Bearer token via the auth interceptor).
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config) {
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("offer: failed to ensure indexes", "error", err)
	}
	repo := NewMongoRepository(db.Collection("offers"))
	products := product.NewProductService(product.NewMongoRepository(db))
	h := NewHandler(repo, products)

	r.With(auth.AuthRequired(cfg.JWTSecret)).Get("/api/v1/offers", h.List)
}
