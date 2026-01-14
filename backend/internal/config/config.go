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
	AuthSecret     string
	AuthCookieName string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		ClickHouseAddr: os.Getenv("CLICKHOUSE_ADDR"),
		ListenAddr:     os.Getenv("API_LISTEN_ADDR"),
		AuthMode:       os.Getenv("AUTH_MODE"),
		AuthSecret:     os.Getenv("AUTH_SECRET"),
		AuthCookieName: os.Getenv("AUTH_COOKIE_NAME"),
	}
	if cfg.ClickHouseAddr == "" {
		return nil, fmt.Errorf("CLICKHOUSE_ADDR is required")
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	if cfg.AuthMode == "" {
		cfg.AuthMode = "jwt"
	}
	if cfg.AuthSecret == "" {
		cfg.AuthSecret = "dev-secret"
	}
	if cfg.AuthCookieName == "" {
		cfg.AuthCookieName = "oteldash_session"
	}
	return cfg, nil
}
