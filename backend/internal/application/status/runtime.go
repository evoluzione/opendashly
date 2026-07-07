package status

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	runtimemetrics "runtime/metrics"
	"strconv"
	"strings"
	"time"

	"opendashly/backend/internal/infrastructure/config"
)

var processStartedAt = time.Now()

type ComponentHealth struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	LatencyMs int64  `json:"latencyMs,omitempty"`
	Error     string `json:"error,omitempty"`
}

type ResourceHealth struct {
	CPUCoresAvailable     float64 `json:"cpuCoresAvailable"`
	CPUUsedPercentOneCore float64 `json:"cpuUsedPercentOneCore"`
	MemoryLimitBytes      uint64  `json:"memoryLimitBytes"`
	MemoryUsedBytes       uint64  `json:"memoryUsedBytes"`
	MemoryUsedPercent     float64 `json:"memoryUsedPercent"`
	GoHeapAllocBytes      uint64  `json:"goHeapAllocBytes"`
	GoRoutines            int     `json:"goRoutines"`
	BackendUptimeSeconds  int64   `json:"backendUptimeSeconds"`
}

type QueryHealth struct {
	RunningNow           uint64  `json:"runningNow"`
	SlowRunningNow       uint64  `json:"slowRunningNow"`
	MaxRunningElapsedSec float64 `json:"maxRunningElapsedSec"`
	SlowQueriesLast15m   uint64  `json:"slowQueriesLast15m"`
	FailedQueriesLast15m uint64  `json:"failedQueriesLast15m"`
	SlowThresholdSec     float64 `json:"slowThresholdSec"`
}

type RuntimeSummary struct {
	Ok          bool              `json:"ok"`
	GeneratedAt time.Time         `json:"generatedAt"`
	Components  []ComponentHealth `json:"components"`
	Resources   ResourceHealth    `json:"resources"`
	Queries     QueryHealth       `json:"queries"`
	Warnings    []string          `json:"warnings,omitempty"`
}

func (s *Service) Runtime(ctx context.Context) RuntimeSummary {
	slowThresholdSec := config.Auto().StatusSlowThresholdSec

	now := time.Now().UTC()
	summary := RuntimeSummary{
		Ok:          true,
		GeneratedAt: now,
		Components: []ComponentHealth{
			{Name: "backend", Status: "up"},
		},
		Queries: QueryHealth{SlowThresholdSec: slowThresholdSec},
	}
	warnings := make([]string, 0, 2)

	if s.Storage == nil {
		summary.Ok = false
		summary.Components = append(summary.Components, ComponentHealth{
			Name:   "database",
			Status: "down",
			Error:  "storage not configured",
		})
		warnings = append(warnings, "Storage non configurato: metriche database non disponibili")
	} else {
		summary.Components = append(summary.Components, s.databaseHealth(ctx))
		queryCtx, cancel := context.WithTimeout(ctx, time.Duration(config.Auto().StatusRuntimeQueryTimeoutMS)*time.Millisecond)
		queries, queryWarnings, queryErr := s.queryHealth(queryCtx, slowThresholdSec)
		cancel()
		if queryErr != nil {
			warnings = append(warnings, fmt.Sprintf("Diagnostica query non disponibile: %v", queryErr))
		} else {
			summary.Queries = queries
		}
		warnings = append(warnings, queryWarnings...)
	}

	collector := s.collectorHealth(ctx)
	summary.Components = append(summary.Components, collector)

	for _, component := range summary.Components {
		if component.Status == "down" {
			summary.Ok = false
			break
		}
	}

	summary.Resources = collectResourceHealth(now)
	if len(warnings) > 0 {
		summary.Warnings = warnings
	}
	return summary
}

func (s *Service) databaseHealth(ctx context.Context) ComponentHealth {
	startedAt := time.Now()
	if err := s.getDatabasePinger()(ctx); err != nil {
		return ComponentHealth{
			Name:      "database",
			Status:    "down",
			LatencyMs: time.Since(startedAt).Milliseconds(),
			Error:     err.Error(),
		}
	}
	return ComponentHealth{
		Name:      "database",
		Status:    "up",
		LatencyMs: time.Since(startedAt).Milliseconds(),
	}
}

func (s *Service) collectorHealth(ctx context.Context) ComponentHealth {
	url := strings.TrimSpace(s.CollectorHealthURL)
	if url == "" {
		url = "http://otel-collector:13133"
	}

	candidates := []string{
		strings.TrimRight(url, "/"),
		strings.TrimRight(url, "/") + "/health/status",
	}

	httpClient := &http.Client{Timeout: time.Duration(config.Auto().StatusCollectorTimeoutMS) * time.Millisecond}
	for _, endpoint := range candidates {
		startedAt := time.Now()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return ComponentHealth{
				Name:      "collector",
				Status:    "up",
				LatencyMs: time.Since(startedAt).Milliseconds(),
			}
		}
	}

	return ComponentHealth{
		Name:   "collector",
		Status: "down",
		Error:  "collector health endpoint unreachable",
	}
}

func (s *Service) queryHealth(ctx context.Context, slowThresholdSec float64) (QueryHealth, []string, error) {
	if s.Storage == nil || s.Storage.Conn == nil {
		return QueryHealth{}, nil, fmt.Errorf("storage not configured")
	}

	queries := QueryHealth{SlowThresholdSec: slowThresholdSec}
	warnings := make([]string, 0, 1)

	const runningQuery = `
		SELECT
			count() AS running,
			countIf(elapsed >= ?) AS slow_running,
			max(elapsed) AS max_elapsed
		FROM system.processes
		WHERE query NOT ILIKE '%system.processes%'
			AND query NOT ILIKE '%system.query_log%'
	`
	row := s.Storage.Conn.QueryRow(ctx, runningQuery, slowThresholdSec)
	if err := row.Scan(&queries.RunningNow, &queries.SlowRunningNow, &queries.MaxRunningElapsedSec); err != nil {
		return queries, warnings, err
	}

	const last15mQuery = `
		SELECT
			countIf(query_duration_ms >= ? * 1000 AND type = 'QueryFinish') AS slow_done,
			countIf(type = 'ExceptionBeforeStart' OR type = 'ExceptionWhileProcessing') AS failed
		FROM system.query_log
		WHERE event_time >= now() - INTERVAL 15 MINUTE
	`
	row = s.Storage.Conn.QueryRow(ctx, last15mQuery, slowThresholdSec)
	if err := row.Scan(&queries.SlowQueriesLast15m, &queries.FailedQueriesLast15m); err != nil {
		warnings = append(warnings, "system.query_log non disponibile o disabilitato")
	}

	return queries, warnings, nil
}

func collectResourceHealth(now time.Time) ResourceHealth {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	memoryLimit := readMemoryLimitBytes()
	memoryUsed := readMemoryCurrentBytes()
	if memoryUsed == 0 {
		memoryUsed = memStats.Alloc
	}

	memoryUsedPercent := 0.0
	if memoryLimit > 0 {
		memoryUsedPercent = float64(memoryUsed) / float64(memoryLimit) * 100
	}

	cpuSeconds := readCPUSeconds()
	uptimeSeconds := now.Sub(processStartedAt).Seconds()
	cpuPercentOfCore := 0.0
	if uptimeSeconds > 0 {
		cpuPercentOfCore = (cpuSeconds / uptimeSeconds) * 100
	}

	cpuCores := readCPUCoresLimit()
	if cpuCores <= 0 {
		cpuCores = float64(runtime.NumCPU())
	}

	return ResourceHealth{
		CPUCoresAvailable:     cpuCores,
		CPUUsedPercentOneCore: cpuPercentOfCore,
		MemoryLimitBytes:      memoryLimit,
		MemoryUsedBytes:       memoryUsed,
		MemoryUsedPercent:     memoryUsedPercent,
		GoHeapAllocBytes:      memStats.Alloc,
		GoRoutines:            runtime.NumGoroutine(),
		BackendUptimeSeconds:  int64(uptimeSeconds),
	}
}

func readCPUSeconds() float64 {
	samples := []runtimemetrics.Sample{{Name: "/cpu/classes/user:cpu-seconds"}, {Name: "/cpu/classes/system:cpu-seconds"}}
	runtimemetrics.Read(samples)
	cpu := 0.0
	for _, sample := range samples {
		if sample.Value.Kind() == runtimemetrics.KindFloat64 {
			cpu += sample.Value.Float64()
		}
	}
	return cpu
}

func readCPUCoresLimit() float64 {
	if raw, err := os.ReadFile("/sys/fs/cgroup/cpu.max"); err == nil {
		parts := strings.Fields(strings.TrimSpace(string(raw)))
		if len(parts) == 2 {
			if parts[0] == "max" {
				return float64(runtime.NumCPU())
			}
			quota, qErr := strconv.ParseFloat(parts[0], 64)
			period, pErr := strconv.ParseFloat(parts[1], 64)
			if qErr == nil && pErr == nil && period > 0 {
				return quota / period
			}
		}
	}

	quotaRaw, qErr := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_quota_us")
	periodRaw, pErr := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_period_us")
	if qErr == nil && pErr == nil {
		quota, qParseErr := strconv.ParseFloat(strings.TrimSpace(string(quotaRaw)), 64)
		period, pParseErr := strconv.ParseFloat(strings.TrimSpace(string(periodRaw)), 64)
		if qParseErr == nil && pParseErr == nil && quota > 0 && period > 0 {
			return quota / period
		}
	}

	return float64(runtime.NumCPU())
}

func readMemoryLimitBytes() uint64 {
	paths := []string{
		"/sys/fs/cgroup/memory.max",
		"/sys/fs/cgroup/memory/memory.limit_in_bytes",
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		value := strings.TrimSpace(string(raw))
		if value == "" || value == "max" {
			continue
		}
		parsed, parseErr := strconv.ParseUint(value, 10, 64)
		if parseErr == nil {
			// Ignore unrealistic values used by some hosts to mean "unlimited".
			if parsed > 1<<62 {
				return 0
			}
			return parsed
		}
	}
	return 0
}

func readMemoryCurrentBytes() uint64 {
	paths := []string{
		"/sys/fs/cgroup/memory.current",
		"/sys/fs/cgroup/memory/memory.usage_in_bytes",
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		parsed, parseErr := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64)
		if parseErr == nil {
			return parsed
		}
	}
	return 0
}
