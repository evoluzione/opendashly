ALTER TABLE telemetry.dashboard_trace_endpoint_1m
  ADD COLUMN IF NOT EXISTS latency_0_100 SimpleAggregateFunction(sum, UInt64) AFTER error_count;

ALTER TABLE telemetry.dashboard_trace_endpoint_1m
  ADD COLUMN IF NOT EXISTS latency_100_250 SimpleAggregateFunction(sum, UInt64) AFTER latency_0_100;

ALTER TABLE telemetry.dashboard_trace_endpoint_1m
  ADD COLUMN IF NOT EXISTS latency_250_500 SimpleAggregateFunction(sum, UInt64) AFTER latency_100_250;

ALTER TABLE telemetry.dashboard_trace_endpoint_1m
  ADD COLUMN IF NOT EXISTS latency_500_1000 SimpleAggregateFunction(sum, UInt64) AFTER latency_250_500;

ALTER TABLE telemetry.dashboard_trace_endpoint_1m
  ADD COLUMN IF NOT EXISTS latency_1000_2000 SimpleAggregateFunction(sum, UInt64) AFTER latency_500_1000;

ALTER TABLE telemetry.dashboard_trace_endpoint_1m
  ADD COLUMN IF NOT EXISTS latency_2000_5000 SimpleAggregateFunction(sum, UInt64) AFTER latency_1000_2000;

ALTER TABLE telemetry.dashboard_trace_endpoint_1m
  ADD COLUMN IF NOT EXISTS latency_5000_10000 SimpleAggregateFunction(sum, UInt64) AFTER latency_2000_5000;

ALTER TABLE telemetry.dashboard_trace_endpoint_1m
  ADD COLUMN IF NOT EXISTS latency_10000_inf SimpleAggregateFunction(sum, UInt64) AFTER latency_5000_10000;

DROP TABLE IF EXISTS telemetry.dashboard_trace_endpoint_1m_mv;

CREATE MATERIALIZED VIEW IF NOT EXISTS telemetry.dashboard_trace_endpoint_1m_mv
TO telemetry.dashboard_trace_endpoint_1m
AS
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

ALTER TABLE telemetry.dashboard_rollup_backfill_chunks
  DELETE WHERE rollup = 'dashboard_v1'
  SETTINGS mutations_sync = 1;
