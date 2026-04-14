package config

import (
	"crypto/rsa"
	"fmt"
	"os"
	"strconv"

	"github.com/dxngee/antifraud-processing/internal/utils"
)

type Config struct {
	PrivateKey     *rsa.PrivateKey
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

	if v := os.Getenv("PRIVATE_KEY"); v != "" {
		privateKey, err := utils.ParsePrivateKey(v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		cfg.PrivateKey = privateKey
	} else {
		return nil, fmt.Errorf("PRIVATE_KEY is required")
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
