-- Retention settings table
CREATE TABLE IF NOT EXISTS telemetry.retention_settings (
  id String,
  signal_type String,
  retention_days UInt32,
  updated_at DateTime,
  updated_by String
) ENGINE = MergeTree()
ORDER BY (signal_type);

-- Default 7-day retention for supported signals
INSERT INTO telemetry.retention_settings (id, signal_type, retention_days, updated_at, updated_by)
VALUES
  (generateUUIDv4(), 'logs', 7, now(), 'system'),
  (generateUUIDv4(), 'traces', 7, now(), 'system');

-- Cleanup jobs audit table
CREATE TABLE IF NOT EXISTS telemetry.cleanup_jobs (
  job_id String,
  job_type String,
  signal_type String,
  service_name String,
  started_at DateTime,
  completed_at Nullable(DateTime),
  status String,
  records_deleted UInt64,
  error_message String
) ENGINE = MergeTree()
ORDER BY (started_at);
