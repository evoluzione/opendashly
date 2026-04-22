CREATE TABLE IF NOT EXISTS telemetry.workspace_settings (
  tenant_id String,
  setting_key String,
  setting_value String,
  updated_at DateTime,
  updated_by String
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (tenant_id, setting_key);
