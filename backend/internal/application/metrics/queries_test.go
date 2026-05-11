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

func TestBuildSlowestEndpointsQuery_UsesLatencyBuckets(t *testing.T) {
	query := BuildSlowestEndpointsQuery(time.Unix(0, 0), time.Unix(3600, 0), "", 10)

	if strings.Contains(query, "duration_quantiles_state") {
		t.Fatalf("slowest endpoints query must not merge TDigest state, got: %s", query)
	}
	for _, expected := range []string{"latency_0_100", "latency_10000_inf", "arrayCumSum(bucket_counts)"} {
		if !strings.Contains(query, expected) {
			t.Fatalf("expected bucket percentile expression %q, got: %s", expected, query)
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

func TestTraceEndpointBackfillQuery_PopulatesLatencyBuckets(t *testing.T) {
	query := buildTraceEndpointBackfillQuery(time.Unix(0, 0), time.Unix(3600, 0))

	for _, expected := range []string{
		"latency_0_100",
		"latency_10000_inf",
		"countIf(duration_ms > 5000 AND duration_ms <= 10000) AS latency_5000_10000",
	} {
		if !strings.Contains(query, expected) {
			t.Fatalf("expected endpoint backfill bucket %q, got: %s", expected, query)
		}
	}
}

func TestDashboardRateQueries_UseTwoStepAggregation(t *testing.T) {
	from := time.Unix(0, 0)
	to := time.Unix(3600, 0)

	tests := []struct {
		name          string
		query         string
		expectOuter   string
		expectInner   string
		forbidPattern string
	}{
		{
			name:          "error hotspots",
			query:         BuildErrorHotspotsQuery(from, to, "", 10),
			expectOuter:   "if(total > 0, (toFloat64(errors) / toFloat64(total)) * 100, 0) AS error_rate",
			expectInner:   "sum(error_count) AS errors",
			forbidPattern: "if(sum(request_count) > 0, (sum(error_count) / sum(request_count)) * 100, 0) AS error_rate",
		},
		{
			name:          "error rate",
			query:         BuildErrorRateQuery(from, to, ""),
			expectOuter:   "if(total > 0, (toFloat64(errors) / toFloat64(total)) * 100, 0) AS error_rate",
			expectInner:   "sum(error_count) AS errors",
			forbidPattern: "if(sum(request_count) > 0, (sum(error_count) / sum(request_count)) * 100, 0) AS error_rate",
		},
		{
			name:          "error rate time series",
			query:         BuildErrorRateTimeSeriesQuery(from, to, ""),
			expectOuter:   "if(total > 0, (toFloat64(errors) / toFloat64(total)) * 100, 0) AS error_rate",
			expectInner:   "sum(error_count) AS errors",
			forbidPattern: "if(sum(request_count) > 0, (sum(error_count) / sum(request_count)) * 100, 0) AS error_rate",
		},
		{
			name:          "top endpoints throughput",
			query:         BuildTopEndpointsThroughputQuery(from, to, "", 10),
			expectOuter:   "if(requests > 0, (toFloat64(errors) / toFloat64(requests)) * 100, 0) AS error_rate",
			expectInner:   "sum(request_count) AS requests",
			forbidPattern: "if(sum(request_count) > 0, (sum(error_count) / sum(request_count)) * 100, 0) AS error_rate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.query, "FROM (") {
				t.Fatalf("expected nested subquery for two-step aggregation, got: %s", tt.query)
			}
			if !strings.Contains(tt.query, tt.expectInner) {
				t.Fatalf("expected inner aggregation %q, got: %s", tt.expectInner, tt.query)
			}
			if !strings.Contains(tt.query, tt.expectOuter) {
				t.Fatalf("expected outer rate expression %q, got: %s", tt.expectOuter, tt.query)
			}
			if strings.Contains(tt.query, tt.forbidPattern) {
				t.Fatalf("unexpected legacy nested-aggregate expression found: %s", tt.query)
			}
		})
	}
}
