package status

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"opendashly/backend/internal/infrastructure/storage"
)

func TestStatusSummaryQueriesDoNotScanRawTelemetryTables(t *testing.T) {
	queries := []string{
		logsCountsQuery,
		tracesCountsQuery,
		logsSeriesQuery,
		tracesSeriesQuery,
	}

	for _, query := range queries {
		lower := strings.ToLower(query)
		if strings.Contains(lower, "from telemetry.otel_logs") {
			t.Fatalf("status summary query must not scan raw logs table: %s", query)
		}
		if strings.Contains(lower, "from telemetry.otel_traces") {
			t.Fatalf("status summary query must not scan raw traces table: %s", query)
		}
	}
}

func TestDatabaseProbeRunsThroughMemoryTracker(t *testing.T) {
	// A bare protocol ping (or an untracked SELECT 1) stays green while the
	// server rejects every real query with code 241, so the probe must be a
	// real query forced through the memory tracker.
	lower := strings.ToLower(databaseProbeQuery)
	if !strings.Contains(lower, "max_untracked_memory = 0") {
		t.Fatalf("database probe must disable untracked memory, got %q", databaseProbeQuery)
	}
	if !strings.Contains(lower, "select") {
		t.Fatalf("database probe must execute a real query, got %q", databaseProbeQuery)
	}
}

func TestRuntimeDatabaseHealthUsesInjectedProbe(t *testing.T) {
	cases := []struct {
		name       string
		probeErr   error
		wantStatus string
	}{
		{name: "probe failure marks database down", probeErr: errors.New("code: 241, memory limit exceeded"), wantStatus: "down"},
		{name: "probe success marks database up", probeErr: nil, wantStatus: "up"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &Service{
				Storage: &storage.Client{},
				// Unroutable port so the collector probe fails fast in tests.
				CollectorHealthURL: "http://127.0.0.1:1",
				pingDatabase: func(context.Context) error {
					return tc.probeErr
				},
				countsFetcher: func(context.Context, *storage.Client, string) (TelemetryCounts, error) {
					return TelemetryCounts{}, nil
				},
				seriesFetcher: func(context.Context, *storage.Client, string, time.Time, int, int) ([]TelemetryPoint, error) {
					return nil, nil
				},
			}

			summary := svc.Runtime(context.Background())

			var database *ComponentHealth
			for i := range summary.Components {
				if summary.Components[i].Name == "database" {
					database = &summary.Components[i]
				}
			}
			if database == nil {
				t.Fatalf("runtime summary missing database component: %+v", summary.Components)
			}
			if database.Status != tc.wantStatus {
				t.Fatalf("database status = %q, want %q (err %q)", database.Status, tc.wantStatus, database.Error)
			}
		})
	}
}

func TestSummaryKeepsDatabaseOkWhenCountFetchFails(t *testing.T) {
	failure := errors.New("memory limit exceeded: OvercommitTracker")
	svc := &Service{
		Storage: &storage.Client{},
		pingDatabase: func(context.Context) error {
			return nil
		},
		countsFetcher: func(context.Context, *storage.Client, string) (TelemetryCounts, error) {
			return TelemetryCounts{}, failure
		},
		seriesFetcher: func(context.Context, *storage.Client, string, time.Time, int, int) ([]TelemetryPoint, error) {
			return []TelemetryPoint{}, nil
		},
	}

	summary := svc.Summary(context.Background())

	if !summary.Ok {
		t.Fatalf("summary should stay ok when database ping succeeds; got error=%q warnings=%v", summary.Error, summary.Warnings)
	}
	if !summary.Checks.Database {
		t.Fatal("database check should remain true")
	}
	if summary.Error != "" {
		t.Fatalf("recoverable count failure should be a warning, got error %q", summary.Error)
	}
	if len(summary.Warnings) == 0 || !strings.Contains(summary.Warnings[0], "logs counts failed") {
		t.Fatalf("expected warning for count failure, got %v", summary.Warnings)
	}
}
