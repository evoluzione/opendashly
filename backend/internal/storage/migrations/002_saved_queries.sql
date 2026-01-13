CREATE TABLE IF NOT EXISTS telemetry.saved_queries (
  id String,
  tenant_id String,
  name String,
  description String,
  request String,
  created_at DateTime64(3)
) ENGINE = MergeTree()
ORDER BY (tenant_id, created_at);
