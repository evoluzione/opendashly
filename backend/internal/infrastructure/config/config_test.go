package config

import "testing"

func TestApplyDefaultsSeedsResilientValues(t *testing.T) {
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

	// Generous, resilient retention caps (adaptive step-down protects them).
	if cfg.MaxLogRetentionDays != 30 {
		t.Fatalf("MaxLogRetentionDays = %d, want 30", cfg.MaxLogRetentionDays)
	}
	if cfg.MaxTraceRetentionDays != 15 {
		t.Fatalf("MaxTraceRetentionDays = %d, want 15", cfg.MaxTraceRetentionDays)
	}

	// A few static safety defaults.
	if !cfg.RetentionAdaptiveEnabled {
		t.Fatalf("RetentionAdaptiveEnabled should be true")
	}
	if !cfg.DashboardHalveOnOOM {
		t.Fatalf("DashboardHalveOnOOM should be true")
	}
	if cfg.ClickHouseMaxOpenConns != 4 || cfg.ClickHouseMaxIdleConns != 2 {
		t.Fatalf("conn pool = %d/%d, want 4/2", cfg.ClickHouseMaxOpenConns, cfg.ClickHouseMaxIdleConns)
	}
}
