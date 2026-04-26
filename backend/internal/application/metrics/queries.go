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

// querySettings keeps every dashboard query inside a tight memory envelope.
// Dashboard widgets read minute rollups, so these settings should be a guard
// rail instead of the primary defense against raw-table aggregations.
const querySettings = ` SETTINGS
	max_memory_usage = 67108864,
	max_bytes_before_external_group_by = 16777216,
	max_bytes_before_external_sort = 16777216,
	group_by_two_level_threshold = 10000,
	group_by_two_level_threshold_bytes = 16777216,
	max_threads = 1,
	distributed_aggregation_memory_efficient = 1,
	optimize_aggregation_in_order = 1`

const (
	traceServiceRollupTable  = "telemetry.dashboard_trace_service_1m"
	traceEndpointRollupTable = "telemetry.dashboard_trace_endpoint_1m"
	logLevelRollupTable      = "telemetry.dashboard_log_level_1m"
)

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
			bucket_start,
			bucket_end,
			cnt
		FROM (
			SELECT
				[toInt32(0), toInt32(100), toInt32(250), toInt32(500), toInt32(1000), toInt32(2000), toInt32(5000), toInt32(10000)] AS bucket_starts,
				[toInt32(100), toInt32(250), toInt32(500), toInt32(1000), toInt32(2000), toInt32(5000), toInt32(10000), toInt32(-1)] AS bucket_ends,
				[
					toUInt64(sum(latency_0_100)),
					toUInt64(sum(latency_100_250)),
					toUInt64(sum(latency_250_500)),
					toUInt64(sum(latency_500_1000)),
					toUInt64(sum(latency_1000_2000)),
					toUInt64(sum(latency_2000_5000)),
					toUInt64(sum(latency_5000_10000)),
					toUInt64(sum(latency_10000_inf))
				] AS counts
			FROM %s
			WHERE time_bucket >= '%s' AND time_bucket <= '%s'
				%s
		)
		ARRAY JOIN bucket_starts AS bucket_start, bucket_ends AS bucket_end, counts AS cnt
		ORDER BY bucket_start
		%s
		`, traceServiceRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), querySettings)
}

// BuildSlowestEndpointsQuery builds a query to get the slowest endpoints by P95.
// quantileTDigest is memory-bounded (O(1) state per group) so it survives high
// SpanName cardinality without exploding the aggregation hash table.
func BuildSlowestEndpointsQuery(from, to time.Time, serviceName string, limit int) string {
	return fmt.Sprintf(`
		SELECT
			endpoint,
			service,
			if(cnt > 0, duration_sum_ms / cnt, 0) AS avg_ms,
			toFloat64(qs[1]) AS p50,
			toFloat64(qs[2]) AS p95,
			toFloat64(qs[3]) AS p99,
			cnt
		FROM (
			SELECT
				Endpoint AS endpoint,
				ServiceName AS service,
				sum(duration_sum_ms) AS duration_sum_ms,
				toUInt64(sum(request_count)) AS cnt,
				quantilesTDigestMerge(0.5, 0.95, 0.99)(duration_quantiles_state) AS qs
			FROM %s
			WHERE time_bucket >= '%s' AND time_bucket <= '%s'
				%s
			GROUP BY endpoint, service
		)
		WHERE cnt >= 5
		ORDER BY p95 DESC
		LIMIT %d
		%s
		`, traceEndpointRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), limit, querySettings)
}

// BuildErrorHotspotsQuery builds a query to get endpoints with highest error rates.
// StatusCode can be stored as String ('Error') or as number, so we check both
func BuildErrorHotspotsQuery(from, to time.Time, serviceName string, limit int) string {
	return fmt.Sprintf(`
		SELECT
			Endpoint AS endpoint,
			ServiceName AS service,
			toUInt64(sum(error_count)) AS error_count,
			toUInt64(sum(request_count)) AS total_count,
			if(sum(request_count) > 0, (sum(error_count) / sum(request_count)) * 100, 0) AS error_rate
		FROM %s
		WHERE time_bucket >= '%s' AND time_bucket <= '%s'
				%s
		GROUP BY endpoint, service
		HAVING total_count >= 5 AND error_count > 0
		ORDER BY error_rate DESC, error_count DESC
		LIMIT %d
		%s
		`, traceEndpointRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), limit, querySettings)
}

// BuildApdexQuery builds a query to calculate APDEX score.
func BuildApdexQuery(from, to time.Time, serviceName string) string {
	return fmt.Sprintf(`
		SELECT
			toUInt64(sum(satisfied_count)) AS satisfied,
			toUInt64(sum(tolerating_count)) AS tolerating,
			toUInt64(sum(frustrated_count)) AS frustrated,
			toUInt64(sum(request_count)) AS total
		FROM %s
		WHERE time_bucket >= '%s' AND time_bucket <= '%s'
				%s
		%s
		`, traceServiceRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), querySettings)
}

// BuildThroughputQuery builds a query to get throughput time series.
func BuildThroughputQuery(from, to time.Time, serviceName string) string {
	interval := timeBucketInterval(from, to)

	return fmt.Sprintf(`
		SELECT
			toStartOfInterval(time_bucket, INTERVAL %s) AS bucket,
			toUInt64(sum(request_count)) AS request_count,
			toUInt64(sum(error_count)) AS error_count
		FROM %s
		WHERE time_bucket >= '%s' AND time_bucket <= '%s'
				%s
		GROUP BY bucket
		ORDER BY bucket
		%s
		`, interval, traceServiceRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), querySettings)
}

// BuildErrorRateQuery builds a query to get overall error rate.
func BuildErrorRateQuery(from, to time.Time, serviceName string) string {
	return fmt.Sprintf(`
		SELECT
			toUInt64(sum(error_count)) AS error_count,
			toUInt64(sum(request_count)) AS total_count,
			if(sum(request_count) > 0, (sum(error_count) / sum(request_count)) * 100, 0) AS error_rate
		FROM %s
		WHERE time_bucket >= '%s' AND time_bucket <= '%s'
				%s
		%s
		`, traceServiceRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), querySettings)
}

// BuildLatencyPercentilesQuery builds a query for latency percentiles over time.
func BuildLatencyPercentilesQuery(from, to time.Time, serviceName string) string {
	interval := timeBucketInterval(from, to)
	return fmt.Sprintf(`
		SELECT
			bucket,
			toFloat64(qs[1]) AS p50,
			toFloat64(qs[2]) AS p95,
			toFloat64(qs[3]) AS p99
		FROM (
			SELECT
				toStartOfInterval(time_bucket, INTERVAL %s) AS bucket,
				quantilesTDigestMerge(0.5, 0.95, 0.99)(duration_quantiles_state) AS qs
			FROM %s
			WHERE time_bucket >= '%s' AND time_bucket <= '%s'
			%s
			GROUP BY bucket
		)
		ORDER BY bucket
		%s
	`, interval, traceServiceRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), querySettings)
}

// BuildErrorRateTimeSeriesQuery builds a query for error rate over time.
func BuildErrorRateTimeSeriesQuery(from, to time.Time, serviceName string) string {
	interval := timeBucketInterval(from, to)
	return fmt.Sprintf(`
		SELECT
			toStartOfInterval(time_bucket, INTERVAL %s) AS bucket,
			toUInt64(sum(error_count)) AS error_count,
			toUInt64(sum(request_count)) AS total_count,
			if(sum(request_count) > 0, (sum(error_count) / sum(request_count)) * 100, 0) AS error_rate
		FROM %s
		WHERE time_bucket >= '%s' AND time_bucket <= '%s'
			%s
		GROUP BY bucket
		ORDER BY bucket
		%s
	`, interval, traceServiceRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), querySettings)
}

// BuildStatusCodeBreakdownQuery builds a query for status code distribution.
func BuildStatusCodeBreakdownQuery(from, to time.Time, serviceName string) string {
	return fmt.Sprintf(`
		SELECT
			code,
			total
		FROM (
			SELECT
				['ok', 'error', 'unset', 'other'] AS codes,
				[
					toUInt64(sum(status_ok)),
					toUInt64(sum(status_error)),
					toUInt64(sum(status_unset)),
					toUInt64(sum(status_other))
				] AS totals
			FROM %s
			WHERE time_bucket >= '%s' AND time_bucket <= '%s'
			%s
		)
		ARRAY JOIN codes AS code, totals AS total
		ORDER BY total DESC
		%s
	`, traceServiceRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), querySettings)
}

// BuildTopEndpointsThroughputQuery builds a query for top endpoints by throughput.
func BuildTopEndpointsThroughputQuery(from, to time.Time, serviceName string, limit int) string {
	return fmt.Sprintf(`
		SELECT
			Endpoint AS endpoint,
			ServiceName AS service,
			toUInt64(sum(request_count)) AS request_count,
			toUInt64(sum(error_count)) AS error_count,
			if(sum(request_count) > 0, (sum(error_count) / sum(request_count)) * 100, 0) AS error_rate
		FROM %s
		WHERE time_bucket >= '%s' AND time_bucket <= '%s'
			%s
		GROUP BY endpoint, service
		ORDER BY request_count DESC
		LIMIT %d
		%s
	`, traceEndpointRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), limit, querySettings)
}

// BuildLogVolumeQuery builds a query for log volume over time.
func BuildLogVolumeQuery(from, to time.Time, serviceName string) string {
	interval := timeBucketInterval(from, to)
	return fmt.Sprintf(`
		SELECT
			toStartOfInterval(time_bucket, INTERVAL %s) AS bucket,
			toUInt64(sum(total)) AS total
		FROM %s
		WHERE time_bucket >= '%s' AND time_bucket <= '%s'%s
		GROUP BY bucket
		ORDER BY bucket
		%s
	`, interval, logLevelRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), querySettings)
}

// BuildLogLevelsQuery builds a query for log level distribution.
func BuildLogLevelsQuery(from, to time.Time, serviceName string) string {
	return fmt.Sprintf(`
		SELECT
			level,
			toUInt64(sum(total)) AS total
		FROM %s
		WHERE time_bucket >= '%s' AND time_bucket <= '%s'%s
		GROUP BY level
		ORDER BY total DESC
		%s
	`, logLevelRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), querySettings)
}

// GetLatencyBuckets returns the latency bucket definitions.
func GetLatencyBuckets() []struct {
	Start int
	End   int
	Label string
} {
	return latencyBuckets
}

func buildTraceServiceBackfillQuery(from, to time.Time) string {
	return fmt.Sprintf(`
		INSERT INTO %s
		SELECT
			time_bucket,
			ServiceName,
			count() AS request_count,
			countIf(status_code IN ('Error', '2', 'STATUS_CODE_ERROR')) AS error_count,
			countIf(duration_ms <= 2000) AS satisfied_count,
			countIf(duration_ms > 2000 AND duration_ms <= 8000) AS tolerating_count,
			countIf(duration_ms > 8000) AS frustrated_count,
			countIf(duration_ms <= 100) AS latency_0_100,
			countIf(duration_ms > 100 AND duration_ms <= 250) AS latency_100_250,
			countIf(duration_ms > 250 AND duration_ms <= 500) AS latency_250_500,
			countIf(duration_ms > 500 AND duration_ms <= 1000) AS latency_500_1000,
			countIf(duration_ms > 1000 AND duration_ms <= 2000) AS latency_1000_2000,
			countIf(duration_ms > 2000 AND duration_ms <= 5000) AS latency_2000_5000,
			countIf(duration_ms > 5000 AND duration_ms <= 10000) AS latency_5000_10000,
			countIf(duration_ms > 10000) AS latency_10000_inf,
			countIf(status_code IN ('Ok', 'STATUS_CODE_OK', '1')) AS status_ok,
			countIf(status_code IN ('Error', 'STATUS_CODE_ERROR', '2')) AS status_error,
			countIf(status_code IN ('Unset', 'STATUS_CODE_UNSET', '0')) AS status_unset,
			countIf(status_code NOT IN ('Ok', 'STATUS_CODE_OK', '1', 'Error', 'STATUS_CODE_ERROR', '2', 'Unset', 'STATUS_CODE_UNSET', '0')) AS status_other,
			sum(duration_ms) AS duration_sum_ms,
			quantilesTDigestState(0.5, 0.95, 0.99)(duration_ms) AS duration_quantiles_state
		FROM (
			SELECT
				toStartOfMinute(toDateTime(Timestamp)) AS time_bucket,
				ServiceName,
				toString(StatusCode) AS status_code,
				toFloat64(Duration) / 1000000.0 AS duration_ms
			FROM telemetry.otel_traces
			WHERE Timestamp >= '%s' AND Timestamp < '%s'
				%s
		)
		GROUP BY time_bucket, ServiceName
		%s
	`, traceServiceRollupTable, formatTime(from), formatTime(to), serverSpanFilter(), querySettings)
}

func buildTraceEndpointBackfillQuery(from, to time.Time) string {
	return fmt.Sprintf(`
		INSERT INTO %s
		SELECT
			time_bucket,
			ServiceName,
			Endpoint,
			count() AS request_count,
			countIf(status_code IN ('Error', '2', 'STATUS_CODE_ERROR')) AS error_count,
			sum(duration_ms) AS duration_sum_ms,
			quantilesTDigestState(0.5, 0.95, 0.99)(duration_ms) AS duration_quantiles_state
		FROM (
			SELECT
				toStartOfMinute(toDateTime(Timestamp)) AS time_bucket,
				ServiceName,
				SpanName AS Endpoint,
				toString(StatusCode) AS status_code,
				toFloat64(Duration) / 1000000.0 AS duration_ms
			FROM telemetry.otel_traces
			WHERE Timestamp >= '%s' AND Timestamp < '%s'
				%s
				AND %s
		)
		GROUP BY time_bucket, ServiceName, Endpoint
		%s
	`, traceEndpointRollupTable, formatTime(from), formatTime(to), serverSpanFilter(), endpointSpanFilter(), querySettings)
}

func buildLogLevelBackfillQuery(from, to time.Time) string {
	return fmt.Sprintf(`
		INSERT INTO %s
		SELECT
			toStartOfMinute(toDateTime(Timestamp)) AS time_bucket,
			ServiceName,
			upper(SeverityText) AS level,
			count() AS total
		FROM telemetry.otel_logs
		WHERE Timestamp >= '%s' AND Timestamp < '%s'
		GROUP BY time_bucket, ServiceName, level
		%s
	`, logLevelRollupTable, formatTime(from), formatTime(to), querySettings)
}
