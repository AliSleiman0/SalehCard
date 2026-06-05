package wallet

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
)

// RegisterRoutes wires the customer-facing wallet routes onto r. /api/v1/wallet
// is guarded by AuthRequired.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config) {
	repo := NewMongoRepository(db)
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("wallet: failed to ensure indexes", "error", err)
	}
	svc := NewWalletService(repo)
	h := NewHandler(svc)

	r.Route("/api/v1/wallet", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Get("/", h.GetWallet)
		r.Post("/topups", h.TopUp)
	})
}

// NewService exposes a constructed Service for callers (e.g. the order module)
// that need wallet operations without re-wiring the repository.
func NewService(db *mongo.Database) Service {
	return NewWalletService(NewMongoRepository(db))
}
