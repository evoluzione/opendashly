CREATE TABLE IF NOT EXISTS telemetry.otel_logs (
  Timestamp DateTime64(9) CODEC(Delta(8), ZSTD(1)),
  TimestampTime DateTime DEFAULT toDateTime(Timestamp),
  TraceId String CODEC(ZSTD(1)),
  SpanId String CODEC(ZSTD(1)),
  TraceFlags UInt8,
  SeverityText LowCardinality(String) CODEC(ZSTD(1)),
  SeverityNumber UInt8,
  ServiceName LowCardinality(String) CODEC(ZSTD(1)),
  Body String CODEC(ZSTD(1)),
  ResourceSchemaUrl LowCardinality(String) CODEC(ZSTD(1)),
  ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
  ScopeSchemaUrl LowCardinality(String) CODEC(ZSTD(1)),
  ScopeName String CODEC(ZSTD(1)),
  ScopeVersion LowCardinality(String) CODEC(ZSTD(1)),
  ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
  LogAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
  EventName String CODEC(ZSTD(1)),
  INDEX idx_trace_id TraceId TYPE bloom_filter(0.001) GRANULARITY 1,
  INDEX idx_logs_service ServiceName TYPE bloom_filter(0.01) GRANULARITY 64,
  INDEX idx_logs_trace TraceId TYPE bloom_filter(0.01) GRANULARITY 64,
  INDEX idx_logs_severity SeverityText TYPE set(128) GRANULARITY 64,
  INDEX idx_res_attr_key mapKeys(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
  INDEX idx_res_attr_value mapValues(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
  INDEX idx_scope_attr_key mapKeys(ScopeAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
  INDEX idx_scope_attr_value mapValues(ScopeAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
  INDEX idx_log_attr_key mapKeys(LogAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
  INDEX idx_log_attr_value mapValues(LogAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
  INDEX idx_body Body TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 8
) ENGINE = MergeTree()
PARTITION BY toDate(TimestampTime)
ORDER BY (ServiceName, TimestampTime, Timestamp)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;

CREATE TABLE IF NOT EXISTS telemetry.otel_traces (
  Timestamp DateTime64(9) CODEC(Delta(8), ZSTD(1)),
  TraceId String CODEC(ZSTD(1)),
  SpanId String CODEC(ZSTD(1)),
  ParentSpanId String CODEC(ZSTD(1)),
  TraceState String CODEC(ZSTD(1)),
  SpanName LowCardinality(String) CODEC(ZSTD(1)),
  SpanKind LowCardinality(String) CODEC(ZSTD(1)),
  ServiceName LowCardinality(String) CODEC(ZSTD(1)),
  ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
  ScopeName String CODEC(ZSTD(1)),
  ScopeVersion String CODEC(ZSTD(1)),
  SpanAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
  Duration UInt64 CODEC(ZSTD(1)),
  StatusCode LowCardinality(String) CODEC(ZSTD(1)),
  StatusMessage String CODEC(ZSTD(1)),
  Events Nested (
    Timestamp DateTime64(9),
    Name LowCardinality(String),
    Attributes Map(LowCardinality(String), String)
  ) CODEC(ZSTD(1)),
  Links Nested (
    TraceId String,
    SpanId String,
    TraceState String,
    Attributes Map(LowCardinality(String), String)
  ) CODEC(ZSTD(1)),
  INDEX idx_trace_id TraceId TYPE bloom_filter(0.001) GRANULARITY 1,
  INDEX idx_traces_service ServiceName TYPE bloom_filter(0.01) GRANULARITY 64,
  INDEX idx_traces_trace TraceId TYPE bloom_filter(0.01) GRANULARITY 64,
  INDEX idx_traces_status StatusCode TYPE set(16) GRANULARITY 64,
  INDEX idx_res_attr_key mapKeys(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
  INDEX idx_res_attr_value mapValues(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
  INDEX idx_span_attr_key mapKeys(SpanAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
  INDEX idx_span_attr_value mapValues(SpanAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
  INDEX idx_duration Duration TYPE minmax GRANULARITY 1
) ENGINE = MergeTree()
PARTITION BY toDate(Timestamp)
ORDER BY (ServiceName, SpanName, toDateTime(Timestamp))
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;

CREATE TABLE IF NOT EXISTS telemetry.otel_traces_trace_id_ts (
  TraceId String CODEC(ZSTD(1)),
  Start DateTime CODEC(Delta(8), ZSTD(1)),
  End DateTime CODEC(Delta(8), ZSTD(1)),
  INDEX idx_trace_id TraceId TYPE bloom_filter(0.01) GRANULARITY 1
) ENGINE = MergeTree()
PARTITION BY toDate(Start)
ORDER BY (TraceId, Start)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS telemetry.otel_traces_trace_id_ts_mv
TO telemetry.otel_traces_trace_id_ts
AS SELECT
  TraceId,
  min(Timestamp) AS Start,
  max(Timestamp) AS End
FROM telemetry.otel_traces
WHERE TraceId != ''
GROUP BY TraceId;
