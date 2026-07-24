package config_test

import (
	"testing"
	"time"

	"github.com/bcomnes/go-todo/pkg/config"
)

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example/go_todo")
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("TOKEN_TTL", "24h")
	t.Setenv("SESSION_COOKIE_SECURE", "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != "8080" {
		t.Fatalf("unexpected defaults: host=%q port=%q", cfg.Host, cfg.Port)
	}
	if cfg.TokenTTL != 24*time.Hour {
		t.Fatalf("unexpected token TTL: %s", cfg.TokenTTL)
	}
	if !cfg.SecureCookie {
		t.Fatal("secure session cookie setting was not loaded")
	}

}

func TestLoadSecureCookieDefaultsOn(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example/go_todo")
	t.Setenv("SESSION_COOKIE_SECURE", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.SecureCookie {
		t.Fatal("secure session cookies should be enabled by default")
	}
}

func TestLoadRejectsInvalidSecureCookie(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example/go_todo")
	t.Setenv("SESSION_COOKIE_SECURE", "sometimes")
	if _, err := config.Load(); err == nil {
		t.Fatal("Load unexpectedly accepted an invalid SESSION_COOKIE_SECURE")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := config.Load(); err == nil {
		t.Fatal("Load unexpectedly accepted an empty DATABASE_URL")
	}
}
