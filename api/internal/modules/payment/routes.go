package payment

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/ratelimit"
)

// RegisterRoutes wires the customer payment routes onto r (AuthRequired):
// POST /api/v1/payments/usdt/topup-intents opens an on-chain top-up intent,
// GET /api/v1/payments/intents/{id} is the client's status poll, and
// GET /api/v1/payments/config is the feature gate. Creation and polling are
// per-IP rate-limited — creation burns HD addresses, polling fans into the
// chain provider budget.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config, svc *Service) {
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("payment: failed to ensure indexes", "error", err)
	}
	if err := EnsureDepositIndexes(context.Background(), db); err != nil {
		slog.Warn("payment: failed to ensure deposit indexes", "error", err)
	}
	h := NewHandler(svc)

	limiter := ratelimit.New(ratelimit.Config{Provider: cfg.RateLimitProvider}, db)
	if _, ok := limiter.(ratelimit.NoopLimiter); !ok {
		if err := ratelimit.EnsureIndexes(context.Background(), db); err != nil {
			slog.Warn("payment: failed to ensure rate-limit indexes", "error", err)
		}
	}
	createLimit := ratelimit.Middleware(limiter, cfg.RateLimitPaymentCreateMax, cfg.RateLimitPaymentCreateWindow, func(req *http.Request) string {
		return "paycreate:" + ratelimit.ClientIP(req)
	})
	pollLimit := ratelimit.Middleware(limiter, cfg.RateLimitPaymentPollMax, cfg.RateLimitPaymentPollWindow, func(req *http.Request) string {
		return "paypoll:" + ratelimit.ClientIP(req)
	})

	r.Route("/api/v1/payments", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Get("/config", h.GetConfig)
		r.With(createLimit).Post("/usdt/topup-intents", h.CreateTopUpIntent)
		r.With(pollLimit).Get("/intents/{id}", h.GetIntent)
	})
}
