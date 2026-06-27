package promo

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
)

// RegisterRoutes wires the customer-facing promo routes onto r.
// POST /api/v1/promos/validate (guarded by AuthRequired) lets the storefront
// preview a code's discount before checkout; the order service applies it
// authoritatively at placement.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config) {
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("promo: failed to ensure indexes", "error", err)
	}
	svc := NewPromoService(NewMongoRepository(db.Collection("promos")))
	h := NewHandler(svc)

	r.Route("/api/v1/promos", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Post("/validate", h.Validate)
	})
}
