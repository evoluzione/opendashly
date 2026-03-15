ALTER TABLE telemetry.otel_logs MODIFY TTL toDateTime(Timestamp) + INTERVAL 7 DAY;

ALTER TABLE telemetry.otel_traces MODIFY TTL toDateTime(Timestamp) + INTERVAL 15 DAY;

ALTER TABLE telemetry.otel_metrics_histogram MODIFY TTL toDateTime(TimeUnix) + INTERVAL 30 DAY;
