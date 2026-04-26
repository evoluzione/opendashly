CREATE TABLE IF NOT EXISTS telemetry.dashboard_trace_service_1m (
  time_bucket DateTime,
  ServiceName LowCardinality(String),
  request_count SimpleAggregateFunction(sum, UInt64),
  error_count SimpleAggregateFunction(sum, UInt64),
  satisfied_count SimpleAggregateFunction(sum, UInt64),
  tolerating_count SimpleAggregateFunction(sum, UInt64),
  frustrated_count SimpleAggregateFunction(sum, UInt64),
  latency_0_100 SimpleAggregateFunction(sum, UInt64),
  latency_100_250 SimpleAggregateFunction(sum, UInt64),
  latency_250_500 SimpleAggregateFunction(sum, UInt64),
  latency_500_1000 SimpleAggregateFunction(sum, UInt64),
  latency_1000_2000 SimpleAggregateFunction(sum, UInt64),
  latency_2000_5000 SimpleAggregateFunction(sum, UInt64),
  latency_5000_10000 SimpleAggregateFunction(sum, UInt64),
  latency_10000_inf SimpleAggregateFunction(sum, UInt64),
  status_ok SimpleAggregateFunction(sum, UInt64),
  status_error SimpleAggregateFunction(sum, UInt64),
  status_unset SimpleAggregateFunction(sum, UInt64),
  status_other SimpleAggregateFunction(sum, UInt64),
  duration_sum_ms SimpleAggregateFunction(sum, Float64),
  duration_quantiles_state AggregateFunction(quantilesTDigest(0.5, 0.95, 0.99), Float64)
) ENGINE = AggregatingMergeTree()
PARTITION BY toDate(time_bucket)
ORDER BY (ServiceName, time_bucket)
TTL time_bucket + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW IF NOT EXISTS telemetry.dashboard_trace_service_1m_mv
TO telemetry.dashboard_trace_service_1m
AS
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
  WHERE
    upper(toString(SpanKind)) IN ('SERVER', 'SPAN_KIND_SERVER')
    OR toString(SpanKind) = '2'
)
GROUP BY time_bucket, ServiceName;

CREATE TABLE IF NOT EXISTS telemetry.dashboard_trace_endpoint_1m (
  time_bucket DateTime,
  ServiceName LowCardinality(String),
  Endpoint LowCardinality(String),
  request_count SimpleAggregateFunction(sum, UInt64),
  error_count SimpleAggregateFunction(sum, UInt64),
  duration_sum_ms SimpleAggregateFunction(sum, Float64),
  duration_quantiles_state AggregateFunction(quantilesTDigest(0.5, 0.95, 0.99), Float64)
) ENGINE = AggregatingMergeTree()
PARTITION BY toDate(time_bucket)
ORDER BY (ServiceName, Endpoint, time_bucket)
TTL time_bucket + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW IF NOT EXISTS telemetry.dashboard_trace_endpoint_1m_mv
TO telemetry.dashboard_trace_endpoint_1m
AS
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
  WHERE
    (
      upper(toString(SpanKind)) IN ('SERVER', 'SPAN_KIND_SERVER')
      OR toString(SpanKind) = '2'
    )
    AND (
      startsWith(SpanName, 'GET ')
      OR startsWith(SpanName, 'POST ')
      OR startsWith(SpanName, 'PUT ')
      OR startsWith(SpanName, 'PATCH ')
      OR startsWith(SpanName, 'DELETE ')
      OR startsWith(SpanName, 'OPTIONS ')
      OR startsWith(SpanName, 'HEAD ')
    )
)
GROUP BY time_bucket, ServiceName, Endpoint;

CREATE TABLE IF NOT EXISTS telemetry.dashboard_log_level_1m (
  time_bucket DateTime,
  ServiceName LowCardinality(String),
  level LowCardinality(String),
  total SimpleAggregateFunction(sum, UInt64)
) ENGINE = AggregatingMergeTree()
PARTITION BY toDate(time_bucket)
ORDER BY (ServiceName, level, time_bucket)
TTL time_bucket + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW IF NOT EXISTS telemetry.dashboard_log_level_1m_mv
TO telemetry.dashboard_log_level_1m
AS
SELECT
  toStartOfMinute(toDateTime(Timestamp)) AS time_bucket,
  ServiceName,
  upper(SeverityText) AS level,
  count() AS total
FROM telemetry.otel_logs
GROUP BY time_bucket, ServiceName, level;

CREATE TABLE IF NOT EXISTS telemetry.dashboard_rollup_backfill_chunks (
  rollup String,
  chunk_start DateTime,
  chunk_end DateTime,
  completed_at DateTime
) ENGINE = ReplacingMergeTree(completed_at)
ORDER BY (rollup, chunk_start, chunk_end);
