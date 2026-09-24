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

// formatTime writes a ClickHouse time literal. Rollup buckets are UTC, so
// times in any other zone (the assistant uses the viewer's) are converted.
func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
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

func endpointLatencyBucketCountsExpression() string {
	return `[
						toUInt64(sum(latency_0_100)),
						toUInt64(sum(latency_100_250)),
						toUInt64(sum(latency_250_500)),
						toUInt64(sum(latency_500_1000)),
						toUInt64(sum(latency_1000_2000)),
						toUInt64(sum(latency_2000_5000)),
						toUInt64(sum(latency_5000_10000)),
						toUInt64(sum(latency_10000_inf))
					]`
}

func endpointLatencyBucketStartsExpression() string {
	return "[toFloat64(0), toFloat64(100), toFloat64(250), toFloat64(500), toFloat64(1000), toFloat64(2000), toFloat64(5000), toFloat64(10000)]"
}

// endpointLatencyBucketEndsExpression returns the upper edge of each bucket.
// The last bucket (>10s) is unbounded, so its end is capped at its own start:
// percentiles landing there report that edge instead of extrapolating into
// the open tail.
func endpointLatencyBucketEndsExpression() string {
	return "[toFloat64(100), toFloat64(250), toFloat64(500), toFloat64(1000), toFloat64(2000), toFloat64(5000), toFloat64(10000), toFloat64(10000)]"
}

// bucketPercentileExpression linearly interpolates the percentile value
// within the histogram bucket that contains the target rank, instead of
// snapping to a bucket edge. It intentionally avoids merging per-endpoint
// TDigest states (see BuildSlowestEndpointsQuery) because that blows the
// query memory budget when grouping by high-cardinality endpoints.
func bucketPercentileExpression(factor string) string {
	target := fmt.Sprintf("toUInt64(ceil(toFloat64(bucket_cnt) * %s))", factor)
	rawIndex := fmt.Sprintf("arrayFirstIndex(x -> x >= %s, cumulative_counts)", target)
	index := fmt.Sprintf("if(%s = 0, length(bucket_counts), %s)", rawIndex, rawIndex)
	prevCum := fmt.Sprintf("if(%s = 1, 0, cumulative_counts[%s - 1])", index, index)
	inBucket := fmt.Sprintf("bucket_counts[%s]", index)
	start := fmt.Sprintf("bucket_starts[%s]", index)
	end := fmt.Sprintf("bucket_ends[%s]", index)
	fraction := fmt.Sprintf("if(%s > 0, (toFloat64(%s) - toFloat64(%s)) / toFloat64(%s), 0)", inBucket, target, prevCum, inBucket)
	return fmt.Sprintf("if(bucket_cnt > 0, %s + %s * (%s - %s), 0)", start, fraction, end, start)
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
// It uses fixed latency buckets (instead of merging high-cardinality TDigest
// states) and linearly interpolates within the bucket that holds each
// percentile's target rank, so values aren't snapped to the bucket edges.
func BuildSlowestEndpointsQuery(from, to time.Time, serviceName string, limit int) string {
	p50Expr := bucketPercentileExpression("0.50")
	p95Expr := bucketPercentileExpression("0.95")
	p99Expr := bucketPercentileExpression("0.99")

	return fmt.Sprintf(`
			SELECT
				endpoint,
				service,
				if(cnt > 0, duration_sum_ms / cnt, 0) AS avg_ms,
				%s AS p50,
				%s AS p95,
				%s AS p99,
				cnt
			FROM (
				SELECT
					endpoint,
					service,
					duration_sum_ms,
					cnt,
					bucket_counts,
					toUInt64(arraySum(bucket_counts)) AS bucket_cnt,
					arrayCumSum(bucket_counts) AS cumulative_counts,
					%s AS bucket_starts,
					%s AS bucket_ends
				FROM (
					SELECT
						Endpoint AS endpoint,
						ServiceName AS service,
						sum(duration_sum_ms) AS duration_sum_ms,
						toUInt64(sum(request_count)) AS cnt,
						%s AS bucket_counts
					FROM %s
					WHERE time_bucket >= '%s' AND time_bucket <= '%s'
						%s
					GROUP BY endpoint, service
				)
			)
			WHERE cnt >= 5
			ORDER BY p95 DESC
			LIMIT %d
			%s
			`, p50Expr, p95Expr, p99Expr, endpointLatencyBucketStartsExpression(), endpointLatencyBucketEndsExpression(), endpointLatencyBucketCountsExpression(), traceEndpointRollupTable, formatTime(from), formatTime(to), serviceFilter(serviceName), limit, querySettings)
}

// BuildErrorHotspotsQuery builds a query to get endpoints with highest error rates.
// StatusCode can be stored as String ('Error') or as number, so we check both
func BuildErrorHotspotsQuery(from, to time.Time, serviceName string, limit int) string {
	return fmt.Sprintf(`
		SELECT
			endpoint,
			service,
			toUInt64(errors) AS error_count,
			toUInt64(total) AS total_count,
			if(total > 0, (toFloat64(errors) / toFloat64(total)) * 100, 0) AS error_rate
		FROM (
			SELECT
				Endpoint AS endpoint,
				ServiceName AS service,
				sum(error_count) AS errors,
				sum(request_count) AS total
			FROM %s
			WHERE time_bucket >= '%s' AND time_bucket <= '%s'
					%s
			GROUP BY endpoint, service
		)
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
			toUInt64(errors) AS error_count,
			toUInt64(total) AS total_count,
			if(total > 0, (toFloat64(errors) / toFloat64(total)) * 100, 0) AS error_rate
		FROM (
			SELECT
				sum(error_count) AS errors,
				sum(request_count) AS total
			FROM %s
			WHERE time_bucket >= '%s' AND time_bucket <= '%s'
					%s
		)
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
			bucket,
			toUInt64(errors) AS error_count,
			toUInt64(total) AS total_count,
			if(total > 0, (toFloat64(errors) / toFloat64(total)) * 100, 0) AS error_rate
		FROM (
			SELECT
				toStartOfInterval(time_bucket, INTERVAL %s) AS bucket,
				sum(error_count) AS errors,
				sum(request_count) AS total
			FROM %s
			WHERE time_bucket >= '%s' AND time_bucket <= '%s'
				%s
			GROUP BY bucket
		)
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
			endpoint,
			service,
			toUInt64(requests) AS request_count,
			toUInt64(errors) AS error_count,
			if(requests > 0, (toFloat64(errors) / toFloat64(requests)) * 100, 0) AS error_rate
		FROM (
			SELECT
				Endpoint AS endpoint,
				ServiceName AS service,
				sum(request_count) AS requests,
				sum(error_count) AS errors
			FROM %s
			WHERE time_bucket >= '%s' AND time_bucket <= '%s'
				%s
			GROUP BY endpoint, service
		)
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
			INSERT INTO %s (
				time_bucket,
				ServiceName,
				Endpoint,
				request_count,
				error_count,
				latency_0_100,
				latency_100_250,
				latency_250_500,
				latency_500_1000,
				latency_1000_2000,
				latency_2000_5000,
				latency_5000_10000,
				latency_10000_inf,
				duration_sum_ms,
				duration_quantiles_state
			)
			SELECT
				time_bucket,
				ServiceName,
				Endpoint,
				count() AS request_count,
				countIf(status_code IN ('Error', '2', 'STATUS_CODE_ERROR')) AS error_count,
				countIf(duration_ms <= 100) AS latency_0_100,
				countIf(duration_ms > 100 AND duration_ms <= 250) AS latency_100_250,
				countIf(duration_ms > 250 AND duration_ms <= 500) AS latency_250_500,
				countIf(duration_ms > 500 AND duration_ms <= 1000) AS latency_500_1000,
				countIf(duration_ms > 1000 AND duration_ms <= 2000) AS latency_1000_2000,
				countIf(duration_ms > 2000 AND duration_ms <= 5000) AS latency_2000_5000,
				countIf(duration_ms > 5000 AND duration_ms <= 10000) AS latency_5000_10000,
				countIf(duration_ms > 10000) AS latency_10000_inf,
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

// routeIDPattern matches path segments that carry ids ("SKU-7503", "12345")
// while keeping short version segments such as "v1".
const routeIDPattern = `/(?:[0-9][^/ ]{2,}|[^/ ][0-9][^/ ]+|[^/ ]{2,}[0-9][^/ ]*)`

// BuildRouteStatsQuery aggregates the endpoint rollup per route (ids in paths
// collapsed to "…") with optional HTTP method and path filters. With byRoute
// false it returns one row with the totals, percentiles included, computed
// from the summed latency histograms.
func BuildRouteStatsQuery(from, to time.Time, serviceName, method, path string, byRoute bool, limit int) string {
	p50Expr := bucketPercentileExpression("0.50")
	p95Expr := bucketPercentileExpression("0.95")
	p99Expr := bucketPercentileExpression("0.99")
	filters := serviceFilter(serviceName)
	if method != "" {
		filters += fmt.Sprintf(" AND match(Endpoint, '(^|\\\\s)%s(\\\\s|$)')", strings.ToUpper(strings.ReplaceAll(method, "'", "")))
	}
	if path != "" {
		filters += fmt.Sprintf(" AND positionCaseInsensitive(Endpoint, '%s') > 0", strings.ReplaceAll(strings.ReplaceAll(path, `\`, `\\`), "'", `\'`))
	}
	keys, groupBy, order := "'' AS route, '' AS service", "", ""
	if byRoute {
		keys = fmt.Sprintf("replaceRegexpAll(Endpoint, '%s', '/…') AS route, ServiceName AS service", routeIDPattern)
		groupBy = "GROUP BY route, service"
		order = fmt.Sprintf("ORDER BY cnt DESC LIMIT %d", limit)
	}
	return fmt.Sprintf(`
		SELECT
			route,
			service,
			if(cnt > 0, duration_sum_ms / cnt, 0) AS avg_ms,
			%s AS p50,
			%s AS p95,
			%s AS p99,
			cnt,
			errors
		FROM (
			SELECT
				route, service, duration_sum_ms, cnt, errors, bucket_counts,
				toUInt64(arraySum(bucket_counts)) AS bucket_cnt,
				arrayCumSum(bucket_counts) AS cumulative_counts,
				%s AS bucket_starts,
				%s AS bucket_ends
			FROM (
				SELECT
					%s,
					sum(duration_sum_ms) AS duration_sum_ms,
					toUInt64(sum(request_count)) AS cnt,
					toUInt64(sum(error_count)) AS errors,
					%s AS bucket_counts
				FROM %s
				WHERE time_bucket >= '%s' AND time_bucket <= '%s'
					%s
				%s
			)
		)
		%s
		%s
		`, p50Expr, p95Expr, p99Expr, endpointLatencyBucketStartsExpression(), endpointLatencyBucketEndsExpression(),
		keys, endpointLatencyBucketCountsExpression(), traceEndpointRollupTable, formatTime(from), formatTime(to), filters, groupBy, order, querySettings)
}
