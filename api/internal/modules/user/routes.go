package user

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	"github.com/AliSleiman0/salehcard/api/internal/platform/auth"
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

	// Pick the SMS sender: Twilio direct-send when configured, else the mip
	// gateway, else a dev fallback that logs the code (no Twilio/gateway needed).
	var sender sms.Sender
	switch {
	case cfg.TwilioAccountSID != "" && cfg.TwilioAuthToken != "" && cfg.TwilioFrom != "":
		sender = sms.NewTwilioSender(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFrom)
		slog.Info("user: sending OTP SMS via Twilio")
	case cfg.SMSGatewayBaseURL != "":
		sender = sms.NewGatewayClient(cfg.SMSGatewayBaseURL, cfg.SMSGatewayProvider, cfg.SMSGatewayToken)
		slog.Info("user: sending OTP SMS via the mip gateway", "url", cfg.SMSGatewayBaseURL)
	default:
		slog.Warn("user: no SMS sender configured — OTP codes will be logged, not sent")
		sender = sms.LogSender{}
	}
	otpCfg := OTPConfig{
		CountryCode:    cfg.DefaultCountryCode,
		Length:         cfg.OTPLength,
		TTL:            cfg.OTPTTL,
		ResendInterval: cfg.OTPResendInterval,
		MaxAttempts:    cfg.OTPMaxAttempts,
	}

	svc := NewUserService(repo, refreshRepo, otpRepo, sender, otpCfg, cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	h := NewHandler(svc, cfg.CookieSecure)

	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/login-phone", h.LoginPhone)
		r.Post("/otp/request", h.RequestOTP)
		r.Post("/otp/verify", h.VerifyOTP)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
	})

	r.Route("/api/v1/users", func(r chi.Router) {
		r.Use(auth.AuthRequired(cfg.JWTSecret))
		r.Get("/me", h.GetProfile)
		r.Patch("/me", h.UpdateProfile)
	})
}
