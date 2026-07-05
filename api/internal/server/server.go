package server

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/audit"
	"github.com/AliSleiman0/salehcard/api/internal/modules/category"
	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/internal/modules/dashboard"
	"github.com/AliSleiman0/salehcard/api/internal/modules/expense"
	"github.com/AliSleiman0/salehcard/api/internal/modules/finance"
	"github.com/AliSleiman0/salehcard/api/internal/modules/kyc"
	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/modules/offer"
	"github.com/AliSleiman0/salehcard/api/internal/modules/order"
	product "github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/promo"
	"github.com/AliSleiman0/salehcard/api/internal/modules/reseller"
	"github.com/AliSleiman0/salehcard/api/internal/modules/review"
	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/blob"
	"github.com/AliSleiman0/salehcard/api/internal/platform/push"
	"github.com/AliSleiman0/salehcard/api/internal/platform/sms"
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

	// Serve the local blob adapter's files directly — only relevant in dev
	// (STORAGE_PROVIDER=local, the default); prod serves uploads from Azure Blob.
	if s.cfg.StorageProvider == "" || s.cfg.StorageProvider == "local" {
		fs := http.FileServer(http.Dir(s.cfg.UploadsDir))
		s.router.Handle("/uploads/*", http.StripPrefix("/uploads/", fs))
	}

	// Customer-facing routes (read-only product catalog stays separate). The
	// catalog is enriched with live-offer sale prices via a read-only offer repo
	// (EnsureIndexes runs in offer.RegisterRoutes, so this second repo skips it).
	offerCat := offerCatalog{repo: offer.NewMongoRepository(s.db.Collection("offers"))}
	product.RegisterRoutes(s.router, s.db, s.cfg, offerCat)

	// Read-only category taxonomy (storefront browses by root domain).
	category.RegisterRoutes(s.router, s.db)

	// Offers listing (storefront Offers tab; sale-price deals) — AuthRequired.
	offer.RegisterRoutes(s.router, s.db, s.cfg)

	// Customer auth + profile (public; /users/* guarded by AuthRequired).
	user.RegisterRoutes(s.router, s.db, s.cfg)

	// Customer-facing notifications are fanned out through the notifier: an
	// inbox row plus a best-effort push via the configured provider (log in dev).
	sender, err := push.New(push.Config{
		Provider: s.cfg.PushProvider,
		FCM: push.FCMConfig{
			CredentialsJSON: s.cfg.FCMCredentialsJSON,
			CredentialsFile: s.cfg.FCMCredentialsFile,
			ProjectID:       s.cfg.FCMProjectID,
		},
	})
	if err != nil {
		slog.Warn("server: push provider misconfigured — falling back to log sender", "provider", s.cfg.PushProvider, "error", err)
		sender = push.LogSender{}
	}
	ntf := notification.NewNotifier(s.db, sender)

	// Outbound SMS for admin bulk messaging. Built here (a second, admin-scoped
	// construction alongside the OTP sender in user.RegisterRoutes) and injected
	// into the admin user routes; falls open to the dev log sender on misconfig.
	// NOTE: in prod this is the live Monty provider — bulk-SMS sends real, paid
	// messages. The recipient cap (BulkSMSMax) + the admin confirm dialog guard it.
	smsSender, err := sms.New(sms.Config{
		Provider: s.cfg.SMSProvider,
		Monty: sms.MontyConfig{
			BaseURL:     s.cfg.MontyBaseURL,
			Username:    s.cfg.MontyUsername,
			APIID:       s.cfg.MontyAPIID,
			AccessToken: s.cfg.MontyAccessToken,
			SenderID:    s.cfg.MontySenderID,
			Campaign:    s.cfg.MontyCampaign,
		},
		Twilio: sms.TwilioConfig{
			AccountSID: s.cfg.TwilioAccountSID,
			AuthToken:  s.cfg.TwilioAuthToken,
			From:       s.cfg.TwilioFrom,
		},
	})
	if err != nil {
		slog.Warn("server: SMS provider misconfigured — falling back to log sender", "provider", s.cfg.SMSProvider, "error", err)
		smsSender = sms.LogSender{}
	}

	// Customer orders + wallet + promo validation + review submission +
	// notification inbox (guarded by AuthRequired).
	order.RegisterRoutes(s.router, s.db, s.cfg, ntf)
	wallet.RegisterRoutes(s.router, s.db, s.cfg)
	promo.RegisterRoutes(s.router, s.db, s.cfg)
	review.RegisterRoutes(s.router, s.db, s.cfg)
	kyc.RegisterRoutes(s.router, s.db, s.cfg)
	notification.RegisterRoutes(s.router, s.db, s.cfg)

	// Product image uploads (admin console editor). Falls open to the dev local
	// adapter on Azure misconfig, mirroring the sms/push fallback.
	store, err := blob.New(blob.Config{
		Provider: s.cfg.StorageProvider,
		Azure: blob.AzureConfig{
			ConnectionString: s.cfg.AzureStorageConnectionString,
			Container:        s.cfg.AzureStorageContainer,
		},
		Local: blob.LocalConfig{
			Dir:        s.cfg.UploadsDir,
			PublicBase: s.cfg.PublicBaseURL,
		},
	})
	if err != nil {
		slog.Warn("server: storage provider misconfigured — falling back to local adapter", "provider", s.cfg.StorageProvider, "error", err)
		store, _ = blob.New(blob.Config{Local: blob.LocalConfig{Dir: s.cfg.UploadsDir, PublicBase: s.cfg.PublicBaseURL}})
	}

	// Admin route group — every /api/admin/* route requires an `admin` JWT role
	// (AdminOnly bypasses only in development when no JWT secret is configured).
	// Mutating admin actions are recorded to the audit log via rec; customer-
	// visible outcomes (order/top-up/KYC decisions) also notify via ntf.
	rec := audit.NewRecorder(s.db)
	s.router.Route("/api/admin", func(r chi.Router) {
		r.Use(auth.AdminOnly(s.cfg.JWTSecret, s.cfg.Env == "development"))

		product.RegisterAdminRoutes(r, s.db, rec, store)
		code.RegisterAdminRoutes(r, s.db, rec, ntf)
		dashboard.RegisterAdminRoutes(r, s.db)
		finance.RegisterAdminRoutes(r, s.db)
		order.RegisterAdminRoutes(r, s.db, s.cfg, rec, ntf)
		user.RegisterAdminRoutes(r, s.db, rec, smsSender, s.cfg.BulkSMSMax)
		reseller.RegisterAdminRoutes(r, s.db, rec)
		promo.RegisterAdminRoutes(r, s.db)
		offer.RegisterAdminRoutes(r, s.db)
		review.RegisterAdminRoutes(r, s.db, rec)
		wallet.RegisterAdminRoutes(r, s.db, rec, ntf)
		expense.RegisterAdminRoutes(r, s.db)
		kyc.RegisterAdminRoutes(r, s.db, rec, ntf)
		audit.RegisterAdminRoutes(r, s.db)
		settings.RegisterAdminRoutes(r, s.db, rec, s.cfg.SMSProvider, s.cfg.PushProvider)
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
