package config

import (
	"errors"
	"fmt"
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

	// BulkPushMax hard-caps the recipients an admin bulk-push send may target in
	// one request. Push is free (FCM), so this is only an abuse/rate guard —
	// hence a higher default than BulkSMSMax. Exceeding it returns
	// 400 BULK_PUSH_LIMIT.
	BulkPushMax int

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
	// Suppliers are the upstream panel suppliers api-mode orders can dispatch
	// to (DESIGN-SUPPLIERS.md). A supplier is ENABLED only when its token is
	// set; otherwise its id resolves to the parking stub — today's behavior.
	Suppliers []SupplierConfig
	// Supplier settler (DESIGN-SUPPLIERS.md Phase 2): tick interval for
	// reconciling parked api-mode orders, and the give-up window after which
	// a still-waiting or retry-exhausted order is flagged stuck (never
	// auto-refunded). The re-dispatch backoff schedule itself is a constant
	// in the order module.
	SupplierSettlerInterval time.Duration
	SupplierSettlerGiveUp   time.Duration

	// On-chain USDT payments (payment module + platform/tron). The feature is
	// enabled iff exactly one of USDTXPub / USDTAddress is set; USDTProvider
	// selects the chain reader: "trongrid" (real chain) or "stub" (dev —
	// auto-pays after USDTStubDelay).
	USDTProvider string
	// USDTXPub is the watch-only BIP44 account key (m/44'/195'/0') deposit
	// addresses derive from (derived mode: one unique address per intent).
	// The mnemonic/xprv never touches the server.
	USDTXPub string
	// USDTAddress is a single fixed TRC20 deposit address every intent shares
	// (shared mode: payments are matched by exact salted amount instead of by
	// address; unmatched transfers go to the admin reconciliation queue).
	USDTAddress string
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

	// BEP20 (BSC) second network — additive to the TRC20 mode above and always
	// shared-address. Setting USDTBEP20Address enables it; USDTBEP20Provider
	// selects the chain reader: "jsonrpc" (real chain, free public BSC nodes,
	// no key), "etherscan" (real chain, Etherscan V2 multichain API — its
	// FREE tier does not cover BSC, a paid plan is required) or "stub" (dev).
	USDTBEP20Provider string
	// USDTBEP20Address is the single fixed BEP20 deposit address every BEP20
	// intent shares.
	USDTBEP20Address string
	// USDTBEP20Contract overrides the token contract (default BSC mainnet USDT).
	USDTBEP20Contract string
	// BSCRPCEndpoints are the public BSC JSON-RPC nodes the jsonrpc provider
	// reads (CSV; defaults to the canonical free BNB Chain dataseeds).
	BSCRPCEndpoints []string
	// EtherscanAPIKey authenticates Etherscan V2 calls (etherscan provider
	// only; the key must belong to a PAID plan for BSC access).
	EtherscanAPIKey  string
	EtherscanBaseURL string
	// USDTBEP20MinConfirmations gates how settled a BSC transfer must be
	// before it counts (default 15 ≈ seconds).
	USDTBEP20MinConfirmations int
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

// SupplierConfig is one upstream supplier. The panel suppliers (jentel /
// speedcard / gift4card) share one white-label API authed by Token; umanage is
// a telecom reseller (Kind "telecom") authed by APIKey/APISecret over a
// per-store path. Credentials are secrets and are never validated at boot: an
// absent credential simply disables the supplier (its provider id falls back to
// the parking stub), matching the SMS/IDCheck lazily-validated convention.
type SupplierConfig struct {
	ID       int    // provider-registry id (also Product.FulfillmentProvider)
	Name     string // slug shown in admin dropdown / logs
	Kind     string // "panel" (jentel/speedcard/gift4card) | "telecom" (umanage)
	Currency string // supplier's own balance currency: "USD" | "LBP"
	BaseURL  string
	// Panel auth (Kind == "panel").
	Token string // api-token header value
	// Telecom auth (Kind == "telecom", umanage).
	APIKey    string
	APISecret string
	StoreID   int // 0 → resolve at boot via GET /stores
}

// Configured reports whether this supplier has the credentials its Kind needs.
func (s SupplierConfig) Configured() bool {
	if s.Kind == "telecom" {
		return s.APIKey != "" && s.APISecret != ""
	}
	return s.Token != ""
}

// EnabledSuppliers returns the suppliers that have credentials configured.
func (c *Config) EnabledSuppliers() []SupplierConfig {
	var out []SupplierConfig
	for _, s := range c.Suppliers {
		if s.Configured() {
			out = append(out, s)
		}
	}
	return out
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

		BulkSMSMax:  getInt("BULK_SMS_MAX", 200),
		BulkPushMax: getInt("BULK_PUSH_MAX", 500),

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
		Suppliers: []SupplierConfig{
			{
				Name:     "jentel",
				Kind:     "panel",
				Currency: "USD",
				ID:       getInt("SUPPLIER_JENTEL_ID", 10),
				BaseURL:  getEnv("SUPPLIER_JENTEL_BASE_URL", "https://api.jentel-cash.com"),
				Token:    os.Getenv("SUPPLIER_JENTEL_TOKEN"),
			},
			{
				Name:     "speedcard",
				Kind:     "panel",
				Currency: "USD",
				ID:       getInt("SUPPLIER_SPEEDCARD_ID", 11),
				BaseURL:  getEnv("SUPPLIER_SPEEDCARD_BASE_URL", "https://api.speedcard.vip"),
				Token:    os.Getenv("SUPPLIER_SPEEDCARD_TOKEN"),
			},
			{
				Name:     "gift4card",
				Kind:     "panel",
				Currency: "USD",
				ID:       getInt("SUPPLIER_GIFT4CARD_ID", 12),
				BaseURL:  getEnv("SUPPLIER_GIFT4CARD_BASE_URL", "https://api.gift4card.com"),
				Token:    os.Getenv("SUPPLIER_GIFT4CARD_TOKEN"),
			},
			{
				Name:      "umanage",
				Kind:      "telecom",
				Currency:  "LBP",
				ID:        getInt("SUPPLIER_UMANAGE_ID", 13),
				BaseURL:   getEnv("SUPPLIER_UMANAGE_BASE_URL", "https://api.umanageapp.uk/api/v1/external"),
				APIKey:    os.Getenv("SUPPLIER_UMANAGE_KEY"),
				APISecret: os.Getenv("SUPPLIER_UMANAGE_SECRET"),
				StoreID:   getInt("SUPPLIER_UMANAGE_STORE_ID", 0),
			},
		},
		SupplierSettlerInterval: getDuration("SUPPLIER_SETTLER_INTERVAL", 60*time.Second),
		SupplierSettlerGiveUp:   getDuration("SUPPLIER_SETTLER_GIVEUP", 24*time.Hour),

		USDTProvider:      getEnv("USDT_PROVIDER", "stub"),
		USDTXPub:          os.Getenv("USDT_XPUB"),
		USDTAddress:       os.Getenv("USDT_ADDRESS"),
		USDTContract:      os.Getenv("USDT_CONTRACT"), // empty → platform/tron default
		TronGridAPIKey:    os.Getenv("TRONGRID_API_KEY"),
		TronGridBaseURL:   getEnv("TRONGRID_BASE_URL", "https://api.trongrid.io"),
		USDTIntentExpiry:  getDuration("USDT_INTENT_EXPIRY", 30*time.Minute),
		USDTWatchInterval: getDuration("USDT_WATCH_INTERVAL", 25*time.Second),
		USDTLateGrace:     getDuration("USDT_LATE_GRACE", 168*time.Hour),
		USDTStubDelay:     getDuration("USDT_STUB_DELAY", 15*time.Second),

		USDTBEP20Provider: getEnv("USDT_BEP20_PROVIDER", "stub"),
		USDTBEP20Address:  os.Getenv("USDT_BEP20_ADDRESS"),
		USDTBEP20Contract: os.Getenv("USDT_BEP20_CONTRACT"), // empty → platform/bsc default
		// Defaults are the free nodes verified to serve filtered eth_getLogs
		// over multi-thousand-block ranges (the canonical bsc-dataseed.*
		// nodes reject getLogs entirely — see platform/bsc/jsonrpc.go).
		BSCRPCEndpoints: getCSV("BSC_RPC_ENDPOINTS", []string{
			"https://rpc-bsc.48.club",
			"https://bsc-mainnet.nodereal.io/v1/64a9df0874fb4a93b9d0a3849de012d3",
		}),
		EtherscanAPIKey:           os.Getenv("ETHERSCAN_API_KEY"),
		EtherscanBaseURL:          getEnv("ETHERSCAN_BASE_URL", "https://api.etherscan.io/v2/api"),
		USDTBEP20MinConfirmations: getInt("USDT_BEP20_MIN_CONFIRMATIONS", 15),

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
			TouchRechargeTemplate: getEnv("BRIDGE_TOUCH_RECHARGE_TEMPLATE", "*300*961{phone}*{card}#"),
			TouchTransferTemplate: getEnv("BRIDGE_TOUCH_TRANSFER_TEMPLATE", "{phone}T{amount}"),
			TouchTransferDest:     getEnv("BRIDGE_TOUCH_TRANSFER_DEST", "1199"),
			// Alfa recharge is a USSD dial (not SMS): *111*<card PIN>*<number>#.
			AlfaRechargeTemplate: getEnv("BRIDGE_ALFA_RECHARGE_TEMPLATE", "*111*{code}*{phone}#"),
			AlfaRechargeDest:     getEnv("BRIDGE_ALFA_RECHARGE_DEST", "1313"),
			AlfaTransferTemplate: getEnv("BRIDGE_ALFA_TRANSFER_TEMPLATE", "{phone}T{amount}"),
			AlfaTransferDest:     getEnv("BRIDGE_ALFA_TRANSFER_DEST", "1313"),
			TouchMinBalance:      getFloat("BRIDGE_TOUCH_MIN_BALANCE", 20),
			AlfaMinBalance:       getFloat("BRIDGE_ALFA_MIN_BALANCE", 20),
			TouchMessageFee:      getFloat("BRIDGE_TOUCH_MESSAGE_FEE", 0.16),
			AlfaMessageFee:       getFloat("BRIDGE_ALFA_MESSAGE_FEE", 0.14),
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
	// An ambiguous USDT addressing mode is a config bug in EVERY env — the two
	// modes are mutually exclusive by design, so refuse before the dev bypass.
	usdtOn := c.USDTXPub != "" || c.USDTAddress != ""
	if c.USDTXPub != "" && c.USDTAddress != "" {
		return errors.New("set exactly one of USDT_XPUB (derived addresses) / USDT_ADDRESS (shared address) — not both")
	}
	// Duplicate fulfillment-provider ids are a config bug in EVERY env: the
	// registry silently keeps one adapter per id, so a collision would misroute
	// paid orders to the wrong supplier. Only ENABLED suppliers (and the mock,
	// when on) occupy ids — disabled ones can share defaults harmlessly.
	seenProviderIDs := map[int]string{}
	if c.FulfillmentMock {
		seenProviderIDs[c.FulfillmentMockID] = "reference mock"
	}
	for _, s := range c.EnabledSuppliers() {
		if prev, dup := seenProviderIDs[s.ID]; dup {
			return fmt.Errorf("supplier %q and %s share fulfillment-provider id %d — set distinct SUPPLIER_*_ID values", s.Name, prev, s.ID)
		}
		seenProviderIDs[s.ID] = "supplier " + strconv.Quote(s.Name)
	}
	if c.Env == "development" {
		return nil
	}
	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET is required when ENV is not development (admin auth would be disabled)")
	}
	if c.AllowedOrigins == "" {
		return errors.New("ALLOWED_ORIGINS is required when ENV is not development")
	}
	if usdtOn && c.USDTProvider == "stub" {
		return errors.New("USDT_PROVIDER=stub would auto-confirm unpaid USDT payments outside development — set USDT_PROVIDER=trongrid or unset USDT_XPUB/USDT_ADDRESS")
	}
	if c.USDTProvider == "trongrid" && usdtOn && c.TronGridAPIKey == "" {
		return errors.New("TRONGRID_API_KEY is required when USDT payments are enabled with the trongrid provider")
	}
	if c.USDTBEP20Address != "" && c.USDTBEP20Provider == "stub" {
		return errors.New("USDT_BEP20_PROVIDER=stub would auto-confirm unpaid USDT payments outside development — set USDT_BEP20_PROVIDER=jsonrpc (or etherscan) or unset USDT_BEP20_ADDRESS")
	}
	if c.USDTBEP20Address != "" && c.USDTBEP20Provider == "etherscan" && c.EtherscanAPIKey == "" {
		return errors.New("ETHERSCAN_API_KEY is required when BEP20 USDT payments are enabled with the etherscan provider")
	}
	// A typo'd BEP20 provider would otherwise silently disable the network at
	// boot (server-side degrade) — fail loud instead, since enabling BEP20 is
	// exactly an env-flip where a typo is likely. (TRC20's USDT_PROVIDER keeps
	// its historical degrade behavior; changing that is out of scope here.)
	if c.USDTBEP20Address != "" && c.USDTBEP20Provider != "jsonrpc" && c.USDTBEP20Provider != "etherscan" {
		return fmt.Errorf("unknown USDT_BEP20_PROVIDER %q — use jsonrpc (free public BSC nodes) or etherscan (paid plan)", c.USDTBEP20Provider)
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
