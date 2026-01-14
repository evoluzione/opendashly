# Data Fidelity and Auditability

## Immutability

- Ingested telemetry is treated as immutable.
- Dashboards render from stored raw data or transparent derivations.

## Sampling and Rollups

- Any sampling, aggregation, or rollup must be explicitly documented per dataset.
- Rollups must include lineage back to the raw OTLP payloads.

## Auditability

- Query results must be reproducible using visible filters and query parameters.
- Hidden filters are not permitted.
