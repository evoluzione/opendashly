CREATE DATABASE IF NOT EXISTS telemetry;

CREATE TABLE IF NOT EXISTS telemetry.query_runs (
  run_id String,
  tenant_id String,
  started_at DateTime64(3),
  completed_at DateTime64(3),
  status String,
  log_count UInt64,
  trace_count UInt64,
  metric_count UInt64
) ENGINE = MergeTree()
ORDER BY (tenant_id, started_at);
