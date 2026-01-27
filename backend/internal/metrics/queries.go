package metrics

import (
	"fmt"
	"time"
)

const (
	// ApdexThresholdMs is the target response time for APDEX calculation (T)
	ApdexThresholdMs = 2000
	// ApdexToleratingMultiplier is the multiplier for tolerating threshold (4T)
	ApdexToleratingMultiplier = 4
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
	return fmt.Sprintf(" AND ServiceName = '%s'", serviceName)
}

func httpServerSpanFilter() string {
	return `
			AND toString(SpanKind) IN ('SERVER', 'SPAN_KIND_SERVER', '2')
			AND (
				startsWith(SpanName, 'GET ') OR
				startsWith(SpanName, 'POST ') OR
				startsWith(SpanName, 'PUT ') OR
				startsWith(SpanName, 'DELETE ') OR
				startsWith(SpanName, 'PATCH ') OR
				startsWith(SpanName, 'OPTIONS ') OR
				startsWith(SpanName, 'HEAD ')
			)`
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
	`, formatTime(from), formatTime(to), httpServerSpanFilter(), serviceFilter(serviceName))
}

// BuildSlowestEndpointsQuery builds a query to get the slowest endpoints by P95.
func BuildSlowestEndpointsQuery(from, to time.Time, serviceName string, limit int) string {
	return fmt.Sprintf(`
		SELECT
			SpanName AS endpoint,
			ServiceName AS service,
			avg(Duration/1000000) AS avg_ms,
			quantile(0.50)(Duration/1000000) AS p50,
			quantile(0.95)(Duration/1000000) AS p95,
			quantile(0.99)(Duration/1000000) AS p99,
			count() AS cnt
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
			%s
			%s
		GROUP BY endpoint, service
		HAVING cnt >= 5
		ORDER BY p95 DESC
		LIMIT %d
	`, formatTime(from), formatTime(to), httpServerSpanFilter(), serviceFilter(serviceName), limit)
}

// BuildErrorHotspotsQuery builds a query to get endpoints with highest error rates.
// StatusCode can be stored as String ('Error') or as number, so we check both
func BuildErrorHotspotsQuery(from, to time.Time, serviceName string, limit int) string {
	return fmt.Sprintf(`
		SELECT
			SpanName AS endpoint,
			ServiceName AS service,
			countIf(toString(StatusCode) = 'Error' OR toString(StatusCode) = '2') AS error_count,
			count() AS total_count,
			if(count() > 0, (countIf(toString(StatusCode) = 'Error' OR toString(StatusCode) = '2') / count()) * 100, 0) AS error_rate
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
			%s
			%s
		GROUP BY endpoint, service
		HAVING total_count >= 5 AND error_count > 0
		ORDER BY error_rate DESC, error_count DESC
		LIMIT %d
	`, formatTime(from), formatTime(to), httpServerSpanFilter(), serviceFilter(serviceName), limit)
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
	`, ApdexThresholdMs, ApdexThresholdMs, toleratingThreshold, toleratingThreshold,
		formatTime(from), formatTime(to), httpServerSpanFilter(), serviceFilter(serviceName))
}

// BuildThroughputQuery builds a query to get throughput time series.
func BuildThroughputQuery(from, to time.Time, serviceName string) string {
	// Calculate interval based on time range
	duration := to.Sub(from)
	interval := "1 MINUTE"
	if duration > 24*time.Hour {
		interval = "1 HOUR"
	} else if duration > 6*time.Hour {
		interval = "15 MINUTE"
	} else if duration > 1*time.Hour {
		interval = "5 MINUTE"
	}

	return fmt.Sprintf(`
		SELECT
			toStartOfInterval(Timestamp, INTERVAL %s) AS bucket,
			count() AS request_count,
			countIf(toString(StatusCode) = 'Error' OR toString(StatusCode) = '2') AS error_count
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
			%s
			%s
		GROUP BY bucket
		ORDER BY bucket
	`, interval, formatTime(from), formatTime(to), httpServerSpanFilter(), serviceFilter(serviceName))
}

// BuildErrorRateQuery builds a query to get overall error rate.
func BuildErrorRateQuery(from, to time.Time, serviceName string) string {
	return fmt.Sprintf(`
		SELECT
			countIf(toString(StatusCode) = 'Error' OR toString(StatusCode) = '2') AS error_count,
			count() AS total_count,
			if(count() > 0, (countIf(toString(StatusCode) = 'Error' OR toString(StatusCode) = '2') / count()) * 100, 0) AS error_rate
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp <= '%s'
			%s
			%s
	`, formatTime(from), formatTime(to), httpServerSpanFilter(), serviceFilter(serviceName))
}

// GetLatencyBuckets returns the latency bucket definitions.
func GetLatencyBuckets() []struct {
	Start int
	End   int
	Label string
} {
	return latencyBuckets
}
