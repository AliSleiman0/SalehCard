package wallet

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/blob"
	"github.com/AliSleiman0/salehcard/api/internal/platform/ratelimit"
)

// RegisterRoutes wires the customer-facing wallet routes onto r. /api/v1/wallet
// is guarded by AuthRequired. store backs the top-up document upload endpoint
// (payment-proof photos).
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config, store blob.Storage) {
	repo := NewMongoRepository(db)
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("wallet: failed to ensure indexes", "error", err)
	}
	if err := EnsureTopUpIndexes(context.Background(), db); err != nil {
		slog.Warn("wallet: failed to ensure topup indexes", "error", err)
	}
	if err := EnsureMethodIndexes(context.Background(), db); err != nil {
		slog.Warn("wallet: failed to ensure topup method indexes", "error", err)
	}
	svc := NewWalletService(repo, NewTopUpRepo(db)).WithMethods(NewMethodRepo(db))
	h := NewHandler(svc, store)

	// Per-IP rate limit on document uploads — each accepted upload writes billed
	// blob storage (reuses the KYC upload limiter config).
	limiter := ratelimit.New(ratelimit.Config{Provider: cfg.RateLimitProvider}, db)
	if _, ok := limiter.(ratelimit.NoopLimiter); !ok {
		if err := ratelimit.EnsureIndexes(context.Background(), db); err != nil {
			slog.Warn("wallet: failed to ensure rate-limit indexes", "error", err)
		}
	}
	uploadLimit := ratelimit.Middleware(limiter, cfg.RateLimitKycUploadMax, cfg.RateLimitKycUploadWindow, func(req *http.Request) string {
		return "topupupload:" + ratelimit.ClientIP(req)
	})

	r.Route("/api/v1/wallet", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Use(auth.RequireActive(db))
		r.Get("/", h.GetWallet)
		r.Post("/topups", h.TopUp)
		r.Get("/topups", h.ListTopUps)
		r.Get("/topup-methods", h.ListMethods)
		r.With(uploadLimit).Post("/topups/documents", h.UploadDocument)
	})
}

// NewService exposes a constructed Service for callers (e.g. the order module)
// that need the wallet payment surface without the top-up queue.
func NewService(db *mongo.Database) Service {
	return NewWalletService(NewMongoRepository(db), nil)
}
