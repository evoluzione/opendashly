package config

import (
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// ResourceProfile is the resource envelope detected from the runtime container.
// Values are best-effort: zero means the platform did not expose a limit.
type ResourceProfile struct {
	CPUCores  int
	MemoryMiB int
}

// AutoSettings contains every non-secret operational knob that used to be
// supplied by environment variables. It is derived locally from cgroup/container
// resources plus conservative safety policies, then refined at runtime by the
// adaptive controllers where live feedback is required.
type AutoSettings struct {
	Resources ResourceProfile

	ServiceListTimeoutSec  int
	CleanupIntervalMinutes int

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
	RetentionPreCount            bool

	ClickHouseTempDiskMiB        int
	ClickHouseMaxExecSec         int
	ClickHouseMaxOpenConns       int
	ClickHouseMaxIdleConns       int
	ClickHouseDialTimeoutSec     int
	ClickHouseReadTimeoutSec     int
	ClickHouseMaxMemoryMiB       int
	ClickHouseExternalGroupByMiB int
	ClickHouseExternalSortMiB    int

	DashboardFreshCacheTTLSec       int
	DashboardStaleCacheTTLSec       int
	DashboardRequestTimeoutSec      int
	DashboardQueryParallelism       int
	DashboardHalveOnOOM             bool
	DashboardRawFallback            bool
	DashboardPressureCooldownSec    int
	DashboardLastGoodTTLSec         int
	DashboardRollupBackfill         bool
	DashboardRollupBackfillHours    int
	DashboardRollupChunkMinutes     int
	DashboardRollupBackfillDelaySec int

	TelemetryQueryConcurrency int

	QueryRequestTimeoutSec     int
	QueryAttributesTimeoutSec  int
	QueryServicesCacheTTLSec   int
	QueryAttributesCacheTTLSec int

	AuthUserLookupTimeoutMS int

	StatusPingTimeoutMS            int
	StatusCountsTimeoutMS          int
	StatusSeriesTimeoutMS          int
	StatusRuntimeQueryTimeoutMS    int
	StatusCollectorTimeoutMS       int
	StatusSlowThresholdSec         float64
	StatusHealthMaxExecSec         int
	StatusRollupBackfillMaxExecSec int

	APIReadTimeoutSec  int
	APIWriteTimeoutSec int
	APIIdleTimeoutSec  int

	TuningIntervalSec   int
	TuningCalmWindowSec int

	LogAttributeKeysMaxExecSec int
}

var (
	autoOnce sync.Once
	autoCfg  AutoSettings
)

// Auto returns the process-wide auto-derived operational configuration.
func Auto() AutoSettings {
	autoOnce.Do(func() { autoCfg = deriveAutoSettings(detectResources()) })
	return autoCfg
}

func deriveAutoSettings(r ResourceProfile) AutoSettings {
	cpu := r.CPUCores
	if cpu <= 0 {
		cpu = runtime.NumCPU()
	}
	if cpu <= 0 {
		cpu = 1
	}
	mem := r.MemoryMiB
	if mem <= 0 {
		mem = 1024
	}

	r.CPUCores = cpu
	r.MemoryMiB = mem

	small := cpu <= 1 || mem <= 1024
	large := cpu >= 4 && mem >= 4096

	chMemFloor := 64
	spill := 16
	openConns := clampInt(cpu*2, 2, 8)
	idleConns := clampInt(cpu, 1, openConns)
	queryTimeout := 10
	dashboardTimeout := 20
	if small {
		openConns = 2
		idleConns = 1
		queryTimeout = 12
		dashboardTimeout = 24
	}

	settings := AutoSettings{
		Resources: r,

		ServiceListTimeoutSec:  dashboardTimeout,
		CleanupIntervalMinutes: 720,

		RetentionAdaptiveEnabled:     true,
		MaxLogRetentionDays:          30,
		MaxTraceRetentionDays:        15,
		RetentionStepDownDays:        2,
		RetentionMaxLevel:            2,
		RetentionMinTraceHours:       24,
		RetentionMinLogHours:         24,
		RetentionPressureCooldownSec: 300,
		RetentionPressureMinSignals:  2,
		RetentionPressureWindowSec:   300,
		RetentionPressureErrorCount:  4,
		RetentionPressureMemPct:      85,
		RetentionPressureMemBudgetMB: clampInt(mem*70/100, 128, 2048),
		RetentionPressureDiskPct:     90,
		RetentionPreCount:            false,

		ClickHouseTempDiskMiB:        clampInt(mem, 512, 4096),
		ClickHouseMaxExecSec:         20,
		ClickHouseMaxOpenConns:       openConns,
		ClickHouseMaxIdleConns:       idleConns,
		ClickHouseDialTimeoutSec:     5,
		ClickHouseReadTimeoutSec:     maxInt(30, dashboardTimeout+10),
		ClickHouseMaxMemoryMiB:       chMemFloor,
		ClickHouseExternalGroupByMiB: spill,
		ClickHouseExternalSortMiB:    spill,

		DashboardFreshCacheTTLSec:       30,
		DashboardStaleCacheTTLSec:       900,
		DashboardRequestTimeoutSec:      dashboardTimeout,
		DashboardQueryParallelism:       1,
		DashboardHalveOnOOM:             true,
		DashboardRawFallback:            false,
		DashboardPressureCooldownSec:    60,
		DashboardLastGoodTTLSec:         1800,
		DashboardRollupBackfill:         true,
		DashboardRollupBackfillHours:    48,
		DashboardRollupChunkMinutes:     15,
		DashboardRollupBackfillDelaySec: 30,

		TelemetryQueryConcurrency: 1,

		QueryRequestTimeoutSec:     queryTimeout,
		QueryAttributesTimeoutSec:  5,
		QueryServicesCacheTTLSec:   1800,
		QueryAttributesCacheTTLSec: 300,

		AuthUserLookupTimeoutMS: 750,

		StatusPingTimeoutMS:            750,
		StatusCountsTimeoutMS:          1200,
		StatusSeriesTimeoutMS:          1200,
		StatusRuntimeQueryTimeoutMS:    750,
		StatusCollectorTimeoutMS:       1500,
		StatusSlowThresholdSec:         2.0,
		StatusHealthMaxExecSec:         2,
		StatusRollupBackfillMaxExecSec: 5,

		APIReadTimeoutSec:  queryTimeout,
		APIWriteTimeoutSec: maxInt(30, dashboardTimeout+6),
		APIIdleTimeoutSec:  60,

		TuningIntervalSec:   20,
		TuningCalmWindowSec: 60,

		LogAttributeKeysMaxExecSec: 3,
	}

	if small {
		settings.DashboardRollupBackfillHours = 24
		settings.DashboardRollupChunkMinutes = 10
		settings.ClickHouseMaxExecSec = 16
	}
	if large {
		settings.MaxLogRetentionDays = 45
		settings.MaxTraceRetentionDays = 21
		settings.DashboardRollupBackfillHours = 72
		settings.DashboardRollupChunkMinutes = 30
	}
	return settings
}

func detectResources() ResourceProfile {
	return ResourceProfile{
		CPUCores:  detectCPUCores(),
		MemoryMiB: detectMemoryMiB(),
	}
}

func detectCPUCores() int {
	host := runtime.NumCPU()
	quota := detectCPUQuotaCores()
	if quota > 0 {
		if host > 0 {
			return minInt(host, quota)
		}
		return quota
	}
	return host
}

func detectCPUQuotaCores() int {
	if data, err := os.ReadFile("/sys/fs/cgroup/cpu.max"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 2 && fields[0] != "max" {
			quota, qerr := strconv.ParseFloat(fields[0], 64)
			period, perr := strconv.ParseFloat(fields[1], 64)
			if qerr == nil && perr == nil && quota > 0 && period > 0 {
				return maxInt(1, int(math.Ceil(quota/period)))
			}
		}
	}
	quotaBytes, qerr := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_quota_us")
	periodBytes, perr := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_period_us")
	if qerr == nil && perr == nil {
		quota, qerr := strconv.ParseFloat(strings.TrimSpace(string(quotaBytes)), 64)
		period, perr := strconv.ParseFloat(strings.TrimSpace(string(periodBytes)), 64)
		if qerr == nil && perr == nil && quota > 0 && period > 0 {
			return maxInt(1, int(math.Ceil(quota/period)))
		}
	}
	return 0
}

func detectMemoryMiB() int {
	for _, path := range []string{
		"/sys/fs/cgroup/memory.max",
		"/sys/fs/cgroup/memory/memory.limit_in_bytes",
	} {
		if data, err := os.ReadFile(path); err == nil {
			raw := strings.TrimSpace(string(data))
			if raw == "" || raw == "max" {
				continue
			}
			bytes, err := strconv.ParseUint(raw, 10, 64)
			if err == nil && bytes > 0 && bytes < (1<<60) {
				return int(bytes / 1024 / 1024)
			}
		}
	}
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, err := strconv.ParseUint(fields[1], 10, 64)
					if err == nil && kb > 0 {
						return int(kb / 1024)
					}
				}
			}
		}
	}
	return 0
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
