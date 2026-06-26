package config

import (
	"os"
	"strconv"
	"time"
)

// Default token lifetimes used when the corresponding env vars are unset or invalid.
const (
	defaultAccessTokenTTL  = 15 * time.Minute
	defaultRefreshTokenTTL = 720 * time.Hour // 30 days
)

// Config holds all application configuration values.
type Config struct {
	MongoURI       string
	DBName         string
	Port           string
	JWTSecret      string
	GoogleClientID string
	AllowedOrigins string
	Env            string

	// AccessTokenTTL is the lifetime of issued access JWTs.
	AccessTokenTTL time.Duration
	// RefreshTokenTTL is the lifetime of issued refresh tokens.
	RefreshTokenTTL time.Duration
	// CookieSecure marks the refresh-token cookie Secure outside development.
	CookieSecure bool

	// Twilio direct-send credentials. When all three are set, OTP SMS is sent
	// straight to the Twilio Messages API (preferred — no gateway/DB needed).
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFrom       string // sender number (E.164) or Messaging Service SID

	// SMSGatewayBaseURL is the base URL of the mip SMS gateway (e.g.
	// http://localhost:8081/SMS_GATEWAY_API). Used only when Twilio is not
	// configured. When neither is set, OTP codes are logged (dev fallback).
	SMSGatewayBaseURL string
	// SMSGatewayProvider selects the gateway's downstream provider ("twilio").
	SMSGatewayProvider string
	// SMSGatewayToken is an optional Bearer token for the gateway's /api/sms/send
	// (only needed when the gateway's IP/JWT security is enabled).
	SMSGatewayToken string

	// DefaultCountryCode is prefixed to local phone numbers entered without an
	// international prefix (e.g. "+961" for Lebanon).
	DefaultCountryCode string
	// OTPLength is the number of digits in a generated OTP code.
	OTPLength int
	// OTPTTL is how long a requested OTP stays valid.
	OTPTTL time.Duration
	// OTPResendInterval is the minimum wait between OTP requests for one number.
	OTPResendInterval time.Duration
	// OTPMaxAttempts is the number of wrong guesses allowed before a code locks.
	OTPMaxAttempts int
}

// Load reads configuration from environment variables, applying defaults where needed.
func Load() *Config {
	env := getEnv("ENV", "development")
	return &Config{
		MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:         getEnv("DB_NAME", "salehcard"),
		Port:           getEnv("PORT", "8080"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),
		AllowedOrigins: os.Getenv("ALLOWED_ORIGINS"),
		Env:            env,

		AccessTokenTTL:  getDuration("ACCESS_TOKEN_TTL", defaultAccessTokenTTL),
		RefreshTokenTTL: getDuration("REFRESH_TOKEN_TTL", defaultRefreshTokenTTL),
		CookieSecure:    env != "development",

		TwilioAccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioFrom:       os.Getenv("TWILIO_FROM"),

		SMSGatewayBaseURL:  os.Getenv("SMS_GATEWAY_BASE_URL"),
		SMSGatewayProvider: getEnv("SMS_GATEWAY_PROVIDER", "twilio"),
		SMSGatewayToken:    os.Getenv("SMS_GATEWAY_TOKEN"),

		DefaultCountryCode: getEnv("DEFAULT_COUNTRY_CODE", "+961"),
		OTPLength:          getInt("OTP_LENGTH", 6),
		OTPTTL:             getDuration("OTP_TTL", 5*time.Minute),
		OTPResendInterval:  getDuration("OTP_RESEND_INTERVAL", 60*time.Second),
		OTPMaxAttempts:     getInt("OTP_MAX_ATTEMPTS", 5),
	}
}

// getInt parses an integer from the environment, falling back to defaultVal when
// unset or unparseable.
func getInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// getDuration parses a Go duration string (e.g. "15m", "720h") from the
// environment, falling back to defaultVal when unset or unparseable.
func getDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}
