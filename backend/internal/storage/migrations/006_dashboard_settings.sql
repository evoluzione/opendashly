-- Dashboard settings table
CREATE TABLE IF NOT EXISTS telemetry.dashboard_settings (
  tenant_id String,
  chart_key String,
  enabled UInt8,
  updated_at DateTime,
  updated_by String
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (tenant_id, chart_key);
