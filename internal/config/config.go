package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	MatchThreshold float64
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:       envOrDefault("HTTP_ADDR", ":8080"),
		MatchThreshold: 0.75,
	}

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if v := os.Getenv("MATCH_THRESHOLD"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid MATCH_THRESHOLD: %w", err)
		}
		cfg.MatchThreshold = f
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
