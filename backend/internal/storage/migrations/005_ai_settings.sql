-- AI settings table
CREATE TABLE IF NOT EXISTS telemetry.ai_settings (
  tenant_id String,
  enabled UInt8, -- 0 or 1
  provider String, -- 'openai', etc.
  model String, -- 'gpt-3.5-turbo', 'gpt-4', etc.
  api_key String, -- stored as plain text for now
  updated_at DateTime
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (tenant_id);
