package review

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
)

// RegisterRoutes wires the customer-facing review routes onto r.
// POST /api/v1/reviews (guarded by AuthRequired) lets a logged-in customer
// submit a review, which is stored pending admin moderation.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config) {
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("review: failed to ensure indexes", "error", err)
	}
	repo := NewMongoRepository(db.Collection("reviews"), db.Collection("products"))
	h := NewHandler(NewReviewService(repo))

	r.Route("/api/v1/reviews", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Post("/", h.Create)
	})

	// Public: the storefront product page reads a product's approved reviews.
	r.Get("/api/v1/products/{id}/reviews", h.ListForProduct)
}
