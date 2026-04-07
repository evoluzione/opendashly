ALTER TABLE telemetry.otel_metrics_sum MODIFY TTL toDateTime(TimeUnix) + INTERVAL 30 DAY;

ALTER TABLE telemetry.otel_metrics_gauge MODIFY TTL toDateTime(TimeUnix) + INTERVAL 30 DAY;

ALTER TABLE telemetry.otel_metrics_exponential_histogram MODIFY TTL toDateTime(TimeUnix) + INTERVAL 30 DAY;

ALTER TABLE telemetry.otel_metrics_summary MODIFY TTL toDateTime(TimeUnix) + INTERVAL 30 DAY;
