CREATE TABLE IF NOT EXISTS telemetry.ai_assistant_sessions (
  tenant_id String,
  user_id String,
  messages_json String,
  updated_at DateTime
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (tenant_id, user_id);
