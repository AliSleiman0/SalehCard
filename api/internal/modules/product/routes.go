package product

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/reseller"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/idcheck"
	"github.com/AliSleiman0/salehcard/api/internal/platform/ratelimit"
)

// RegisterRoutes wires up the product module and mounts all routes onto r.
// offers may be nil (disables live-offer price enrichment on catalog reads). cfg
// supplies the JWT secret + ID-verification provider for the authenticated
// verify-account route.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config, offers OfferLookup, categories CategoryResolver) {
	repo := NewMongoRepository(db)

	// Best-effort index creation at startup; log or handle errors in production.
	_ = EnsureIndexes(context.Background(), db)

	// Reseller catalog pricing uses the same repository the order module wires
	// as its pricing port (order/routes.go), so a reseller browses at exactly
	// the price checkout will charge. categories makes the ?categoryId= filter
	// tree-aware (list a node's whole subtree).
	svc := NewProductService(repo,
		WithOffers(offers),
		WithResellerPricing(reseller.NewMongoRepository(db)),
		WithCategoryResolver(categories),
	)
	h := NewHandler(svc)

	// Game-account ID verification (check_name). The verifier is selected by
	// IDCheckProvider; on misconfig (e.g. rapidapi selected without a key) fall
	// back to the stub so catalog reads still boot — verify then reports
	// "unavailable" and the client fails open.
	verifier, err := idcheck.New(idcheck.Config{Provider: cfg.IDCheckProvider, RapidAPIKey: cfg.RapidAPIKey})
	if err != nil {
		slog.Warn("product: ID-check provider misconfigured — falling back to stub", "provider", cfg.IDCheckProvider, "error", err)
		verifier = idcheck.StubVerifier{}
	}
	h.verifier = verifier

	// Per-IP rate limit on the verify route — each call hits a paid upstream API.
	limiter := ratelimit.New(ratelimit.Config{Provider: cfg.RateLimitProvider}, db)
	if _, ok := limiter.(ratelimit.NoopLimiter); !ok {
		if err := ratelimit.EnsureIndexes(context.Background(), db); err != nil {
			slog.Warn("product: failed to ensure rate-limit indexes", "error", err)
		}
	}
	verifyLimit := ratelimit.Middleware(limiter, cfg.RateLimitVerifyMax, cfg.RateLimitVerifyWindow, func(req *http.Request) string {
		return "verify:" + ratelimit.ClientIP(req)
	})

	// Read-only catalog. Mutations live exclusively under /api/admin/products
	// (RegisterAdminRoutes), behind auth.AdminOnly. auth.Optional identifies —
	// never requires — the caller so reseller sessions get their own pricing;
	// anonymous and customer requests are untouched. (On verify-account,
	// AuthRequired simply overwrites the same context key — harmless.)
	r.Route("/api/v1/products", func(r chi.Router) {
		r.Use(auth.Optional(cfg.JWTSecret))
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		// Authenticated + rate-limited: resolve a game player ID to its nickname
		// before purchase. RequireActive too — each call is a billed third-party
		// request, so a suspended/deleted account's lingering token must not
		// keep burning it.
		r.With(auth.AuthRequired(cfg.JWTSecret), auth.RequireActive(db), verifyLimit).Post("/{id}/verify-account", h.VerifyAccount)
	})
}
