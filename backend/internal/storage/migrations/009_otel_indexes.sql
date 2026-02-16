-- Data-skipping indexes for OpenTelemetry tables created by the collector.
-- These improve selective filters on large datasets (millions of rows).

ALTER TABLE telemetry.otel_logs
ADD INDEX IF NOT EXISTS idx_logs_service ServiceName TYPE bloom_filter(0.01) GRANULARITY 64;

ALTER TABLE telemetry.otel_logs
ADD INDEX IF NOT EXISTS idx_logs_trace TraceId TYPE bloom_filter(0.01) GRANULARITY 64;

ALTER TABLE telemetry.otel_logs
ADD INDEX IF NOT EXISTS idx_logs_severity SeverityText TYPE set(128) GRANULARITY 64;

ALTER TABLE telemetry.otel_traces
ADD INDEX IF NOT EXISTS idx_traces_service ServiceName TYPE bloom_filter(0.01) GRANULARITY 64;

ALTER TABLE telemetry.otel_traces
ADD INDEX IF NOT EXISTS idx_traces_trace TraceId TYPE bloom_filter(0.01) GRANULARITY 64;

ALTER TABLE telemetry.otel_traces
ADD INDEX IF NOT EXISTS idx_traces_status StatusCode TYPE set(16) GRANULARITY 64;

ALTER TABLE telemetry.otel_metrics_sum
ADD INDEX IF NOT EXISTS idx_metrics_sum_service ServiceName TYPE bloom_filter(0.01) GRANULARITY 64;

ALTER TABLE telemetry.otel_metrics_gauge
ADD INDEX IF NOT EXISTS idx_metrics_gauge_service ServiceName TYPE bloom_filter(0.01) GRANULARITY 64;
