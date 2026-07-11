package kyc

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

// RegisterRoutes wires the customer-facing KYC routes onto r (guarded by
// AuthRequired): POST /api/v1/kyc submits background info + document photo
// URLs; POST /api/v1/kyc/documents uploads one document photo (rate-limited);
// GET /api/v1/kyc/me returns the caller's derived verification status.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config, store blob.Storage) {
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("kyc: failed to ensure indexes", "error", err)
	}
	repo := NewMongoRepository(db.Collection("kyc_submissions"), db.Collection("users"))
	h := NewHandler(NewService(repo), store)

	// Per-IP rate limit on document uploads — each accepted upload writes
	// billed blob storage.
	limiter := ratelimit.New(ratelimit.Config{Provider: cfg.RateLimitProvider}, db)
	if _, ok := limiter.(ratelimit.NoopLimiter); !ok {
		if err := ratelimit.EnsureIndexes(context.Background(), db); err != nil {
			slog.Warn("kyc: failed to ensure rate-limit indexes", "error", err)
		}
	}
	uploadLimit := ratelimit.Middleware(limiter, cfg.RateLimitKycUploadMax, cfg.RateLimitKycUploadWindow, func(req *http.Request) string {
		return "kycupload:" + ratelimit.ClientIP(req)
	})

	r.Route("/api/v1/kyc", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Use(auth.RequireActive(db))
		r.Post("/", h.Submit)
		r.Get("/me", h.GetMe)
		r.With(uploadLimit).Post("/documents", h.UploadDocument)
	})
}
