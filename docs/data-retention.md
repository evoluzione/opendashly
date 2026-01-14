# Data Retention

## Retention Policy

- Retention is configured at the storage layer (ClickHouse).
- Default retention is 30 days for logs, metrics, and traces unless overridden by operators.

## Deletion

- Data is removed according to retention policies; manual deletion is not exposed in the UI.
