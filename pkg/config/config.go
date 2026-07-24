// Package config loads and validates process configuration from environment
// variables and optional dotenv files.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const defaultTokenTTL = 7 * 24 * time.Hour

// Config contains the shared runtime settings used by the HTTP server.
type Config struct {
	DatabaseURL  string
	Host         string
	Port         string
	TokenTTL     time.Duration
	SecureCookie bool
}

// Load reads configuration from the process environment. If envPaths are
// provided, they are loaded in order before values are read; otherwise Load
// makes a best-effort attempt to load .env from the current directory.
func Load(envPaths ...string) (Config, error) {
	if len(envPaths) > 0 {
		_ = godotenv.Load(envPaths...)
	} else {
		_ = godotenv.Load()
	}

	cfg := Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		Host:         fallback(os.Getenv("HOST"), "127.0.0.1"),
		Port:         fallback(os.Getenv("PORT"), "8080"),
		TokenTTL:     defaultTokenTTL,
		SecureCookie: true,
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	if rawTTL := os.Getenv("TOKEN_TTL"); rawTTL != "" {
		ttl, err := time.ParseDuration(rawTTL)
		if err != nil {
			return Config{}, fmt.Errorf("parse TOKEN_TTL: %w", err)
		}
		if ttl <= 0 {
			return Config{}, errors.New("TOKEN_TTL must be greater than zero")
		}
		cfg.TokenTTL = ttl
	}

	if rawSecureCookie := os.Getenv("SESSION_COOKIE_SECURE"); rawSecureCookie != "" {
		secureCookie, err := strconv.ParseBool(rawSecureCookie)
		if err != nil {
			return Config{}, fmt.Errorf("parse SESSION_COOKIE_SECURE: %w", err)
		}
		cfg.SecureCookie = secureCookie
	}

	return cfg, nil
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
