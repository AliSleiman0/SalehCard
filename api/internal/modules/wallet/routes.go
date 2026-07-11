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
	if err := EnsureTopUpIndexes(context.Background(), db); err != nil {
		slog.Warn("wallet: failed to ensure topup indexes", "error", err)
	}
	svc := NewWalletService(repo, NewTopUpRepo(db))
	h := NewHandler(svc)

	r.Route("/api/v1/wallet", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Use(auth.RequireActive(db))
		r.Get("/", h.GetWallet)
		r.Post("/topups", h.TopUp)
		r.Get("/topups", h.ListTopUps)
	})
}

// NewService exposes a constructed Service for callers (e.g. the order module)
// that need the wallet payment surface without the top-up queue.
func NewService(db *mongo.Database) Service {
	return NewWalletService(NewMongoRepository(db), nil)
}
