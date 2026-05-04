package retention

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestPressureMonitorSnapshot_TriggersWithTwoSignals(t *testing.T) {
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	monitor := NewPressureMonitor(nil, AdaptiveRetentionOptions{
		PressureMinActiveSignals:            2,
		PressureErrorWindow:                 5 * time.Minute,
		PressureErrorThreshold:              3,
		PressureMemoryThresholdPercent:      85,
		PressureMemoryBudgetMiB:             100,
		PressureClickHouseDiskThresholdPerc: 90,
	})
	monitor.clockNow = func() time.Time { return now }
	monitor.readMemStats = func(stats *runtime.MemStats) {
		stats.Alloc = 95 * 1024 * 1024
	}
	monitor.queryDiskStats = func(context.Context) (uint64, uint64, error) {
		return 91, 100, nil
	}

	snapshot := monitor.Snapshot(context.Background())
	if !snapshot.Pressure {
		t.Fatalf("expected pressure=true")
	}
	if len(snapshot.Reasons) < 2 {
		t.Fatalf("expected at least two reasons, got %v", snapshot.Reasons)
	}
}

func TestPressureMonitorSnapshot_UsesRecoverableErrorsWindow(t *testing.T) {
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	monitor := NewPressureMonitor(nil, AdaptiveRetentionOptions{
		PressureMinActiveSignals:            1,
		PressureErrorWindow:                 5 * time.Minute,
		PressureErrorThreshold:              2,
		PressureMemoryThresholdPercent:      95,
		PressureMemoryBudgetMiB:             100,
		PressureClickHouseDiskThresholdPerc: 99,
	})
	monitor.clockNow = func() time.Time { return now }
	monitor.readMemStats = func(stats *runtime.MemStats) {
		stats.Alloc = 10 * 1024 * 1024
	}
	monitor.queryDiskStats = func(context.Context) (uint64, uint64, error) {
		return 10, 100, nil
	}

	monitor.ObserveRecoverableError("timeout")
	monitor.ObserveRecoverableError("memory limit exceeded")
	monitor.ObserveRecoverableError("custom validation error")

	snapshot := monitor.Snapshot(context.Background())
	if !snapshot.Pressure {
		t.Fatalf("expected pressure=true from query_errors")
	}
	if snapshot.RecentErrors != 2 {
		t.Fatalf("RecentErrors=%d, want 2", snapshot.RecentErrors)
	}
	if len(snapshot.Reasons) != 1 || snapshot.Reasons[0] != "query_errors" {
		t.Fatalf("Reasons=%v, want [query_errors]", snapshot.Reasons)
	}
}
