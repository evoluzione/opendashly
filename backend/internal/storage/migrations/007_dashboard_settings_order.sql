-- Dashboard settings order support
ALTER TABLE telemetry.dashboard_settings
ADD COLUMN IF NOT EXISTS order_index Int32 DEFAULT 0;
