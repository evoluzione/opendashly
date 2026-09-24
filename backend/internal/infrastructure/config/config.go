package config

import (
	"crypto/rand"
	"encoding/base64"
	"os"
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
	CORSAllowedOrigins           []string
	DebugQuery                   bool
}

// Load builds runtime configuration. Operational knobs are auto-derived; only
// secrets and deployment-facing integration policy remain environment-driven.
func Load() (*Config, error) {
	cfg := &Config{
		ClickHouseAddr:     "clickhouse:9000",
		ClickHouseUser:     "otel",
		ClickHousePassword: os.Getenv("CLICKHOUSE_PASSWORD"),
		CollectorHealthURL: "http://otel-collector:13133",
		ListenAddr:         ":8080",
		AuthMode:           "jwt",
		AuthSecret:         os.Getenv("AUTH_SECRET"),
		AuthCookieName:     "oteldash_session",
		CORSAllowedOrigins: envCSV("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
		DebugQuery:         false,
	}
	if cfg.AuthSecret == "" {
		cfg.AuthSecret = generateEphemeralSecret()
	}
	cfg.applyDefaults()
	return cfg, nil
}

func envCSV(key string, defaultVals []string) []string {
	raw := os.Getenv(key)
	if strings.TrimSpace(raw) == "" {
		return defaultVals
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return defaultVals
	}
	return values
}

func generateEphemeralSecret() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "opendashly-ephemeral-auth-secret"
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// applyDefaults derives every non-secret operational knob locally. Environment
// variables no longer participate in tuning: explicit env overrides caused the
// deployed system to freeze on stale values instead of adapting to the machine.
func (cfg *Config) applyDefaults() {
	auto := Auto()

	cfg.ServiceListTimeoutSec = auto.ServiceListTimeoutSec
	cfg.CleanupIntervalMinutes = auto.CleanupIntervalMinutes
	cfg.RetentionAdaptiveEnabled = auto.RetentionAdaptiveEnabled
	cfg.MaxLogRetentionDays = auto.MaxLogRetentionDays
	cfg.MaxTraceRetentionDays = auto.MaxTraceRetentionDays
	cfg.RetentionStepDownDays = auto.RetentionStepDownDays
	cfg.RetentionMaxLevel = auto.RetentionMaxLevel
	cfg.RetentionMinTraceHours = auto.RetentionMinTraceHours
	cfg.RetentionMinLogHours = auto.RetentionMinLogHours
	cfg.RetentionPressureCooldownSec = auto.RetentionPressureCooldownSec
	cfg.RetentionPressureMinSignals = auto.RetentionPressureMinSignals
	cfg.RetentionPressureWindowSec = auto.RetentionPressureWindowSec
	cfg.RetentionPressureErrorCount = auto.RetentionPressureErrorCount
	cfg.RetentionPressureMemPct = auto.RetentionPressureMemPct
	cfg.RetentionPressureMemBudgetMB = auto.RetentionPressureMemBudgetMB
	cfg.RetentionPressureDiskPct = auto.RetentionPressureDiskPct
	cfg.ClickHouseTempDiskMiB = auto.ClickHouseTempDiskMiB
	cfg.ClickHouseMaxExecSec = auto.ClickHouseMaxExecSec
	cfg.ClickHouseMaxOpenConns = auto.ClickHouseMaxOpenConns
	cfg.ClickHouseMaxIdleConns = auto.ClickHouseMaxIdleConns
	cfg.ClickHouseDialTimeout = auto.ClickHouseDialTimeoutSec
	cfg.ClickHouseReadTimeout = auto.ClickHouseReadTimeoutSec
	cfg.DashboardFreshCacheTTLSec = auto.DashboardFreshCacheTTLSec
	cfg.DashboardStaleCacheTTLSec = auto.DashboardStaleCacheTTLSec
	cfg.DashboardRequestTimeoutSec = auto.DashboardRequestTimeoutSec
	cfg.DashboardHalveOnOOM = auto.DashboardHalveOnOOM
	cfg.DashboardRawFallback = auto.DashboardRawFallback
	cfg.DashboardPressureCooldownSec = auto.DashboardPressureCooldownSec
	cfg.DashboardLastGoodTTLSec = auto.DashboardLastGoodTTLSec
	cfg.DashboardRollupBackfill = auto.DashboardRollupBackfill
	cfg.DashboardRollupBackfillHours = auto.DashboardRollupBackfillHours

	// Load-sensitive knobs start at the floor and are then owned by the runtime
	// AIMD tuning controller.
	cfg.DashboardQueryParallelism = auto.DashboardQueryParallelism
	cfg.TelemetryQueryConcurrency = auto.TelemetryQueryConcurrency
	cfg.ClickHouseMaxMemoryMiB = auto.ClickHouseMaxMemoryMiB
	cfg.ClickHouseExternalGroupByMiB = auto.ClickHouseExternalGroupByMiB
	cfg.ClickHouseExternalSortMiB = auto.ClickHouseExternalSortMiB
}
