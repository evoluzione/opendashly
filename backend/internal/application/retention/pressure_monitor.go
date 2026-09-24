package retention

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type PressureMonitor struct {
	conn driver.Conn
	opts AdaptiveRetentionOptions

	mu            sync.Mutex
	recentErrors  []time.Time
	lastSnapshot  PressureSnapshot
	clockNow      func() time.Time
	readMemStats  func(*runtime.MemStats)
	queryDiskStats func(context.Context) (uint64, uint64, error)
}

func NewPressureMonitor(conn driver.Conn, opts AdaptiveRetentionOptions) *PressureMonitor {
	m := &PressureMonitor{
		conn:         conn,
		opts:         opts,
		clockNow:     time.Now,
		readMemStats: runtime.ReadMemStats,
	}
	m.queryDiskStats = m.loadDiskStats
	return m
}

func (m *PressureMonitor) ObserveRecoverableError(msg string) {
	if !isRecoverablePressureError(msg) {
		return
	}
	now := m.clockNow().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recentErrors = append(m.recentErrors, now)
	m.trimErrorsLocked(now)
}

func (m *PressureMonitor) Snapshot(ctx context.Context) PressureSnapshot {
	now := m.clockNow().UTC()
	memoryUsageMiB, memoryThresholdMiB, memoryPct := m.memoryPressure(now)
	diskUsed, diskTotal, diskPct := m.diskPressure(ctx)

	m.mu.Lock()
	m.trimErrorsLocked(now)
	recentErrors := len(m.recentErrors)
	m.mu.Unlock()

	reasons := make([]string, 0, 3)
	activeSignals := 0

	if memoryThresholdMiB > 0 && memoryPct >= float64(m.opts.PressureMemoryThresholdPercent) {
		reasons = append(reasons, "backend_memory")
		activeSignals++
	}
	if diskTotal > 0 && diskPct >= float64(m.opts.PressureClickHouseDiskThresholdPerc) {
		reasons = append(reasons, "clickhouse_disk")
		activeSignals++
	}
	if m.opts.PressureErrorThreshold > 0 && recentErrors >= m.opts.PressureErrorThreshold {
		reasons = append(reasons, "query_errors")
		activeSignals++
	}

	sort.Strings(reasons)
	pressure := activeSignals >= max(1, m.opts.PressureMinActiveSignals)

	snapshot := PressureSnapshot{
		At:               now,
		MemoryUsageMiB:   memoryUsageMiB,
		MemoryThreshold:  memoryThresholdMiB,
		MemoryPercent:    memoryPct,
		DiskUsedBytes:    diskUsed,
		DiskTotalBytes:   diskTotal,
		DiskUsagePercent: diskPct,
		RecentErrors:     recentErrors,
		Reasons:          reasons,
		Pressure:         pressure,
	}

	m.mu.Lock()
	m.lastSnapshot = snapshot
	m.mu.Unlock()

	return snapshot
}

func (m *PressureMonitor) LastSnapshot() PressureSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastSnapshot
}

func (m *PressureMonitor) memoryPressure(now time.Time) (uint64, uint64, float64) {
	if m.opts.PressureMemoryBudgetMiB <= 0 {
		return 0, 0, 0
	}
	var stats runtime.MemStats
	m.readMemStats(&stats)
	usageMiB := bytesToMiB(stats.Alloc)
	thresholdMiB := uint64(m.opts.PressureMemoryBudgetMiB)
	if thresholdMiB == 0 {
		return usageMiB, 0, 0
	}
	pct := (float64(usageMiB) / float64(thresholdMiB)) * 100.0
	_ = now
	return usageMiB, thresholdMiB, pct
}

func (m *PressureMonitor) diskPressure(ctx context.Context) (uint64, uint64, float64) {
	used, total, err := m.queryDiskStats(ctx)
	if err != nil || total == 0 {
		return used, total, 0
	}
	pct := (float64(used) / float64(total)) * 100.0
	return used, total, pct
}

func (m *PressureMonitor) loadDiskStats(ctx context.Context) (uint64, uint64, error) {
	if m.conn == nil {
		return 0, 0, nil
	}
	// UInt64 - UInt64 is Int64 in ClickHouse; the uncast scan always failed, so
	// disk pressure was never detected.
	query := `SELECT toUInt64(sum(total_space - free_space)), sum(total_space)
		FROM system.disks
		WHERE total_space > 0`
	var used uint64
	var total uint64
	if err := m.conn.QueryRow(ctx, query).Scan(&used, &total); err != nil {
		return 0, 0, fmt.Errorf("query disk stats: %w", err)
	}
	return used, total, nil
}

func (m *PressureMonitor) trimErrorsLocked(now time.Time) {
	if len(m.recentErrors) == 0 {
		return
	}
	window := m.opts.PressureErrorWindow
	if window <= 0 {
		window = 5 * time.Minute
	}
	cutoff := now.Add(-window)
	idx := 0
	for idx < len(m.recentErrors) && m.recentErrors[idx].Before(cutoff) {
		idx++
	}
	if idx > 0 {
		m.recentErrors = append([]time.Time(nil), m.recentErrors[idx:]...)
	}
}

func isRecoverablePressureError(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return false
	}
	patterns := []string{
		"memory limit exceeded",
		"overcommittracker",
		"deadline exceeded",
		"timeout",
		"temporarily unavailable",
	}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

func bytesToMiB(v uint64) uint64 {
	return v / (1024 * 1024)
}
