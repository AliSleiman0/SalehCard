package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/category"
	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/internal/modules/dashboard"
	"github.com/AliSleiman0/salehcard/api/internal/modules/finance"
	"github.com/AliSleiman0/salehcard/api/internal/modules/order"
	product "github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/promo"
	"github.com/AliSleiman0/salehcard/api/internal/modules/reseller"
	"github.com/AliSleiman0/salehcard/api/internal/modules/review"
	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

const pingTimeout = 5 * time.Second

// Server holds the HTTP router, database connection, and application config.
type Server struct {
	router *chi.Mux
	db     *mongo.Database
	cfg    *config.Config
}

// New creates a new Server instance with the provided config and database.
func New(cfg *config.Config, db *mongo.Database) *Server {
	return &Server{
		router: chi.NewRouter(),
		db:     db,
		cfg:    cfg,
	}
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Routes registers all middleware and application routes.
func (s *Server) Routes() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	// Configure CORS using allowed origins from config.
	allowedOrigins := strings.Split(s.cfg.AllowedOrigins, ",")
	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "Idempotency-Key", "X-Client"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	s.router.Use(c.Handler)

	s.router.Get("/health", s.handleHealth)

	// Customer-facing routes (read-only product catalog stays separate).
	product.RegisterRoutes(s.router, s.db)

	// Read-only category taxonomy (storefront browses by root domain).
	category.RegisterRoutes(s.router, s.db)

	// Customer auth + profile (public; /users/* guarded by AuthRequired).
	user.RegisterRoutes(s.router, s.db, s.cfg)

	// Customer orders + wallet + promo validation + review submission
	// (guarded by AuthRequired).
	order.RegisterRoutes(s.router, s.db, s.cfg)
	wallet.RegisterRoutes(s.router, s.db, s.cfg)
	promo.RegisterRoutes(s.router, s.db, s.cfg)
	review.RegisterRoutes(s.router, s.db, s.cfg)

	// Admin route group — every /api/admin/* route requires an `admin` JWT role
	// (AdminOnly bypasses in dev when no JWT secret is configured).
	s.router.Route("/api/admin", func(r chi.Router) {
		r.Use(auth.AdminOnly(s.cfg.JWTSecret))

		// Fully implemented (extends the product reference slice):
		product.RegisterAdminRoutes(r, s.db)
		code.RegisterAdminRoutes(r, s.db)
		dashboard.RegisterAdminRoutes(r, s.db)
		finance.RegisterAdminRoutes(r, s.db)

		// Route map registered; handlers stubbed (501) pending implementation:
		order.RegisterAdminRoutes(r, s.db)
		user.RegisterAdminRoutes(r, s.db)
		reseller.RegisterAdminRoutes(r, s.db)
		promo.RegisterAdminRoutes(r, s.db)
		review.RegisterAdminRoutes(r, s.db)
		settings.RegisterAdminRoutes(r, s.db)
	})
}

// handleHealth pings MongoDB and returns a status response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), pingTimeout)
	defer cancel()

	if err := s.db.Client().Ping(ctx, nil); err != nil {
		response.Error(w, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
