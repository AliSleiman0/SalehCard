package config

import "os"

// Config holds all application configuration values.
type Config struct {
	MongoURI       string
	DBName         string
	Port           string
	JWTSecret      string
	GoogleClientID string
	AllowedOrigins string
	Env            string
}

// Load reads configuration from environment variables, applying defaults where needed.
func Load() *Config {
	return &Config{
		MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:         getEnv("DB_NAME", "salehcard"),
		Port:           getEnv("PORT", "8080"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),
		AllowedOrigins: os.Getenv("ALLOWED_ORIGINS"),
		Env:            getEnv("ENV", "development"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
