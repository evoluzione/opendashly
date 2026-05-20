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
