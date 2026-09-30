package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds the runtime settings sourced from the environment.
type Config struct {
	AppName     string
	Port        string
	DatabaseURL string
}

// Load reads configuration from the environment, optionally from a .env file.
func Load() Config {
	_ = godotenv.Load()

	return Config{
		AppName:     env("APP_NAME", "M.N.G.R Manager"),
		Port:        env("PORT", "8000"),
		DatabaseURL: env("DATABASE_URL", "postgres://manager:secret@localhost:5432/manager?sslmode=disable"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
