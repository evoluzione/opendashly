package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds runtime configuration for the API.
type Config struct {
	ClickHouseAddr               string
	ClickHouseUser               string
	ClickHousePassword           string
	ClickHouseMaxMemoryMiB       int
	ClickHouseExternalGroupByMiB int
	ClickHouseExternalSortMiB    int
	ClickHouseTempDiskMiB        int
	ClickHouseMaxExecSec         int
	ClickHouseMaxOpenConns       int
	ClickHouseMaxIdleConns       int
	ClickHouseDialTimeout        int
	ClickHouseReadTimeout        int
	DashboardFreshCacheTTLSec    int
	DashboardStaleCacheTTLSec    int
	DashboardRequestTimeoutSec   int
	DashboardQueryParallelism    int
	DashboardHalveOnOOM          bool
	CollectorHealthURL           string
	ListenAddr                   string
	AuthMode                     string
	AuthSecret                   string
	AuthCookieName               string
	CleanupIntervalMinutes       int
	ServiceListTimeoutSec        int
	RetentionPreCount            bool
	CORSAllowedOrigins           []string
	DebugQuery                   bool
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		ClickHouseAddr:               os.Getenv("CLICKHOUSE_ADDR"),
		ClickHouseUser:               os.Getenv("CLICKHOUSE_USER"),
		ClickHousePassword:           os.Getenv("CLICKHOUSE_PASSWORD"),
		ClickHouseMaxMemoryMiB:       getEnvInt("CLICKHOUSE_MAX_MEMORY_MIB", 200),
		ClickHouseExternalGroupByMiB: getEnvInt("CLICKHOUSE_MAX_BYTES_BEFORE_EXTERNAL_GROUP_BY_MIB", 64),
		ClickHouseExternalSortMiB:    getEnvInt("CLICKHOUSE_MAX_BYTES_BEFORE_EXTERNAL_SORT_MIB", 64),
		ClickHouseTempDiskMiB:        getEnvInt("CLICKHOUSE_MAX_TEMP_DATA_ON_DISK_MIB", 1024),
		ClickHouseMaxExecSec:         getEnvInt("CLICKHOUSE_MAX_EXECUTION_TIME_SECONDS", 8),
		ClickHouseMaxOpenConns:       getEnvInt("CLICKHOUSE_MAX_OPEN_CONNS", 5),
		ClickHouseMaxIdleConns:       getEnvInt("CLICKHOUSE_MAX_IDLE_CONNS", 3),
		ClickHouseDialTimeout:        getEnvInt("CLICKHOUSE_DIAL_TIMEOUT_SECONDS", 5),
		ClickHouseReadTimeout:        getEnvInt("CLICKHOUSE_READ_TIMEOUT_SECONDS", 10),
		DashboardFreshCacheTTLSec:    getEnvInt("DASHBOARD_FRESH_CACHE_TTL_SECONDS", 30),
		DashboardStaleCacheTTLSec:    getEnvInt("DASHBOARD_STALE_CACHE_TTL_SECONDS", 900),
		DashboardRequestTimeoutSec:   getEnvInt("DASHBOARD_REQUEST_TIMEOUT_SECONDS", 15),
		DashboardQueryParallelism:    getEnvInt("DASHBOARD_QUERY_PARALLELISM", 1),
		DashboardHalveOnOOM:          getEnvBool("DASHBOARD_HALVE_ON_OOM", true),
		CollectorHealthURL:           os.Getenv("COLLECTOR_HEALTH_URL"),
		ListenAddr:                   os.Getenv("API_LISTEN_ADDR"),
		AuthMode:                     os.Getenv("AUTH_MODE"),
		AuthSecret:                   os.Getenv("AUTH_SECRET"),
		AuthCookieName:               os.Getenv("AUTH_COOKIE_NAME"),
		CleanupIntervalMinutes:       getEnvInt("CLEANUP_INTERVAL_MINUTES", 1440),
		ServiceListTimeoutSec:        getEnvInt("SERVICE_LIST_TIMEOUT_SECONDS", 15),
		RetentionPreCount:            getEnvBool("RETENTION_COUNT_PRECHECK_ENABLED", false),
		CORSAllowedOrigins:           getEnvCSV("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
		DebugQuery:                   os.Getenv("VITE_DEBUG_QUERY") == "true",
	}
	if cfg.ClickHouseAddr == "" {
		return nil, fmt.Errorf("CLICKHOUSE_ADDR is required")
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	if cfg.CollectorHealthURL == "" {
		cfg.CollectorHealthURL = "http://otel-collector:13133"
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

func getEnvCSV(key string, defaultVals []string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVals
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			out = append(out, value)
		}
	}
	if len(out) == 0 {
		return defaultVals
	}
	return out
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}
