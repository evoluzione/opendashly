-- Dashboard settings deterministic versioning for argMax reads
ALTER TABLE telemetry.dashboard_settings
ADD COLUMN IF NOT EXISTS version UInt64 DEFAULT 0;
