package config_test

import (
	"testing"

	"github.com/bcomnes/go-todo/internal/config"
)

func TestConfigLoad(t *testing.T) {
	cfg := config.Load("../../.env")

	if cfg.DatabaseURL == "" {
		t.Fatal("DATABASE_URL should not be empty (check your environment or .env file)")
	}
	if cfg.Host == "" {
		t.Errorf("HOST should not be empty")
	}
	if cfg.Port == "" {
		t.Errorf("PORT should not be empty")
	}
}
