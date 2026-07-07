package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
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

	// BulkSMSMax hard-caps the recipients an admin bulk-SMS send may target in one
	// request (the send goes over the live Monty provider — real, paid SMS — so
	// this is a spend guardrail). Exceeding it returns 400 BULK_SMS_LIMIT.
	BulkSMSMax int

	// IDCheckProvider selects the game-account ID-verification adapter: "rapidapi"
	// (RapidAPI ID Game Checker) or "stub" (dev — returns a placeholder username).
	// Default "stub".
	IDCheckProvider string
	// RapidAPIKey authenticates the RapidAPI ID Game Checker adapter. Required when
	// IDCheckProvider is "rapidapi"; server-side only — never shipped to clients.
	RapidAPIKey string
	// Rate limiting on the ID-verification endpoint (per-IP fixed window). Each call
	// hits a paid upstream API, so it is capped independently of the auth limiter.
	RateLimitVerifyMax    int
	RateLimitVerifyWindow time.Duration
	// Rate limiting on customer KYC document uploads (per-IP fixed window). Each
	// accepted upload writes billed blob storage.
	RateLimitKycUploadMax    int
	RateLimitKycUploadWindow time.Duration

	// Storage (product image uploads). StorageProvider selects the active blob
	// adapter: "azure" or "local" (default — dev works with zero Azure config).
	StorageProvider              string
	AzureStorageConnectionString string
	AzureStorageContainer        string
	// UploadsDir is where the local adapter writes files (dev only).
	UploadsDir string
	// PublicBaseURL prefixes URLs the local adapter returns (dev only).
	PublicBaseURL string

	// PaymentProvider selects the checkout payment gateway: "mock" enables the
	// sandbox card/usdt path; "" / "log" keeps checkout wallet-only (the default).
	PaymentProvider string
	// FulfillmentMock registers the reference upstream-fulfillment adapter under
	// FulfillmentMockID so api-mode orders routed to that provider id complete
	// (instead of parking). Off by default.
	FulfillmentMock   bool
	FulfillmentMockID int

	// On-chain USDT payments (payment module + platform/tron). The feature is
	// enabled iff USDTXPub is set; USDTProvider selects the chain reader:
	// "trongrid" (real chain) or "stub" (dev — auto-pays after USDTStubDelay).
	USDTProvider string
	// USDTXPub is the watch-only BIP44 account key (m/44'/195'/0') deposit
	// addresses derive from. The mnemonic/xprv never touches the server.
	USDTXPub string
	// USDTContract overrides the token contract (default mainnet USDT).
	USDTContract string
	// TronGridAPIKey raises the TronGrid rate limit (required in prod).
	TronGridAPIKey  string
	TronGridBaseURL string
	// USDTIntentExpiry is the customer's payment window (default 30m).
	USDTIntentExpiry time.Duration
	// USDTWatchInterval is the watcher tick (default 25s).
	USDTWatchInterval time.Duration
	// USDTLateGrace keeps scanning expired-unpaid addresses so late transfers
	// still credit the wallet (default 168h).
	USDTLateGrace time.Duration
	// USDTStubDelay is the stub reader's auto-pay delay (dev only).
	USDTStubDelay time.Duration
	// Rate limiting on payment-intent creation (burns HD addresses) and status
	// polling (the app polls every ~7s ≈ 9/min; 60 leaves headroom).
	RateLimitPaymentCreateMax    int
	RateLimitPaymentCreateWindow time.Duration
	RateLimitPaymentPollMax      int
	RateLimitPaymentPollWindow   time.Duration

	// Whish redirect payments (payment module + platform/whish). The feature is
	// enabled iff a real provider + credentials + the shared PAYMENTS_* callback
	// config are all set; WhishProvider selects the adapter: "whish" (real API)
	// or "stub" (dev — fake collectUrl, auto-success on re-poll).
	WhishProvider   string
	WhishBaseURL    string
	WhishChannel    string
	WhishSecret     string
	WhishWebsiteURL string
	WhishUserAgent  string
	// PaymentsWebhookBaseURL is the public host Whish reaches for server
	// callbacks (a dev tunnel; the prod API origin). Whish's sandbox cannot
	// reach localhost.
	PaymentsWebhookBaseURL string
	// PaymentsHMACSecret signs the tokens on Whish's unsigned callback URLs.
	PaymentsHMACSecret string
	// WhishSuccessRedirectURL / WhishFailureRedirectURL are where the browser
	// lands after the hosted page (optional for native clients).
	WhishSuccessRedirectURL string
	WhishFailureRedirectURL string
	// WhishIntentExpiry is the customer's Whish payment window (default 30m).
	WhishIntentExpiry time.Duration
	// WhishSweepInterval is the reconciliation-sweep tick (default 60s).
	WhishSweepInterval time.Duration
}

// whishConfigured reports whether Whish credentials + the shared callback config
// are present (independent of provider validity — see WhishEnabled at runtime).
func (c *Config) whishConfigured() bool {
	return c.WhishChannel != "" && c.WhishSecret != "" && c.WhishWebsiteURL != "" &&
		c.PaymentsWebhookBaseURL != "" && c.PaymentsHMACSecret != ""
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

		BulkSMSMax: getInt("BULK_SMS_MAX", 200),

		IDCheckProvider:       getEnv("IDCHECK_PROVIDER", "stub"),
		RapidAPIKey:           os.Getenv("RAPIDAPI_KEY"),
		RateLimitVerifyMax:    getInt("RATE_LIMIT_VERIFY_MAX", 20),
		RateLimitVerifyWindow: getDuration("RATE_LIMIT_VERIFY_WINDOW", time.Minute),

		RateLimitKycUploadMax:    getInt("RATE_LIMIT_KYC_UPLOAD_MAX", 10),
		RateLimitKycUploadWindow: getDuration("RATE_LIMIT_KYC_UPLOAD_WINDOW", time.Minute),

		StorageProvider:              getEnv("STORAGE_PROVIDER", "local"),
		AzureStorageConnectionString: os.Getenv("AZURE_STORAGE_CONNECTION_STRING"),
		AzureStorageContainer:        getEnv("AZURE_STORAGE_CONTAINER", "product-images"),
		UploadsDir:                   getEnv("UPLOADS_DIR", "./uploads"),
		PublicBaseURL:                getEnv("PUBLIC_BASE_URL", "http://localhost:8090"),

		PaymentProvider:   getEnv("PAYMENT_PROVIDER", "log"),
		FulfillmentMock:   getBool("FULFILLMENT_MOCK", false),
		FulfillmentMockID: getInt("FULFILLMENT_MOCK_ID", 1),

		USDTProvider:      getEnv("USDT_PROVIDER", "stub"),
		USDTXPub:          os.Getenv("USDT_XPUB"),
		USDTContract:      os.Getenv("USDT_CONTRACT"), // empty → platform/tron default
		TronGridAPIKey:    os.Getenv("TRONGRID_API_KEY"),
		TronGridBaseURL:   getEnv("TRONGRID_BASE_URL", "https://api.trongrid.io"),
		USDTIntentExpiry:  getDuration("USDT_INTENT_EXPIRY", 30*time.Minute),
		USDTWatchInterval: getDuration("USDT_WATCH_INTERVAL", 25*time.Second),
		USDTLateGrace:     getDuration("USDT_LATE_GRACE", 168*time.Hour),
		USDTStubDelay:     getDuration("USDT_STUB_DELAY", 15*time.Second),

		RateLimitPaymentCreateMax:    getInt("RATE_LIMIT_PAYMENT_CREATE_MAX", 10),
		RateLimitPaymentCreateWindow: getDuration("RATE_LIMIT_PAYMENT_CREATE_WINDOW", time.Minute),
		RateLimitPaymentPollMax:      getInt("RATE_LIMIT_PAYMENT_POLL_MAX", 60),
		RateLimitPaymentPollWindow:   getDuration("RATE_LIMIT_PAYMENT_POLL_WINDOW", time.Minute),

		WhishProvider:           getEnv("WHISH_PROVIDER", "stub"),
		WhishBaseURL:            os.Getenv("WHISH_BASE_URL"), // empty → platform/whish sandbox default
		WhishChannel:            os.Getenv("WHISH_CHANNEL"),
		WhishSecret:             os.Getenv("WHISH_SECRET"),
		WhishWebsiteURL:         os.Getenv("WHISH_WEBSITE_URL"),
		WhishUserAgent:          os.Getenv("WHISH_USER_AGENT"), // empty → platform/whish default
		PaymentsWebhookBaseURL:  os.Getenv("PAYMENTS_WEBHOOK_BASE_URL"),
		PaymentsHMACSecret:      os.Getenv("PAYMENTS_HMAC_SECRET"),
		WhishSuccessRedirectURL: os.Getenv("WHISH_SUCCESS_REDIRECT_URL"),
		WhishFailureRedirectURL: os.Getenv("WHISH_FAILURE_REDIRECT_URL"),
		WhishIntentExpiry:       getDuration("WHISH_INTENT_EXPIRY", 30*time.Minute),
		WhishSweepInterval:      getDuration("WHISH_SWEEP_INTERVAL", 60*time.Second),
	}
}

// getBool parses a boolean from the environment ("1"/"true"/"yes", case-
// insensitive), falling back to defaultVal when unset.
func getBool(key string, defaultVal bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return defaultVal
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
	if c.USDTXPub != "" && c.USDTProvider == "stub" {
		return errors.New("USDT_PROVIDER=stub would auto-confirm unpaid USDT payments outside development — set USDT_PROVIDER=trongrid or unset USDT_XPUB")
	}
	if c.USDTProvider == "trongrid" && c.USDTXPub != "" && c.TronGridAPIKey == "" {
		return errors.New("TRONGRID_API_KEY is required when USDT payments are enabled with the trongrid provider")
	}
	// Whish: the stub auto-confirms unpaid payments on re-poll, so it must never
	// run with real credentials outside development.
	if c.whishConfigured() && c.WhishProvider == "stub" {
		return errors.New("WHISH_PROVIDER=stub would auto-confirm unpaid Whish payments outside development — set WHISH_PROVIDER=whish or unset the WHISH_* credentials")
	}
	// A real Whish provider needs its credentials AND the shared callback config
	// (a public webhook host + an HMAC secret) or callbacks can't be received or
	// authenticated.
	if c.WhishProvider == "whish" && !c.whishConfigured() {
		return errors.New("Whish is set to the real provider but is missing credentials/callback config — set WHISH_CHANNEL, WHISH_SECRET, WHISH_WEBSITE_URL, PAYMENTS_WEBHOOK_BASE_URL, and PAYMENTS_HMAC_SECRET")
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
