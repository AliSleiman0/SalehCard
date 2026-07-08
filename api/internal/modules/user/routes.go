package user

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/modules/role"
	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
	"github.com/AliSleiman0/salehcard/api/internal/platform/ratelimit"
	"github.com/AliSleiman0/salehcard/api/internal/platform/sms"
)

// RegisterRoutes wires the customer-facing auth + profile routes onto r.
// These are public (outside the /api/admin group); /users/* is guarded by
// AuthRequired.
func RegisterRoutes(r chi.Router, db *mongo.Database, cfg *config.Config) {
	repo := NewMongoRepository(db)
	refreshRepo := NewMongoRefreshRepository(db)
	otpRepo := NewMongoOTPRepository(db)
	if err := EnsureIndexes(context.Background(), db); err != nil {
		slog.Warn("user: failed to ensure indexes", "error", err)
	}

	// Build the configured SMS adapter (monty | twilio | log). On a config error
	// fall back to the dev log sender so the API still boots.
	sender, err := sms.New(sms.Config{
		Provider: cfg.SMSProvider,
		Monty: sms.MontyConfig{
			BaseURL:     cfg.MontyBaseURL,
			Username:    cfg.MontyUsername,
			APIID:       cfg.MontyAPIID,
			AccessToken: cfg.MontyAccessToken,
			SenderID:    cfg.MontySenderID,
			Campaign:    cfg.MontyCampaign,
		},
		Twilio: sms.TwilioConfig{
			AccountSID: cfg.TwilioAccountSID,
			AuthToken:  cfg.TwilioAuthToken,
			From:       cfg.TwilioFrom,
		},
	})
	if err != nil {
		slog.Warn("user: SMS provider misconfigured — falling back to logging codes", "error", err)
		sender = sms.LogSender{}
	} else {
		slog.Info("user: OTP SMS sender ready", "provider", cfg.SMSProvider)
	}
	otpCfg := OTPConfig{
		CountryCode:    cfg.DefaultCountryCode,
		Length:         cfg.OTPLength,
		TTL:            cfg.OTPTTL,
		ResendInterval: cfg.OTPResendInterval,
		MaxAttempts:    cfg.OTPMaxAttempts,
	}

	// WithSettings enables the admin SMS-2FA gate (reads the app_settings singleton
	// at login time); without it admin login stays password-only. WithRolePerms
	// resolves a custom admin role's RBAC permission set into issued tokens.
	roleRepo := role.NewMongoRepository(db)
	svc := NewUserService(repo, refreshRepo, otpRepo, sender, otpCfg, cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL,
		WithSettings(settings.NewMongoRepository(db)),
		WithRolePerms(func(ctx context.Context, id bson.ObjectID) ([]string, error) {
			rl, err := roleRepo.FindByID(ctx, id)
			if err != nil {
				return nil, err
			}
			return rl.Permissions, nil
		}))
	h := NewHandler(svc, cfg.CookieSecure, cfg.RefreshTokenTTL)

	// Per-IP rate limiting on the public auth endpoints (RealIP upstream gives the
	// true client IP). A general cap on all auth calls + a tighter cap on OTP
	// requests, which cost SMS. Fails open on a limiter backend error.
	limiter := ratelimit.New(ratelimit.Config{Provider: cfg.RateLimitProvider}, db)
	if _, ok := limiter.(ratelimit.NoopLimiter); !ok {
		if err := ratelimit.EnsureIndexes(context.Background(), db); err != nil {
			slog.Warn("user: failed to ensure rate-limit indexes", "error", err)
		}
	}
	authLimit := ratelimit.Middleware(limiter, cfg.RateLimitAuthMax, cfg.RateLimitAuthWindow, ratelimit.ClientIP)
	otpLimit := ratelimit.Middleware(limiter, cfg.RateLimitOTPMax, cfg.RateLimitOTPWindow, func(req *http.Request) string {
		return "otp:" + ratelimit.ClientIP(req)
	})

	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Use(authLimit)
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/login-phone", h.LoginPhone)
		r.With(otpLimit).Post("/otp/request", h.RequestOTP)
		r.Post("/otp/verify", h.VerifyOTP)
		r.Post("/2fa/verify", h.VerifyTwoFactor)
		r.With(otpLimit).Post("/2fa/resend", h.ResendTwoFactor)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
	})

	r.Route("/api/v1/users", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Get("/me", h.GetProfile)
		r.Patch("/me", h.UpdateProfile)
	})
}
