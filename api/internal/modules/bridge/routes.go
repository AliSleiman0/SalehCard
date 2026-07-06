package bridge

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/platform/ratelimit"
)

// deviceRateMax/Window cap a single device's request rate (poll + report). A
// device polls every ~5s (~12/min) and reports a handful; 120/min is generous
// headroom while still bounding a misbehaving client.
const (
	deviceRateMax    = 120
	deviceRateWindow = 60 // seconds
)

// Registration bundles the constructed bridge pieces so the server can mount the
// device routes here, wire the admin routes, hand the service to the order
// module, and run the reaper — all sharing one store.
type Registration struct {
	Service *Service
	Store   *MongoStore
}

// RegisterRoutes builds the bridge service + store, mounts the device-facing
// endpoints under /api/v1/bridge (DeviceAuth + per-device rate limit), and
// returns the pieces for admin wiring and the reaper. The routes are always
// mounted so a provisioned device can reach config/poll; DispatchOrder and the
// reaper are gated on cfg.Bridge.Enabled by their callers.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config) *Registration {
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("bridge: failed to ensure indexes", "error", err)
	}
	store := NewMongoStore(db)
	svc := NewService(store, configFrom(cfg))
	h := NewHandler(svc)

	limiter := ratelimit.New(ratelimit.Config{Provider: cfg.RateLimitProvider}, db)
	if _, ok := limiter.(ratelimit.NoopLimiter); !ok {
		if err := ratelimit.EnsureIndexes(context.Background(), db); err != nil {
			slog.Warn("bridge: failed to ensure rate-limit indexes", "error", err)
		}
	}
	deviceLimit := ratelimit.Middleware(limiter, deviceRateMax, deviceRateWindow*1e9, func(req *http.Request) string {
		if d, ok := DeviceFromContext(req.Context()); ok {
			return "bridge:" + d.ID.Hex()
		}
		return "bridge:" + ratelimit.ClientIP(req)
	})

	r.Route("/api/v1/bridge", func(r chi.Router) {
		r.Use(DeviceAuth(store))
		r.Use(deviceLimit)
		r.Get("/config", h.GetConfig)
		r.Get("/commands", h.Poll)
		r.Post("/commands/{id}/result", h.PostResult)
		r.Post("/heartbeat", h.PostHeartbeat)
		r.Post("/logs", h.PostLogs)
	})

	return &Registration{Service: svc, Store: store}
}

// configFrom maps the app config into the bridge module's Config.
func configFrom(cfg *config.Config) Config {
	b := cfg.Bridge
	return Config{
		Enabled:           b.Enabled,
		Stub:              b.Stub,
		StubDelay:         b.StubDelay,
		LeaseTTL:          b.LeaseTTL,
		MaxAttempts:       b.MaxAttempts,
		QueueTimeout:      b.QueueTimeout,
		PollInterval:      b.PollInterval,
		HeartbeatInterval: b.HeartbeatInterval,
		MaxSMSPerHalfHour: b.MaxSMSPerHalfHour,
		SuccessPatterns:   b.SuccessPatterns,
		FailurePatterns:   b.FailurePatterns,
		Operators: OperatorConfig{
			TouchBalanceUSSD:      b.TouchBalanceUSSD,
			AlfaBalanceUSSD:       b.AlfaBalanceUSSD,
			TouchRechargeTemplate: b.TouchRechargeTemplate,
			TouchTransferTemplate: b.TouchTransferTemplate,
			TouchTransferDest:     b.TouchTransferDest,
			AlfaRechargeTemplate:  b.AlfaRechargeTemplate,
			AlfaRechargeDest:      b.AlfaRechargeDest,
			AlfaTransferTemplate:  b.AlfaTransferTemplate,
			AlfaTransferDest:      b.AlfaTransferDest,
			TouchMinBalance:       b.TouchMinBalance,
			AlfaMinBalance:        b.AlfaMinBalance,
			TouchMessageFee:       b.TouchMessageFee,
			AlfaMessageFee:        b.AlfaMessageFee,
		},
	}
}
