package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestBuildLatencyDistributionQuery_UsesRollupTable(t *testing.T) {
	query := BuildLatencyDistributionQuery(time.Unix(0, 0), time.Unix(3600, 0), "")

	if !strings.Contains(query, traceServiceRollupTable) {
		t.Fatalf("expected service rollup table, got: %s", query)
	}
	if strings.Contains(query, "telemetry.otel_traces") {
		t.Fatalf("dashboard query must not read raw traces, got: %s", query)
	}
}

func TestBuildStatusCodeBreakdownQuery_UsesRollupTable(t *testing.T) {
	query := BuildStatusCodeBreakdownQuery(time.Unix(0, 0), time.Unix(3600, 0), "")

	if !strings.Contains(query, traceServiceRollupTable) {
		t.Fatalf("expected service rollup table, got: %s", query)
	}
	if strings.Contains(query, "telemetry.otel_traces") {
		t.Fatalf("dashboard query must not read raw traces, got: %s", query)
	}
}

func TestDashboardQueryBuilders_DoNotReadRawTelemetryTables(t *testing.T) {
	from := time.Unix(0, 0)
	to := time.Unix(3600, 0)
	builders := []string{
		BuildLatencyDistributionQuery(from, to, ""),
		BuildSlowestEndpointsQuery(from, to, "", 10),
		BuildErrorHotspotsQuery(from, to, "", 10),
		BuildApdexQuery(from, to, ""),
		BuildThroughputQuery(from, to, ""),
		BuildErrorRateQuery(from, to, ""),
		BuildLatencyPercentilesQuery(from, to, ""),
		BuildErrorRateTimeSeriesQuery(from, to, ""),
		BuildStatusCodeBreakdownQuery(from, to, ""),
		BuildTopEndpointsThroughputQuery(from, to, "", 10),
		BuildLogVolumeQuery(from, to, ""),
		BuildLogLevelsQuery(from, to, ""),
	}

	for _, query := range builders {
		if strings.Contains(query, "telemetry.otel_traces") || strings.Contains(query, "telemetry.otel_logs") {
			t.Fatalf("dashboard query must use rollups only, got: %s", query)
		}
	}
}

func TestBackfillQueryBuilders_ReadRawTelemetryTables(t *testing.T) {
	query := buildTraceServiceBackfillQuery(time.Unix(0, 0), time.Unix(3600, 0))

	if !strings.Contains(query, "telemetry.otel_traces") {
		t.Fatalf("expected backfill to read raw traces, got: %s", query)
	}
	if !strings.Contains(query, "upper(toString(SpanKind)) IN ('SERVER', 'SPAN_KIND_SERVER')") {
		t.Fatalf("expected normalized SpanKind filter, got: %s", query)
	}
}
