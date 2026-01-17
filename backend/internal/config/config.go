package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds runtime configuration for the API.
type Config struct {
	ClickHouseAddr         string
	ClickHouseUser         string
	ClickHousePassword     string
	ListenAddr             string
	AuthMode               string
	AuthSecret             string
	AuthCookieName         string
	CleanupIntervalMinutes int
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		ClickHouseAddr:         os.Getenv("CLICKHOUSE_ADDR"),
		ClickHouseUser:         os.Getenv("CLICKHOUSE_USER"),
		ClickHousePassword:     os.Getenv("CLICKHOUSE_PASSWORD"),
		ListenAddr:             os.Getenv("API_LISTEN_ADDR"),
		AuthMode:               os.Getenv("AUTH_MODE"),
		AuthSecret:             os.Getenv("AUTH_SECRET"),
		AuthCookieName:         os.Getenv("AUTH_COOKIE_NAME"),
		CleanupIntervalMinutes: getEnvInt("CLEANUP_INTERVAL_MINUTES", 1440),
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

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
