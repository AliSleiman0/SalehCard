package config

import (
	"os"
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
	}
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
