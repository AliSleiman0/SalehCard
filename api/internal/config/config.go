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
	defaultAccessTokenTTL  = 90 * 24 * time.Hour // 90 days
	defaultRefreshTokenTTL = 90 * 24 * time.Hour // 90 days
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

	// Bridge (Lebanese mobile-recharge automation via the Android bridge device).
	// Off by default: until BRIDGE_ENABLED, recharge orders park for manual
	// completion exactly as before. See internal/modules/bridge.
	Bridge BridgeConfig
}

// BridgeConfig groups the mobile-bridge tunables. The control knobs govern the
// command lease/reaper lifecycle; the operator block carries the per-network
// SMS/USSD templates, shortcodes, and fees the device dials with — all
// env-overridable so a live-SIM calibration never needs an APK rebuild.
type BridgeConfig struct {
	Enabled           bool
	Stub              bool // dev-only auto-succeed; Validate rejects outside development
	StubDelay         time.Duration
	LeaseTTL          time.Duration
	MaxAttempts       int
	QueueTimeout      time.Duration
	ReaperInterval    time.Duration
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
	MaxSMSPerHalfHour int
	SuccessPatterns   []string
	FailurePatterns   []string

	TouchBalanceUSSD      string
	AlfaBalanceUSSD       string
	TouchRechargeTemplate string
	TouchTransferTemplate string
	TouchTransferDest     string
	AlfaRechargeTemplate  string
	AlfaRechargeDest      string
	AlfaTransferTemplate  string
	AlfaTransferDest      string
	TouchMinBalance       float64
	AlfaMinBalance        float64
	TouchMessageFee       float64
	AlfaMessageFee        float64
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

		Bridge: BridgeConfig{
			Enabled:           getBool("BRIDGE_ENABLED", false),
			Stub:              getBool("BRIDGE_STUB", false),
			StubDelay:         getDuration("BRIDGE_STUB_DELAY", 10*time.Second),
			LeaseTTL:          getDuration("BRIDGE_LEASE_TTL", 3*time.Minute),
			MaxAttempts:       getInt("BRIDGE_MAX_ATTEMPTS", 3),
			QueueTimeout:      getDuration("BRIDGE_QUEUE_TIMEOUT", 30*time.Minute),
			ReaperInterval:    getDuration("BRIDGE_REAPER_INTERVAL", 30*time.Second),
			PollInterval:      getDuration("BRIDGE_POLL_INTERVAL", 5*time.Second),
			HeartbeatInterval: getDuration("BRIDGE_HEARTBEAT_INTERVAL", 5*time.Minute),
			MaxSMSPerHalfHour: getInt("BRIDGE_MAX_SMS_PER_HALF_HOUR", 25),
			SuccessPatterns:   getCSV("BRIDGE_SUCCESS_PATTERNS", []string{"success", "transferred"}),
			FailurePatterns:   getCSV("BRIDGE_FAILURE_PATTERNS", []string{"fail", "do not have", "insufficient"}),

			TouchBalanceUSSD:      getEnv("BRIDGE_TOUCH_BALANCE_USSD", "*220#"),
			AlfaBalanceUSSD:       getEnv("BRIDGE_ALFA_BALANCE_USSD", "*11#"),
			TouchRechargeTemplate: getEnv("BRIDGE_TOUCH_RECHARGE_TEMPLATE", "*300*{phone}#{card}"),
			TouchTransferTemplate: getEnv("BRIDGE_TOUCH_TRANSFER_TEMPLATE", "{phone}T{amount}"),
			TouchTransferDest:     getEnv("BRIDGE_TOUCH_TRANSFER_DEST", "1199"),
			AlfaRechargeTemplate:  getEnv("BRIDGE_ALFA_RECHARGE_TEMPLATE", "{phone}R{code}"),
			AlfaRechargeDest:      getEnv("BRIDGE_ALFA_RECHARGE_DEST", "1313"),
			AlfaTransferTemplate:  getEnv("BRIDGE_ALFA_TRANSFER_TEMPLATE", "{phone}T{amount}"),
			AlfaTransferDest:      getEnv("BRIDGE_ALFA_TRANSFER_DEST", "1399"),
			TouchMinBalance:       getFloat("BRIDGE_TOUCH_MIN_BALANCE", 20),
			AlfaMinBalance:        getFloat("BRIDGE_ALFA_MIN_BALANCE", 20),
			TouchMessageFee:       getFloat("BRIDGE_TOUCH_MESSAGE_FEE", 0.16),
			AlfaMessageFee:        getFloat("BRIDGE_ALFA_MESSAGE_FEE", 0.14),
		},
	}
}

// getFloat parses a float from the environment, falling back to defaultVal when
// unset or unparseable.
func getFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

// getCSV parses a comma-separated list from the environment (trimming blanks),
// falling back to defaultVal when unset.
func getCSV(key string, defaultVal []string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultVal
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return defaultVal
	}
	return out
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
	if c.Bridge.Stub {
		return errors.New("BRIDGE_STUB auto-completes recharge orders without a real device — unset it outside development")
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
