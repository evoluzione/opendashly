# Performance Budgets

## Ingest and Query Latency

- Ingest latency budget: 2 seconds end-to-end from OTLP ingest to queryable storage at typical load.
- Query latency budget: 2 seconds p95 for interactive dashboard queries at typical load.

## Interactive UI

- Tab switch + filter updates should render within 2 seconds for typical queries.
- Auth endpoints should complete within 300ms p95.

## Scope

These budgets apply to single-tenant deployments with tens of concurrent users.
