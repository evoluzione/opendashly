-- Dashboard settings layout support
ALTER TABLE telemetry.dashboard_settings
ADD COLUMN IF NOT EXISTS x Int32 DEFAULT 0;

ALTER TABLE telemetry.dashboard_settings
ADD COLUMN IF NOT EXISTS y Int32 DEFAULT 0;

ALTER TABLE telemetry.dashboard_settings
ADD COLUMN IF NOT EXISTS w Int32 DEFAULT 0;

ALTER TABLE telemetry.dashboard_settings
ADD COLUMN IF NOT EXISTS h Int32 DEFAULT 0;
