package config

import "testing"

func TestApplyDefaultsSeedsAutoTunedValues(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	// Load-sensitive knobs start at the controller floor.
	if cfg.DashboardQueryParallelism != 1 {
		t.Fatalf("DashboardQueryParallelism = %d, want 1 (floor)", cfg.DashboardQueryParallelism)
	}
	if cfg.TelemetryQueryConcurrency != 1 {
		t.Fatalf("TelemetryQueryConcurrency = %d, want 1 (floor)", cfg.TelemetryQueryConcurrency)
	}
	if cfg.ClickHouseMaxMemoryMiB != 64 {
		t.Fatalf("ClickHouseMaxMemoryMiB = %d, want 64 (floor)", cfg.ClickHouseMaxMemoryMiB)
	}
	if cfg.ClickHouseExternalGroupByMiB != 16 || cfg.ClickHouseExternalSortMiB != 16 {
		t.Fatalf("spill thresholds = %d/%d, want 16/16", cfg.ClickHouseExternalGroupByMiB, cfg.ClickHouseExternalSortMiB)
	}

	if !cfg.RetentionAdaptiveEnabled {
		t.Fatalf("RetentionAdaptiveEnabled should be true")
	}
	if !cfg.DashboardHalveOnOOM {
		t.Fatalf("DashboardHalveOnOOM should be true")
	}
	if cfg.RetentionPressureMemBudgetMB <= 0 {
		t.Fatalf("RetentionPressureMemBudgetMB should be auto-sized, got %d", cfg.RetentionPressureMemBudgetMB)
	}
	if cfg.ClickHouseMaxOpenConns < 2 || cfg.ClickHouseMaxIdleConns < 1 || cfg.ClickHouseMaxIdleConns > cfg.ClickHouseMaxOpenConns {
		t.Fatalf("invalid auto-sized conn pool open=%d idle=%d", cfg.ClickHouseMaxOpenConns, cfg.ClickHouseMaxIdleConns)
	}
	if cfg.MaxLogRetentionDays < 30 || cfg.MaxTraceRetentionDays < 15 {
		t.Fatalf("retention caps too low: logs=%d traces=%d", cfg.MaxLogRetentionDays, cfg.MaxTraceRetentionDays)
	}
}
