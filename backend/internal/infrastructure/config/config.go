package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds runtime configuration for the API.
type Config struct {
	MachineProfile               string
	MachineRAMGB                 int
	MachineCPUCores              int
	MachineAutoTuning            bool
	ResolvedMachineProfile       string
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

type machinePreset struct {
	ServiceListTimeoutSec        int
	CleanupIntervalMinutes       int
	RetentionPreCount            bool
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
	DashboardRawFallback         bool
	DashboardPressureCooldownSec int
	DashboardLastGoodTTLSec      int
	DashboardRollupBackfill      bool
	DashboardRollupBackfillHours int
}

var machinePresets = map[string]machinePreset{
	"small": {
		ServiceListTimeoutSec:        25,
		CleanupIntervalMinutes:       1440,
		RetentionPreCount:            false,
		ClickHouseMaxMemoryMiB:       96,
		ClickHouseExternalGroupByMiB: 24,
		ClickHouseExternalSortMiB:    24,
		ClickHouseTempDiskMiB:        512,
		ClickHouseMaxExecSec:         25,
		ClickHouseMaxOpenConns:       2,
		ClickHouseMaxIdleConns:       1,
		ClickHouseDialTimeout:        5,
		ClickHouseReadTimeout:        40,
		DashboardFreshCacheTTLSec:    30,
		DashboardStaleCacheTTLSec:    900,
		DashboardRequestTimeoutSec:   25,
		DashboardQueryParallelism:    1,
		DashboardHalveOnOOM:          true,
		DashboardRawFallback:         false,
		DashboardPressureCooldownSec: 60,
		DashboardLastGoodTTLSec:      1800,
		DashboardRollupBackfill:      true,
		DashboardRollupBackfillHours: 48,
	},
	"standard": {
		ServiceListTimeoutSec:        20,
		CleanupIntervalMinutes:       720,
		RetentionPreCount:            false,
		ClickHouseMaxMemoryMiB:       160,
		ClickHouseExternalGroupByMiB: 48,
		ClickHouseExternalSortMiB:    48,
		ClickHouseTempDiskMiB:        768,
		ClickHouseMaxExecSec:         20,
		ClickHouseMaxOpenConns:       4,
		ClickHouseMaxIdleConns:       2,
		ClickHouseDialTimeout:        5,
		ClickHouseReadTimeout:        30,
		DashboardFreshCacheTTLSec:    30,
		DashboardStaleCacheTTLSec:    900,
		DashboardRequestTimeoutSec:   20,
		DashboardQueryParallelism:    1,
		DashboardHalveOnOOM:          true,
		DashboardRawFallback:         false,
		DashboardPressureCooldownSec: 60,
		DashboardLastGoodTTLSec:      1800,
		DashboardRollupBackfill:      true,
		DashboardRollupBackfillHours: 48,
	},
	"big": {
		ServiceListTimeoutSec:        15,
		CleanupIntervalMinutes:       360,
		RetentionPreCount:            false,
		ClickHouseMaxMemoryMiB:       256,
		ClickHouseExternalGroupByMiB: 64,
		ClickHouseExternalSortMiB:    64,
		ClickHouseTempDiskMiB:        1024,
		ClickHouseMaxExecSec:         15,
		ClickHouseMaxOpenConns:       8,
		ClickHouseMaxIdleConns:       4,
		ClickHouseDialTimeout:        5,
		ClickHouseReadTimeout:        20,
		DashboardFreshCacheTTLSec:    30,
		DashboardStaleCacheTTLSec:    900,
		DashboardRequestTimeoutSec:   15,
		DashboardQueryParallelism:    2,
		DashboardHalveOnOOM:          true,
		DashboardRawFallback:         false,
		DashboardPressureCooldownSec: 60,
		DashboardLastGoodTTLSec:      1800,
		DashboardRollupBackfill:      true,
		DashboardRollupBackfillHours: 48,
	},
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		MachineProfile:          strings.ToLower(strings.TrimSpace(os.Getenv("MACHINE_PROFILE"))),
		MachineRAMGB:            getEnvInt("MACHINE_RAM_GB", 0),
		MachineCPUCores:         getEnvInt("MACHINE_CPU_CORES", 0),
		ClickHouseAddr:          os.Getenv("CLICKHOUSE_ADDR"),
		ClickHouseUser:          os.Getenv("CLICKHOUSE_USER"),
		ClickHousePassword:      os.Getenv("CLICKHOUSE_PASSWORD"),
		ClickHouseDialTimeout:   5,
		DashboardHalveOnOOM:     true,
		DashboardRollupBackfill: true,
		CollectorHealthURL:      os.Getenv("COLLECTOR_HEALTH_URL"),
		ListenAddr:              os.Getenv("API_LISTEN_ADDR"),
		AuthMode:                os.Getenv("AUTH_MODE"),
		AuthSecret:              os.Getenv("AUTH_SECRET"),
		AuthCookieName:          os.Getenv("AUTH_COOKIE_NAME"),
		RetentionPreCount:       false,
		CORSAllowedOrigins:      getEnvCSV("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
		DebugQuery:              os.Getenv("VITE_DEBUG_QUERY") == "true",
	}
	cfg.applyMachineTuning()
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

func (cfg *Config) applyMachineTuning() {
	profile := resolveMachineProfile(cfg.MachineProfile, cfg.MachineRAMGB, cfg.MachineCPUCores)

	preset, ok := machinePresets[profile]
	if !ok {
		preset = machinePresets["standard"]
		profile = "standard"
	}

	cfg.MachineAutoTuning = true
	cfg.ResolvedMachineProfile = profile

	// Auto-tuning intentionally overrides low-level manual knobs when active.
	cfg.ServiceListTimeoutSec = preset.ServiceListTimeoutSec
	cfg.CleanupIntervalMinutes = preset.CleanupIntervalMinutes
	cfg.RetentionPreCount = preset.RetentionPreCount
	cfg.ClickHouseMaxMemoryMiB = preset.ClickHouseMaxMemoryMiB
	cfg.ClickHouseExternalGroupByMiB = preset.ClickHouseExternalGroupByMiB
	cfg.ClickHouseExternalSortMiB = preset.ClickHouseExternalSortMiB
	cfg.ClickHouseTempDiskMiB = preset.ClickHouseTempDiskMiB
	cfg.ClickHouseMaxExecSec = preset.ClickHouseMaxExecSec
	cfg.ClickHouseMaxOpenConns = preset.ClickHouseMaxOpenConns
	cfg.ClickHouseMaxIdleConns = preset.ClickHouseMaxIdleConns
	cfg.ClickHouseDialTimeout = preset.ClickHouseDialTimeout
	cfg.ClickHouseReadTimeout = preset.ClickHouseReadTimeout
	cfg.DashboardFreshCacheTTLSec = preset.DashboardFreshCacheTTLSec
	cfg.DashboardStaleCacheTTLSec = preset.DashboardStaleCacheTTLSec
	cfg.DashboardRequestTimeoutSec = preset.DashboardRequestTimeoutSec
	cfg.DashboardQueryParallelism = preset.DashboardQueryParallelism
	cfg.DashboardHalveOnOOM = preset.DashboardHalveOnOOM
	cfg.DashboardRawFallback = preset.DashboardRawFallback
	cfg.DashboardPressureCooldownSec = preset.DashboardPressureCooldownSec
	cfg.DashboardLastGoodTTLSec = preset.DashboardLastGoodTTLSec
	cfg.DashboardRollupBackfill = preset.DashboardRollupBackfill
	cfg.DashboardRollupBackfillHours = preset.DashboardRollupBackfillHours
}

func resolveMachineProfile(profile string, ramGB int, cpuCores int) string {
	if _, ok := machinePresets[profile]; ok {
		return profile
	}

	if ramGB <= 0 && cpuCores <= 0 {
		return "standard"
	}

	classByRAM := resourceClassFromRAM(ramGB)
	classByCPU := resourceClassFromCPU(cpuCores)
	class := minPositiveClass(classByRAM, classByCPU)

	switch class {
	case 1:
		return "small"
	case 2:
		return "standard"
	default:
		return "big"
	}
}

func resourceClassFromRAM(ramGB int) int {
	if ramGB <= 0 {
		return 0
	}
	if ramGB <= 2 {
		return 1
	}
	if ramGB <= 4 {
		return 2
	}
	return 3
}

func resourceClassFromCPU(cpuCores int) int {
	if cpuCores <= 0 {
		return 0
	}
	if cpuCores <= 1 {
		return 1
	}
	if cpuCores <= 2 {
		return 2
	}
	return 3
}

func minPositiveClass(a int, b int) int {
	if a == 0 {
		return b
	}
	if b == 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
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
