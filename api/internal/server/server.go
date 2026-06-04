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
	product "github.com/AliSleiman0/salehcard/api/internal/modules/product"
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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	s.router.Use(c.Handler)

	s.router.Get("/health", s.handleHealth)

	product.RegisterRoutes(s.router, s.db)
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
