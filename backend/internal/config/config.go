package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	Env         string
	HTTPAddr    string
	DatabaseURL string
	JWTSecret   string
	CORSOrigins []string
	TokenTTL    time.Duration
}

func Load() Config {
	return Config{
		Env:         getEnv("APP_ENV", "development"),
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:123456@127.0.0.1:5432/image-ai?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "local-dev-secret"),
		CORSOrigins: splitCSV(getEnv("CORS_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")),
		TokenTTL:    24 * time.Hour,
	}
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
