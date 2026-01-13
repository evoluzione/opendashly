package config

import (
	"fmt"
	"os"
)

// Config holds runtime configuration for the API.
type Config struct {
	ClickHouseAddr string
	ListenAddr     string
	AuthMode       string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		ClickHouseAddr: os.Getenv("CLICKHOUSE_ADDR"),
		ListenAddr:     os.Getenv("API_LISTEN_ADDR"),
		AuthMode:       os.Getenv("AUTH_MODE"),
	}
	if cfg.ClickHouseAddr == "" {
		return nil, fmt.Errorf("CLICKHOUSE_ADDR is required")
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	if cfg.AuthMode == "" {
		cfg.AuthMode = "header"
	}
	return cfg, nil
}
