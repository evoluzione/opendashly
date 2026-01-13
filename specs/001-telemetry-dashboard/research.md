# Phase 0 Research: OpenTelemetry Telemetry Dashboard

## Decision: OpenTelemetry Collector as ingress
**Rationale**: Collector provides standard OTLP ingest, reliable batching, and vendor-neutral pipelines. It preserves resource and signal attributes and supports lossless export to ClickHouse.
**Alternatives considered**: Direct OTLP ingestion inside the Go API; Fluent Bit pipelines. Rejected to keep ingestion decoupled and standards-based.

## Decision: ClickHouse as telemetry storage
**Rationale**: Columnar storage fits high-volume logs and metrics, supports fast aggregations, and scales horizontally for long retention.
**Alternatives considered**: Elasticsearch, PostgreSQL with time-series extensions. Rejected due to cost/performance tradeoffs for mixed signals at scale.

## Decision: Go for query and correlation APIs
**Rationale**: Go is well suited for high-throughput APIs, integrates cleanly with ClickHouse drivers, and offers straightforward concurrency for query fan-out.
**Alternatives considered**: Node.js, Java. Rejected due to throughput and operational complexity at target scale.

## Decision: SvelteKit + uPlot for UI
**Rationale**: SvelteKit supports fast, reactive UIs and uPlot provides performant time-series rendering for dense telemetry.
**Alternatives considered**: React + charting libraries, Grafana embedding. Rejected due to heavier runtime and limited customization for query workflows.

## Decision: Query model with explicit filters and saved queries
**Rationale**: Explicit filter objects ensure queries are visible, reproducible, and auditable. Saved queries are first-class entities aligned with the constitution.
**Alternatives considered**: Implicit UI filters and ad-hoc query strings. Rejected due to hidden state and poor auditability.

## Decision: Correlation via shared resource and trace identifiers
**Rationale**: Using resource attributes and trace/span IDs enables deterministic pivoting across signals without heuristics.
**Alternatives considered**: Heuristic correlation by time windows only. Rejected due to higher false positives.

## Decision: Data fidelity and immutability
**Rationale**: Raw telemetry remains immutable; aggregations and sampling are stored with lineage metadata to maintain auditability.
**Alternatives considered**: Overwriting or compacting raw data. Rejected because it obscures root-cause analysis.

## Decision: Performance budgets
**Rationale**: Targets align with success criteria and provide measurable ingest/query expectations for scaling and caching strategies.
**Alternatives considered**: No explicit budgets. Rejected because they are required by the constitution and needed for UX targets.
