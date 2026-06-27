package kyc

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
)

// RegisterRoutes wires the customer-facing KYC routes onto r (guarded by
// AuthRequired): POST /api/v1/kyc submits background info; GET /api/v1/kyc/me
// returns the caller's derived verification status.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config) {
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("kyc: failed to ensure indexes", "error", err)
	}
	repo := NewMongoRepository(db.Collection("kyc_submissions"), db.Collection("users"))
	h := NewHandler(NewService(repo))

	r.Route("/api/v1/kyc", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Post("/", h.Submit)
		r.Get("/me", h.GetMe)
	})
}
