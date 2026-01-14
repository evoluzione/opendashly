CREATE TABLE IF NOT EXISTS telemetry.users (
  id String,
  username String,
  password_hash String,
  role String,
  must_change_password UInt8,
  is_disabled UInt8,
  created_at DateTime,
  last_login_at Nullable(DateTime)
) ENGINE = MergeTree()
ORDER BY (username);
