CREATE DATABASE IF NOT EXISTS telemetry;

CREATE TABLE IF NOT EXISTS telemetry.logs (
  tenant_id String,
  service_name String,
  timestamp DateTime64(3),
  severity String,
  body String,
  attributes Map(String, String),
  trace_id String,
  span_id String
) ENGINE = MergeTree()
ORDER BY (tenant_id, service_name, timestamp);

CREATE TABLE IF NOT EXISTS telemetry.traces (
  tenant_id String,
  service_name String,
  trace_id String,
  span_id String,
  parent_span_id String,
  name String,
  start_time DateTime64(3),
  end_time DateTime64(3),
  status String,
  attributes Map(String, String)
) ENGINE = MergeTree()
ORDER BY (tenant_id, trace_id, start_time);

CREATE TABLE IF NOT EXISTS telemetry.metrics (
  tenant_id String,
  service_name String,
  name String,
  type String,
  unit String,
  timestamp DateTime64(3),
  value Float64,
  attributes Map(String, String)
) ENGINE = MergeTree()
ORDER BY (tenant_id, name, timestamp);

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
