package database_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bcomnes/go-todo/internal/config"
	"github.com/bcomnes/go-todo/internal/database"
	"github.com/joho/godotenv"
)

func TestDatabaseConnection(t *testing.T) {
	// Optionally load .env from the module root
	rootEnv := filepath.Join("..", "..", ".env")
	if _, err := os.Stat(rootEnv); err == nil {
		_ = godotenv.Load(rootEnv)
	}

	cfg := config.Load() // Uses env already loaded

	if cfg.DatabaseURL == "" {
		t.Fatal("DATABASE_URL must be set (check .env or environment)")
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		t.Fatalf("database ping failed: %v", err)
	}
}
