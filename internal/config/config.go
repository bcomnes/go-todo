package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Host        string
	Port        string
}

func Load(envPaths ...string) Config {
	// Optional: allow test to pass explicit .env path
	_ = godotenv.Load(envPaths...)

	return Config{
			DatabaseURL: os.Getenv("DATABASE_URL"),
			Host:        fallback(os.Getenv("HOST"), "localhost"),
			Port:        fallback(os.Getenv("PORT"), "8080"),
		}
}

func fallback(val, def string) string {
	if val != "" {
		return val
	}
	return def
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return val
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
