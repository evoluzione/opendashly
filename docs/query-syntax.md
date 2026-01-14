# Query Syntax

## Core Fields

- `signals`: logs, traces, metrics
- `timeRange`: ISO 8601 timestamps for `from` and `to`
- `filters`: key/value pairs for attributes such as `service.name`
- `limit`: optional max number of rows
- `orderBy`: `timestamp_desc` or `timestamp_asc`

## Example

```json
{
  "signals": ["logs", "traces", "metrics"],
  "timeRange": {
    "from": "2026-01-01T00:00:00Z",
    "to": "2026-01-02T00:00:00Z"
  },
  "filters": {
    "service.name": "checkout"
  },
  "limit": 500,
  "orderBy": "timestamp_desc"
}
```

## Query Visibility Rules

- All filters applied to the dashboard are user-visible and editable.
- Service selection is represented as an explicit `service.name` filter (or no filter when set to "Tutti").
- Saved queries must persist the full filter set without hidden defaults.
