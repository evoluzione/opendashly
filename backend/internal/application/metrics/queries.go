package metrics

import (
	"fmt"
	"strings"
	"time"
)

const (
	// ApdexThresholdMs is the target response time for APDEX calculation (T)
	ApdexThresholdMs = 2000
	// ApdexToleratingMultiplier is the multiplier for tolerating threshold (4T)
	ApdexToleratingMultiplier = 4
)

// querySettings keeps every dashboard query inside a tight memory envelope so
// ClickHouse can run them on 1-2 GB droplets without tripping the
// OvercommitTracker. External GROUP BY / sort spill to disk well before the
// 150 MiB ceiling so hash tables never dominate RAM.
const querySettings = ` SETTINGS
	max_memory_usage = 157286400,
	max_bytes_before_external_group_by = 33554432,
	max_bytes_before_external_sort = 33554432,
	group_by_two_level_threshold = 10000,
	group_by_two_level_threshold_bytes = 33554432,
	max_threads = 1,
	distributed_aggregation_memory_efficient = 1,
	optimize_aggregation_in_order = 1`

// Latency bucket definitions in milliseconds
var latencyBuckets = []struct {
	Start int
	End   int
	Label string
}{
	{0, 100, "0-100ms"},
	{100, 250, "100-250ms"},
	{250, 500, "250-500ms"},
	{500, 1000, "500ms-1s"},
	{1000, 2000, "1-2s"},
	{2000, 5000, "2-5s"},
	{5000, 10000, "5-10s"},
	{10000, -1, ">10s"},
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func serviceFilter(serviceName string) string {
	if serviceName == "" {
		return ""
	}
	escaped := strings.ReplaceAll(serviceName, "'", "\\'")
	return fmt.Sprintf(" AND ServiceName = '%s'", escaped)
}

func serverSpanFilter() string {
	return `
			AND (
				upper(toString(SpanKind)) IN ('SERVER', 'SPAN_KIND_SERVER')
				OR toString(SpanKind) = '2'
			)`
}

func errorStatusClause() string {
	return "toString(StatusCode) IN ('Error', '2', 'STATUS_CODE_ERROR')"
}

func endpointSpanFilter() string {
	return "(startsWith(SpanName, 'GET ') OR startsWith(SpanName, 'POST ') OR startsWith(SpanName, 'PUT ') OR startsWith(SpanName, 'PATCH ') OR startsWith(SpanName, 'DELETE ') OR startsWith(SpanName, 'OPTIONS ') OR startsWith(SpanName, 'HEAD '))"
}

func timeBucketInterval(from, to time.Time) string {
	duration := to.Sub(from)
	interval := "1 MINUTE"
	if duration > 24*time.Hour {
		interval = "1 HOUR"
	} else if duration > 6*time.Hour {
		interval = "15 MINUTE"
	} else if duration > 1*time.Hour {
		interval = "5 MINUTE"
	}
	return interval
}

// BuildLatencyDistributionQuery builds a query to get latency distribution histogram.
func BuildLatencyDistributionQuery(from, to time.Time, serviceName string) string {
	return fmt.Sprintf(`
		SELECT
			CAST(multiIf(
				Duration/1000000 <= 100, 0,
				Duration/1000000 <= 250, 100,
				Duration/1000000 <= 500, 250,
				Duration/1000000 <= 1000, 500,
				Duration/1000000 <= 2000, 1000,
				Duration/1000000 <= 5000, 2000,
				Duration/1000000 <= 10000, 5000,
				10000
			) AS Int32) AS bucket_start,
			CAST(multiIf(
				Duration/1000000 <= 100, 100,
				Duration/1000000 <= 250, 250,
				Duration/1000000 <= 500, 500,
				Duration/1000000 <= 1000, 1000,
				Duration/1000000 <= 2000, 2000,
				Duration/1000000 <= 5000, 5000,
				Duration/1000000 <= 10000, 10000,
				-1
			) AS Int32) AS bucket_end,
			count() AS cnt
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
				%s
				%s
		GROUP BY bucket_start, bucket_end
		ORDER BY bucket_start
		%s
		`, formatTime(from), formatTime(to), serverSpanFilter(), serviceFilter(serviceName), querySettings)
}

// BuildSlowestEndpointsQuery builds a query to get the slowest endpoints by P95.
// quantileTDigest is memory-bounded (O(1) state per group) so it survives high
// SpanName cardinality without exploding the aggregation hash table.
func BuildSlowestEndpointsQuery(from, to time.Time, serviceName string, limit int) string {
	return fmt.Sprintf(`
		SELECT
			SpanName AS endpoint,
			ServiceName AS service,
			avg(Duration/1000000) AS avg_ms,
			toFloat64(quantileTDigest(0.50)(Duration/1000000)) AS p50,
			toFloat64(quantileTDigest(0.95)(Duration/1000000)) AS p95,
			toFloat64(quantileTDigest(0.99)(Duration/1000000)) AS p99,
			count() AS cnt
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
				AND %s
				%s
				%s
		GROUP BY endpoint, service
		HAVING cnt >= 5
		ORDER BY p95 DESC
		LIMIT %d
		%s
		`, formatTime(from), formatTime(to), endpointSpanFilter(), serverSpanFilter(), serviceFilter(serviceName), limit, querySettings)
}

// BuildErrorHotspotsQuery builds a query to get endpoints with highest error rates.
// StatusCode can be stored as String ('Error') or as number, so we check both
func BuildErrorHotspotsQuery(from, to time.Time, serviceName string, limit int) string {
	return fmt.Sprintf(`
		SELECT
			SpanName AS endpoint,
			ServiceName AS service,
			countIf(%s) AS error_count,
			count() AS total_count,
			if(count() > 0, (countIf(%s) / count()) * 100, 0) AS error_rate
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
				AND %s
				%s
				%s
		GROUP BY endpoint, service
		HAVING total_count >= 5 AND error_count > 0
		ORDER BY error_rate DESC, error_count DESC
		LIMIT %d
		%s
		`, errorStatusClause(), errorStatusClause(), formatTime(from), formatTime(to), endpointSpanFilter(), serverSpanFilter(), serviceFilter(serviceName), limit, querySettings)
}

// BuildApdexQuery builds a query to calculate APDEX score.
func BuildApdexQuery(from, to time.Time, serviceName string) string {
	toleratingThreshold := ApdexThresholdMs * ApdexToleratingMultiplier
	return fmt.Sprintf(`
		SELECT
			countIf(Duration/1000000 <= %d) AS satisfied,
			countIf(Duration/1000000 > %d AND Duration/1000000 <= %d) AS tolerating,
			countIf(Duration/1000000 > %d) AS frustrated,
			count() AS total
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
				%s
				%s
		%s
		`, ApdexThresholdMs, ApdexThresholdMs, toleratingThreshold, toleratingThreshold,
		formatTime(from), formatTime(to), serverSpanFilter(), serviceFilter(serviceName), querySettings)
}

// BuildThroughputQuery builds a query to get throughput time series.
func BuildThroughputQuery(from, to time.Time, serviceName string) string {
	interval := timeBucketInterval(from, to)

	return fmt.Sprintf(`
		SELECT
			toStartOfInterval(Timestamp, INTERVAL %s) AS bucket,
			count() AS request_count,
			countIf(%s) AS error_count
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
				%s
				%s
		GROUP BY bucket
		ORDER BY bucket
		%s
		`, interval, errorStatusClause(), formatTime(from), formatTime(to), serverSpanFilter(), serviceFilter(serviceName), querySettings)
}

// BuildErrorRateQuery builds a query to get overall error rate.
func BuildErrorRateQuery(from, to time.Time, serviceName string) string {
	return fmt.Sprintf(`
		SELECT
			countIf(%s) AS error_count,
			count() AS total_count,
			if(count() > 0, (countIf(%s) / count()) * 100, 0) AS error_rate
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
				%s
				%s
		%s
		`, errorStatusClause(), errorStatusClause(), formatTime(from), formatTime(to), serverSpanFilter(), serviceFilter(serviceName), querySettings)
}

// BuildLatencyPercentilesQuery builds a query for latency percentiles over time.
func BuildLatencyPercentilesQuery(from, to time.Time, serviceName string) string {
	interval := timeBucketInterval(from, to)
	return fmt.Sprintf(`
		SELECT
			toStartOfInterval(Timestamp, INTERVAL %s) AS bucket,
			toFloat64(quantileTDigest(0.50)(Duration/1000000)) AS p50,
			toFloat64(quantileTDigest(0.95)(Duration/1000000)) AS p95,
			toFloat64(quantileTDigest(0.99)(Duration/1000000)) AS p99
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
			%s
			%s
		GROUP BY bucket
		ORDER BY bucket
		%s
	`, interval, formatTime(from), formatTime(to), serverSpanFilter(), serviceFilter(serviceName), querySettings)
}

// BuildErrorRateTimeSeriesQuery builds a query for error rate over time.
func BuildErrorRateTimeSeriesQuery(from, to time.Time, serviceName string) string {
	interval := timeBucketInterval(from, to)
	return fmt.Sprintf(`
		SELECT
			toStartOfInterval(Timestamp, INTERVAL %s) AS bucket,
			countIf(%s) AS error_count,
			count() AS total_count,
			if(count() > 0, (countIf(%s) / count()) * 100, 0) AS error_rate
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
			%s
			%s
		GROUP BY bucket
		ORDER BY bucket
		%s
	`, interval, errorStatusClause(), errorStatusClause(), formatTime(from), formatTime(to), serverSpanFilter(), serviceFilter(serviceName), querySettings)
}

// BuildStatusCodeBreakdownQuery builds a query for status code distribution.
func BuildStatusCodeBreakdownQuery(from, to time.Time, serviceName string) string {
	return fmt.Sprintf(`
		SELECT
			multiIf(
				toString(StatusCode) IN ('Ok', 'STATUS_CODE_OK', '1'), 'ok',
				toString(StatusCode) IN ('Error', 'STATUS_CODE_ERROR', '2'), 'error',
				toString(StatusCode) IN ('Unset', 'STATUS_CODE_UNSET', '0'), 'unset',
				'other'
			) AS code,
			count() AS total
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
			%s
			%s
		GROUP BY code
		ORDER BY total DESC
		%s
	`, formatTime(from), formatTime(to), serverSpanFilter(), serviceFilter(serviceName), querySettings)
}

// BuildTopEndpointsThroughputQuery builds a query for top endpoints by throughput.
func BuildTopEndpointsThroughputQuery(from, to time.Time, serviceName string, limit int) string {
	return fmt.Sprintf(`
		SELECT
			SpanName AS endpoint,
			ServiceName AS service,
			count() AS request_count,
			countIf(%s) AS error_count,
			if(count() > 0, (countIf(%s) / count()) * 100, 0) AS error_rate
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
			AND %s
			%s
			%s
		GROUP BY endpoint, service
		ORDER BY request_count DESC
		LIMIT %d
		%s
	`, errorStatusClause(), errorStatusClause(), formatTime(from), formatTime(to), endpointSpanFilter(), serverSpanFilter(), serviceFilter(serviceName), limit, querySettings)
}

// BuildLogVolumeQuery builds a query for log volume over time.
func BuildLogVolumeQuery(from, to time.Time, serviceName string) string {
	interval := timeBucketInterval(from, to)
	return fmt.Sprintf(`
		SELECT
			toStartOfInterval(Timestamp, INTERVAL %s) AS bucket,
			count() AS total
		FROM telemetry.otel_logs
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'%s
		GROUP BY bucket
		ORDER BY bucket
		%s
	`, interval, formatTime(from), formatTime(to), serviceFilter(serviceName), querySettings)
}

// BuildLogLevelsQuery builds a query for log level distribution.
func BuildLogLevelsQuery(from, to time.Time, serviceName string) string {
	return fmt.Sprintf(`
		SELECT
			upper(SeverityText) AS level,
			count() AS total
		FROM telemetry.otel_logs
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'%s
		GROUP BY level
		ORDER BY total DESC
		%s
	`, formatTime(from), formatTime(to), serviceFilter(serviceName), querySettings)
}

// GetLatencyBuckets returns the latency bucket definitions.
func GetLatencyBuckets() []struct {
	Start int
	End   int
	Label string
} {
	return latencyBuckets
}
