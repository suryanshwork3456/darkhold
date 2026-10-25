package config

import (
	"log"
	"os"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	TokenTTL    time.Duration
}

func Load() Config {
	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgresrsu@localhost:5432/darkhold?sslmode=disable"),
		JWTSecret:   getEnv("Ao7DhxvpaLtrpT+ttctyXbTFng+Uav9fp8Z2gU9hlFg=", "dev-secret-change-me"),
		TokenTTL:    getDuration("TOKEN_TTL", 24*time.Hour),
	}

	if cfg.JWTSecret == "dev-secret-change-me" {
		log.Println("WARNING: using default JWT secret, set JWT_SECRET")
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Printf("bad %s value %q, using default", key, v)
		return fallback
	}
	return d
}