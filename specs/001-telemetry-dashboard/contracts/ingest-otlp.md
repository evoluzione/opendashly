# OTLP Ingestion Contract

## Scope
This system ingests telemetry using the OpenTelemetry Collector with OTLP over gRPC and HTTP.

## Endpoints
- **OTLP gRPC**: `:4317`
- **OTLP HTTP**: `:4318`

## Supported Signals
- Logs (resource logs, log records)
- Traces (resource spans, spans)
- Metrics (resource metrics, metric data points)

## Required Attributes
- Resource attributes: `service.name`, `deployment.environment`
- Span attributes: `trace_id`, `span_id`, `span.name`, `span.kind`
- Log attributes: `severity_text`, `body`
- Metric attributes: `metric.name`, `metric.type`

## Fidelity Requirements
- No attribute normalization that drops keys or values.
- Any sampling or aggregation must be stored with lineage metadata.

## Error Handling
- Invalid payloads return OTLP standard error responses.
- Collector export failures are retried with backoff and logged.
