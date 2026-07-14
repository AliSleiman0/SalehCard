package server

import (
	"context"
	"log"
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
	"github.com/AliSleiman0/salehcard/api/internal/modules/bridge"
	"github.com/AliSleiman0/salehcard/api/internal/modules/category"
	"github.com/AliSleiman0/salehcard/api/internal/modules/code"
	"github.com/AliSleiman0/salehcard/api/internal/modules/dashboard"
	"github.com/AliSleiman0/salehcard/api/internal/modules/expense"
	"github.com/AliSleiman0/salehcard/api/internal/modules/finance"
	"github.com/AliSleiman0/salehcard/api/internal/modules/kyc"
	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/modules/offer"
	"github.com/AliSleiman0/salehcard/api/internal/modules/order"
	"github.com/AliSleiman0/salehcard/api/internal/modules/payment"
	product "github.com/AliSleiman0/salehcard/api/internal/modules/product"
	"github.com/AliSleiman0/salehcard/api/internal/modules/promo"
	"github.com/AliSleiman0/salehcard/api/internal/modules/reseller"
	"github.com/AliSleiman0/salehcard/api/internal/modules/review"
	"github.com/AliSleiman0/salehcard/api/internal/modules/role"
	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
	"github.com/AliSleiman0/salehcard/api/internal/modules/user"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/blob"
	"github.com/AliSleiman0/salehcard/api/internal/platform/push"
	"github.com/AliSleiman0/salehcard/api/internal/platform/sms"
	"github.com/AliSleiman0/salehcard/api/internal/platform/bsc"
	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

const pingTimeout = 5 * time.Second

// Server holds the HTTP router, database connection, and application config.
type Server struct {
	router *chi.Mux
	db     *mongo.Database
	cfg    *config.Config
	// watcher is the on-chain USDT payment watcher (nil when USDT payments
	// are not enabled); main.go runs it alongside the HTTP listener.
	watcher *payment.Watcher
	// bridgeReaper requeues stale mobile-bridge command leases (nil when bridge
	// fulfillment is disabled); main.go runs it alongside the HTTP listener.
	bridgeReaper *bridge.Reaper
}

// Watcher returns the USDT payment watcher, or nil when the feature is off.
func (s *Server) Watcher() *payment.Watcher { return s.watcher }

// BridgeReaper returns the mobile-bridge reaper, or nil when bridge is disabled.
func (s *Server) BridgeReaper() *bridge.Reaper { return s.bridgeReaper }

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

	// Public legal pages (Play Store listing + in-app links). Bilingual static
	// HTML with the admin-configurable support email injected per request.
	pages := newLegalPages(settings.NewMongoRepository(s.db))
	s.router.Get("/privacy", pages.privacy)
	s.router.Get("/delete-account", pages.deleteAccount)

	// Blob storage for product images (POST /api/admin/products/images) and
	// customer KYC document photos (POST /api/v1/kyc/documents).
	// Falls open to the dev local adapter on Azure misconfig, mirroring the
	// sms/push fallback — constructed directly (NewLocal is infallible) so a
	// failed Azure setup logs a warning instead of leaving a nil store.
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
		store = blob.NewLocal(blob.LocalConfig{Dir: s.cfg.UploadsDir, PublicBase: s.cfg.PublicBaseURL})
	}

	// Serve the local adapter's files directly. Gate on the CONSTRUCTED adapter,
	// not the config: a fallback-to-local (Azure misconfig) still needs its
	// /uploads mount, and a real Azure adapter must not expose local disk.
	if _, ok := store.(*blob.LocalStorage); ok {
		fs := http.FileServer(http.Dir(s.cfg.UploadsDir))
		s.router.Handle("/uploads/*", http.StripPrefix("/uploads/", fs))
	}

	// Customer-facing routes (read-only product catalog stays separate). The
	// catalog is enriched with live-offer sale prices via a read-only offer repo
	// (EnsureIndexes runs in offer.RegisterRoutes, so this second repo skips it).
	offerCat := offerCatalog{repo: offer.NewMongoRepository(s.db.Collection("offers"))}
	// The category repository doubles as the product module's CategoryResolver
	// (tree-aware ?categoryId= filtering + taxonomy denormalization on write).
	catResolver := category.NewMongoRepository(s.db)
	product.RegisterRoutes(s.router, s.db, s.cfg, offerCat, catResolver)

	// Read-only category taxonomy (storefront browses the tree: root domains,
	// subcategories via ?parentId=, product counts).
	category.RegisterRoutes(s.router, s.db)

	// Offers listing (storefront Offers tab; sale-price deals) — AuthRequired.
	offer.RegisterRoutes(s.router, s.db, s.cfg)

	// Admin audit-log recorder. Constructed here (rather than with the admin
	// group below) because customer self-deletion also records to it.
	rec := audit.NewRecorder(s.db)

	// Customer auth + profile (public; /users/* guarded by AuthRequired).
	// Account deletion (DELETE /users/me) reaches across modules through
	// narrow ports: guard counts from order/payment, the KYC purge (which also
	// deletes document blobs from the shared store), and device-token cleanup.
	// Second repo constructions, like offerCat above — EnsureIndexes still runs
	// in each module's own RegisterRoutes.
	user.RegisterRoutes(s.router, s.db, s.cfg, rec, user.DeletionPorts{
		Orders:   order.NewMongoRepository(s.db.Collection("orders")),
		Payments: payment.NewMongoStore(s.db),
		TopUps:   wallet.NewTopUpRepo(s.db),
		KYC:      kyc.NewPurger(kyc.NewMongoRepository(s.db.Collection("kyc_submissions"), s.db.Collection("users")), store),
		Tokens:   notification.NewMongoRepository(s.db.Collection("notifications"), s.db.Collection("device_tokens")),
	})

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

	// On-chain USDT payments. The routes are always mounted so the client
	// feature gate (GET /payments/config) answers even when the feature is
	// off (no USDT_XPUB → Enabled()==false → intent creation refuses); the
	// background watcher runs only when it is on.
	paySvc := s.buildPaymentService(ntf)
	payment.RegisterRoutes(s.router, s.db, s.cfg, paySvc)
	if paySvc.Enabled() {
		s.watcher = payment.NewWatcher(paySvc, s.cfg.USDTWatchInterval)
	}

	// Mobile Bridge (Lebanese recharge automation). Device routes are always
	// mounted so a provisioned phone can reach config/poll; DispatchOrder and the
	// reaper are gated on BRIDGE_ENABLED. The order service is handed to the
	// bridge as its OrderSettler (the device-result completion callback).
	bridgeReg := bridge.RegisterRoutes(s.router, s.db, s.cfg)

	// Customer orders + wallet + promo validation + review submission +
	// notification inbox (guarded by AuthRequired). Order takes the payment
	// service as its USDT-intents port and the bridge service as its recharge
	// dispatcher, and we close the loop by handing the order service back to the
	// payment module (OrderSettler for confirmed on-chain payments) and the bridge
	// module (OrderSettler for confirmed device recharges).
	orderSvc := order.RegisterRoutes(s.router, s.db, s.cfg, ntf, paySvc, bridgeReg.Service)
	paySvc.SetOrderSettler(orderSvc)
	bridgeReg.Service.SetOrderSettler(orderSvc)
	if bridgeReg.Service.Enabled() {
		s.bridgeReaper = bridge.NewReaper(bridgeReg.Service, s.cfg.Bridge.ReaperInterval)
	}
	wallet.RegisterRoutes(s.router, s.db, s.cfg)
	promo.RegisterRoutes(s.router, s.db, s.cfg)
	review.RegisterRoutes(s.router, s.db, s.cfg)
	kyc.RegisterRoutes(s.router, s.db, s.cfg, store)
	notification.RegisterRoutes(s.router, s.db, s.cfg)

	// Admin route group — every /api/admin/* route requires an `admin` JWT role
	// (AdminOnly bypasses only in development when no JWT secret is configured).
	// Mutating admin actions are recorded to the audit log via rec (constructed
	// above, shared with customer self-deletion); customer-visible outcomes
	// (order/top-up/KYC decisions) also notify via ntf.
	s.router.Route("/api/admin", func(r chi.Router) {
		r.Use(auth.AdminOnly(s.cfg.JWTSecret, s.cfg.Env == "development"))

		// RBAC: each module is wrapped in its permission domain — GET/HEAD need
		// "<domain>.view", mutations need "<domain>.manage" (super admins carry
		// the "*" wildcard and pass everything). The domain key MUST exist in the
		// catalog (modules/role/permissions.go): the helper fail-fasts at boot on
		// a typo, otherwise the module would sit behind a permission string that
		// no custom role can ever be granted (silent 403 for every limited admin).
		wrapped := map[string]bool{}
		domain := func(name string, register func(chi.Router)) {
			if !role.ValidPermission(name + ".view") {
				log.Fatalf("server: admin domain %q is not in the RBAC permission catalog (modules/role/permissions.go)", name)
			}
			wrapped[name] = true
			r.Group(func(g chi.Router) {
				g.Use(auth.RequireDomain(name))
				register(g)
			})
		}
		defer func() {
			// Reverse check: every catalog domain must have a module mounted behind
			// it, or a grantable permission would gate nothing (a role could be
			// given e.g. payments.manage with no payments routes to authorize).
			for _, d := range role.Domains {
				if !wrapped[d.Key] {
					log.Fatalf("server: RBAC catalog domain %q has no admin routes mounted (modules/role/permissions.go vs server.go)", d.Key)
				}
			}
		}()

		domain("products", func(g chi.Router) { product.RegisterAdminRoutes(g, s.db, rec, store, catResolver) })
		domain("categories", func(g chi.Router) { category.RegisterAdminRoutes(g, s.db, rec) })
		domain("inventory", func(g chi.Router) { code.RegisterAdminRoutes(g, s.db, rec) })
		domain("dashboard", func(g chi.Router) { dashboard.RegisterAdminRoutes(g, s.db) })
		domain("finance", func(g chi.Router) { finance.RegisterAdminRoutes(g, s.db) })
		domain("orders", func(g chi.Router) {
			order.RegisterAdminRoutes(g, s.db, s.cfg, rec, ntf)
			// resend-code is an order operation (keyed by order id) though its
			// handler lives in the code module — mount it here so it requires
			// orders.manage, not inventory.manage.
			code.RegisterOrderResendRoutes(g, s.db, rec, ntf)
		})
		domain("users", func(g chi.Router) {
			user.RegisterAdminRoutes(g, s.db, rec, smsSender, s.cfg.BulkSMSMax, ntf, s.cfg.BulkPushMax)
		})
		domain("resellers", func(g chi.Router) { reseller.RegisterAdminRoutes(g, s.db, rec) })
		domain("promos", func(g chi.Router) { promo.RegisterAdminRoutes(g, s.db) })
		domain("offers", func(g chi.Router) { offer.RegisterAdminRoutes(g, s.db) })
		domain("reviews", func(g chi.Router) { review.RegisterAdminRoutes(g, s.db, rec) })
		domain("topups", func(g chi.Router) { wallet.RegisterAdminRoutes(g, s.db, rec, ntf) })
		domain("expenses", func(g chi.Router) { expense.RegisterAdminRoutes(g, s.db) })
		domain("kyc", func(g chi.Router) { kyc.RegisterAdminRoutes(g, s.db, rec, ntf) })
		domain("audit", func(g chi.Router) { audit.RegisterAdminRoutes(g, s.db) })
		domain("settings", func(g chi.Router) { settings.RegisterAdminRoutes(g, s.db, rec, s.cfg.SMSProvider, s.cfg.PushProvider) })
		domain("payments", func(g chi.Router) { payment.RegisterAdminRoutes(g, s.db, paySvc, rec) })
		domain("bridge", func(g chi.Router) { bridge.RegisterAdminRoutes(g, bridgeReg.Service, bridgeReg.Store, rec) })

		// Role management mounts outside the domain wrapper: listing roles + the
		// permission catalog is open to every admin (the console needs names to
		// render assignments), while mutations are super-admin-only internally.
		role.RegisterAdminRoutes(r, s.db, rec)
	})
}

// buildPaymentService constructs the USDT payment service. Misconfiguration
// (bad provider name, invalid xpub, malformed shared address) logs and
// degrades to a disabled service — the store still serves reads, but no new
// intents can be created and no watcher runs. config.Validate separately
// refuses stub+enabled outside dev and both-modes-set in every env.
func (s *Server) buildPaymentService(ntf notification.Notifier) *payment.Service {
	xpub := s.cfg.USDTXPub
	shared := s.cfg.USDTAddress
	reader, err := tron.New(tron.Config{
		Provider: s.cfg.USDTProvider,
		TronGrid: tron.TronGridConfig{
			BaseURL:  s.cfg.TronGridBaseURL,
			APIKey:   s.cfg.TronGridAPIKey,
			Contract: s.cfg.USDTContract,
		},
		Stub: tron.StubConfig{Delay: s.cfg.USDTStubDelay},
	})
	if err != nil {
		slog.Error("payment: chain provider misconfigured — USDT payments disabled", "provider", s.cfg.USDTProvider, "error", err)
		reader, xpub, shared = nil, "", ""
	}
	// Startup smoke-checks: a bad key/address should surface in the boot log,
	// not on the first customer intent (a typo'd shared address would send
	// customer funds somewhere unrecoverable).
	if xpub != "" {
		if addr, derr := tron.DeriveAddress(xpub, 0); derr != nil {
			slog.Error("payment: USDT_XPUB is invalid — USDT payments disabled", "error", derr)
			xpub = ""
		} else {
			slog.Info("payment: on-chain USDT payments enabled (derived-address mode)",
				"provider", s.cfg.USDTProvider, "network", payment.NetworkTRC20, "address0", addr)
		}
	}
	if shared != "" {
		if verr := tron.ValidateAddress(shared); verr != nil {
			slog.Error("payment: USDT_ADDRESS is invalid — USDT payments disabled", "error", verr)
			shared = ""
		} else {
			slog.Info("payment: on-chain USDT payments enabled (shared-address mode)",
				"provider", s.cfg.USDTProvider, "network", payment.NetworkTRC20, "address", shared)
		}
	}
	// BEP20 second network (always shared mode). Misconfiguration disables
	// BEP20 only — TRC20 keeps running.
	var bep20 tron.TransferLister
	bep20Addr := s.cfg.USDTBEP20Address
	if bep20Addr != "" {
		lister, berr := bsc.New(bsc.Config{
			Provider: s.cfg.USDTBEP20Provider,
			JSONRPC: bsc.JSONRPCConfig{
				Endpoints:        s.cfg.BSCRPCEndpoints,
				Contract:         s.cfg.USDTBEP20Contract,
				MinConfirmations: int64(s.cfg.USDTBEP20MinConfirmations),
			},
			Etherscan: bsc.EtherscanConfig{
				BaseURL:          s.cfg.EtherscanBaseURL,
				APIKey:           s.cfg.EtherscanAPIKey,
				Contract:         s.cfg.USDTBEP20Contract,
				MinConfirmations: int64(s.cfg.USDTBEP20MinConfirmations),
			},
			Stub: tron.StubConfig{Delay: s.cfg.USDTStubDelay},
		})
		verr := bsc.ValidateAddress(bep20Addr)
		switch {
		case berr != nil:
			slog.Error("payment: BEP20 chain provider misconfigured — BEP20 USDT payments disabled",
				"provider", s.cfg.USDTBEP20Provider, "error", berr)
			bep20Addr = ""
		case verr != nil:
			slog.Error("payment: USDT_BEP20_ADDRESS is invalid — BEP20 USDT payments disabled", "error", verr)
			bep20Addr = ""
		default:
			bep20 = lister
			slog.Info("payment: on-chain USDT payments enabled (shared-address mode)",
				"provider", s.cfg.USDTBEP20Provider, "network", payment.NetworkBEP20, "address", bep20Addr)
		}
	}
	wsvc := wallet.NewWalletService(wallet.NewMongoRepository(s.db), nil)
	return payment.NewService(payment.NewMongoStore(s.db), reader, bep20, wsvc, ntf, payment.Config{
		XPub:               xpub,
		SharedAddress:      shared,
		BEP20SharedAddress: bep20Addr,
		IntentExpiry:       s.cfg.USDTIntentExpiry,
		LateGrace:          s.cfg.USDTLateGrace,
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
