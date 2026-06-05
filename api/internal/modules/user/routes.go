package user

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
)

// RegisterRoutes wires the customer-facing auth + profile routes onto r.
// These are public (outside the /api/admin group); /users/* is guarded by
// AuthRequired.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config) {
	repo := NewMongoRepository(db)
	refreshRepo := NewMongoRefreshRepository(db)
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("user: failed to ensure indexes", "error", err)
	}
	svc := NewUserService(repo, refreshRepo, cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	h := NewHandler(svc, cfg.CookieSecure)

	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
	})

	r.Route("/api/v1/users", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Get("/me", h.GetProfile)
		r.Patch("/me", h.UpdateProfile)
	})
}
