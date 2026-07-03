package config

import (
	"errors"
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

	// SMSProvider selects the active OTP SMS adapter: "monty" (Lebanon),
	// "twilio" (international), or "log" (dev — logs the code). Default "log".
	SMSProvider string

	// Monty Mobile (Lebanon) credentials. Required when SMSProvider is "monty".
	MontyBaseURL     string // default https://sms.montymobile.com
	MontyUsername    string
	MontyAPIID       string // apiId query parameter
	MontyAccessToken string // X-Access-Token header
	MontySenderID    string // registered alphanumeric Source (e.g. "SalehCard")
	MontyCampaign    string // optional campaignname; defaults to username

	// Twilio credentials. Required when SMSProvider is "twilio".
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFrom       string // sender number (E.164) or Messaging Service SID

	// PushProvider selects the active push-notification adapter: "fcm" or "log"
	// (dev — logs the push). Default "log". Credentials are validated lazily
	// inside push.New, mirroring the SMS layer.
	PushProvider string

	// Firebase Cloud Messaging (HTTP v1) service-account credentials. Required
	// when PushProvider is "fcm" — either the inline JSON (Azure app setting) or
	// a file path (local dev). ProjectID is optional (derived from the account).
	FCMCredentialsJSON string
	FCMCredentialsFile string
	FCMProjectID       string

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

	// Rate limiting on the public auth endpoints (per-IP fixed window).
	// RateLimitProvider is "mongo" (default; a TTL-counter collection) or
	// "noop"/"off" (disabled). The Auth pair caps all auth calls; the tighter OTP
	// pair caps OTP requests (which cost SMS).
	RateLimitProvider   string
	RateLimitAuthMax    int
	RateLimitAuthWindow time.Duration
	RateLimitOTPMax     int
	RateLimitOTPWindow  time.Duration

	// Email provider for bulk/transactional email. EmailProvider is "smtp",
	// "sendgrid", or "log" (default; logs instead of sending). EmailFrom is the
	// shared sender address for both adapters.
	EmailProvider  string
	EmailFrom      string
	SMTPHost       string
	SMTPPort       int
	SMTPUsername   string
	SMTPPassword   string
	SendGridAPIKey string
}

// Load reads configuration from environment variables, applying defaults where
// needed. ENV defaults to "production" so a deployment that forgets to set it
// fails closed (strict auth, Validate enforced) rather than silently running
// with development affordances; local dev sets ENV=development in api/.env.
func Load() *Config {
	env := getEnv("ENV", "production")
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

		SMSProvider: getEnv("SMS_PROVIDER", "log"),

		MontyBaseURL:     getEnv("MONTY_BASE_URL", "https://sms.montymobile.com"),
		MontyUsername:    os.Getenv("MONTY_USERNAME"),
		MontyAPIID:       os.Getenv("MONTY_API_ID"),
		MontyAccessToken: os.Getenv("MONTY_ACCESS_TOKEN"),
		MontySenderID:    os.Getenv("MONTY_SENDER_ID"),
		MontyCampaign:    os.Getenv("MONTY_CAMPAIGN"),

		TwilioAccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioFrom:       os.Getenv("TWILIO_FROM"),

		PushProvider:       getEnv("PUSH_PROVIDER", "log"),
		FCMCredentialsJSON: os.Getenv("FCM_CREDENTIALS_JSON"),
		FCMCredentialsFile: os.Getenv("FCM_CREDENTIALS_FILE"),
		FCMProjectID:       os.Getenv("FCM_PROJECT_ID"),

		DefaultCountryCode: getEnv("DEFAULT_COUNTRY_CODE", "+961"),
		OTPLength:          getInt("OTP_LENGTH", 6),
		OTPTTL:             getDuration("OTP_TTL", 5*time.Minute),
		OTPResendInterval:  getDuration("OTP_RESEND_INTERVAL", 60*time.Second),
		OTPMaxAttempts:     getInt("OTP_MAX_ATTEMPTS", 5),

		RateLimitProvider:   getEnv("RATE_LIMIT_PROVIDER", "mongo"),
		RateLimitAuthMax:    getInt("RATE_LIMIT_AUTH_MAX", 30),
		RateLimitAuthWindow: getDuration("RATE_LIMIT_AUTH_WINDOW", time.Minute),
		RateLimitOTPMax:     getInt("RATE_LIMIT_OTP_MAX", 5),
		RateLimitOTPWindow:  getDuration("RATE_LIMIT_OTP_WINDOW", time.Minute),

		EmailProvider:  getEnv("EMAIL_PROVIDER", "log"),
		EmailFrom:      os.Getenv("EMAIL_FROM"),
		SMTPHost:       os.Getenv("SMTP_HOST"),
		SMTPPort:       getInt("SMTP_PORT", 587),
		SMTPUsername:   os.Getenv("SMTP_USERNAME"),
		SMTPPassword:   os.Getenv("SMTP_PASSWORD"),
		SendGridAPIKey: os.Getenv("SENDGRID_API_KEY"),
	}
}

// Validate rejects configurations that would run an unsafe server outside
// development: an empty JWT_SECRET would disable admin auth entirely, and an
// empty ALLOWED_ORIGINS leaves CORS misconfigured for the deployed SPAs.
func (c *Config) Validate() error {
	if c.Env == "development" {
		return nil
	}
	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET is required when ENV is not development (admin auth would be disabled)")
	}
	if c.AllowedOrigins == "" {
		return errors.New("ALLOWED_ORIGINS is required when ENV is not development")
	}
	return nil
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
