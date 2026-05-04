package query

// LogsSchemaDescription defines the structure of the logs table for AI context.
const LogsSchemaDescription = `
Table: telemetry.otel_logs
Description: Contains application logs with severity, body, and attributes.
Columns:
- Timestamp (DateTime): Time of the log entry.
- SeverityText (String): Log level (INFO, WARN, ERROR, DEBUG, etc.).
- Body (String): The main log message.
- TraceId (String): Associated trace ID.
- SpanId (String): Associated span ID.
- ServiceName (String): Name of the service that generated the log.
- ResourceAttributes (Map(String, String)): Key-value pairs describing resource (e.g., host.name, k8s.pod.name).
- LogAttributes (Map(String, String)): Key-value pairs describing structured log fields.
`

// TracesSchemaDescription defines the structure of the traces table for AI context.
const TracesSchemaDescription = `
Table: telemetry.otel_traces
Description: Contains distributed tracing spans.
Columns:
- Timestamp (DateTime): Time of the span.
- TraceId (String): Unique identifier of the trace.
- SpanId (String): Unique identifier of the span.
- ParentSpanId (String): ID of the parent span (empty if root).
- TraceState (String): W3C trace state.
- SpanName (String): Name of the operation.
- SpanKind (String): Kind of span (CLIENT, SERVER, INTERNAL, etc.).
- ServiceName (String): Name of the service.
- ResourceAttributes (Map(String, String)): Key-value pairs describing resource.
- SpanAttributes (Map(String, String)): Key-value pairs describing span attributes.
- Duration (UInt64): Duration of the span in nanoseconds.
- StatusCode (String): Status of the span (Unset, Ok, Error).
- StatusMessage (String): error description if applicable.
`
