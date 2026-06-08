CREATE TABLE IF NOT EXISTS telemetry.status_logs_1m (
  time_bucket DateTime,
  total SimpleAggregateFunction(sum, UInt64)
) ENGINE = AggregatingMergeTree()
PARTITION BY toDate(time_bucket)
ORDER BY time_bucket
TTL time_bucket + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW IF NOT EXISTS telemetry.status_logs_1m_mv
TO telemetry.status_logs_1m
AS
SELECT
  toStartOfMinute(toDateTime(Timestamp)) AS time_bucket,
  count() AS total
FROM telemetry.otel_logs
GROUP BY time_bucket;

CREATE TABLE IF NOT EXISTS telemetry.status_traces_1m (
  time_bucket DateTime,
  total SimpleAggregateFunction(sum, UInt64)
) ENGINE = AggregatingMergeTree()
PARTITION BY toDate(time_bucket)
ORDER BY time_bucket
TTL time_bucket + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW IF NOT EXISTS telemetry.status_traces_1m_mv
TO telemetry.status_traces_1m
AS
SELECT
  toStartOfMinute(toDateTime(Timestamp)) AS time_bucket,
  count() AS total
FROM telemetry.otel_traces
GROUP BY time_bucket;

CREATE TABLE IF NOT EXISTS telemetry.status_rollup_backfill_chunks (
  rollup String,
  chunk_start DateTime,
  chunk_end DateTime,
  completed_at DateTime
) ENGINE = ReplacingMergeTree(completed_at)
ORDER BY (rollup, chunk_start, chunk_end);
