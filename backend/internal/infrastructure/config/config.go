package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds runtime configuration for the API.
//
// The load-sensitive knobs below (DashboardQueryParallelism,
// TelemetryQueryConcurrency, ClickHouseMaxMemoryMiB and its spill thresholds)
// are seeded here at their safe floor and then owned at runtime by the adaptive
// tuning controller. Every other knob is a single resilient default — there are
// no longer small/standard/big machine profiles; the system discovers its own
// ceiling under real pressure.
type Config struct {
	RetentionAdaptiveEnabled     bool
	MaxLogRetentionDays          int
	MaxTraceRetentionDays        int
	RetentionStepDownDays        int
	RetentionMaxLevel            int
	RetentionMinTraceHours       int
	RetentionMinLogHours         int
	RetentionPressureCooldownSec int
	RetentionPressureMinSignals  int
	RetentionPressureWindowSec   int
	RetentionPressureErrorCount  int
	RetentionPressureMemPct      int
	RetentionPressureMemBudgetMB int
	RetentionPressureDiskPct     int
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
	TelemetryQueryConcurrency    int
	DashboardHalveOnOOM          bool
	DashboardRawFallback         bool
	DashboardPressureCooldownSec int
	DashboardLastGoodTTLSec      int
	DashboardRollupBackfill      bool
	DashboardRollupBackfillHours int
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
		ClickHouseAddr:     os.Getenv("CLICKHOUSE_ADDR"),
		ClickHouseUser:     os.Getenv("CLICKHOUSE_USER"),
		ClickHousePassword: os.Getenv("CLICKHOUSE_PASSWORD"),
		CollectorHealthURL: os.Getenv("COLLECTOR_HEALTH_URL"),
		ListenAddr:         os.Getenv("API_LISTEN_ADDR"),
		AuthMode:           os.Getenv("AUTH_MODE"),
		AuthSecret:         os.Getenv("AUTH_SECRET"),
		AuthCookieName:     os.Getenv("AUTH_COOKIE_NAME"),
		CORSAllowedOrigins: getEnvCSV("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
		DebugQuery:         os.Getenv("VITE_DEBUG_QUERY") == "true",
	}
	cfg.applyDefaults()
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

// applyDefaults seeds the single resilient default for every tuning knob. There
// are no machine profiles: knobs that are not safe to vary live (timeouts,
// pools, retention caps, pressure thresholds) get one conservative value, while
// the load-sensitive knobs (DashboardQueryParallelism, TelemetryQueryConcurrency,
// ClickHouseMaxMemoryMiB and its spill thresholds) are seeded at their floor and
// then driven at runtime by the adaptive tuning controller.
func (cfg *Config) applyDefaults() {
	// Static, resilient defaults (set once, never varied at runtime).
	cfg.ServiceListTimeoutSec = 20
	cfg.CleanupIntervalMinutes = 720
	cfg.RetentionAdaptiveEnabled = true
	// Generous retention caps: adaptive retention + disk-pressure step-down pull
	// these down only under real pressure, so we never pre-emptively drop data.
	cfg.MaxLogRetentionDays = 30
	cfg.MaxTraceRetentionDays = 15
	cfg.RetentionStepDownDays = 2
	cfg.RetentionMaxLevel = 2
	cfg.RetentionMinTraceHours = 24
	cfg.RetentionMinLogHours = 24
	cfg.RetentionPressureCooldownSec = 300
	cfg.RetentionPressureMinSignals = 2
	cfg.RetentionPressureWindowSec = 300
	cfg.RetentionPressureErrorCount = 4
	cfg.RetentionPressureMemPct = 85
	cfg.RetentionPressureMemBudgetMB = 256
	cfg.RetentionPressureDiskPct = 90
	cfg.RetentionPreCount = false
	cfg.ClickHouseTempDiskMiB = 768
	cfg.ClickHouseMaxExecSec = 20
	cfg.ClickHouseMaxOpenConns = 4
	cfg.ClickHouseMaxIdleConns = 2
	cfg.ClickHouseDialTimeout = 5
	cfg.ClickHouseReadTimeout = 30
	cfg.DashboardFreshCacheTTLSec = 30
	cfg.DashboardStaleCacheTTLSec = 900
	cfg.DashboardRequestTimeoutSec = 20
	cfg.DashboardHalveOnOOM = true
	cfg.DashboardRawFallback = false
	cfg.DashboardPressureCooldownSec = 60
	cfg.DashboardLastGoodTTLSec = 1800
	cfg.DashboardRollupBackfill = true
	cfg.DashboardRollupBackfillHours = 48

	// Load-sensitive knobs: seeded at the controller floor. The tuning
	// controller raises them while the backend is calm and cuts them under
	// pressure. The values here are the connection-level safety floor used
	// before the controller's first tick and as a fallback for non-dashboard
	// queries.
	cfg.DashboardQueryParallelism = 1
	cfg.TelemetryQueryConcurrency = 1
	cfg.ClickHouseMaxMemoryMiB = 64
	cfg.ClickHouseExternalGroupByMiB = 16
	cfg.ClickHouseExternalSortMiB = 16
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
