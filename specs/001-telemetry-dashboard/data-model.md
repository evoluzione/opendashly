# Data Model: OpenTelemetry Telemetry Dashboard

## Entities

### Tenant
- **Fields**: id, name, created_at
- **Relationships**: has many Users, Resources, Queries, Dashboards
- **Validation**: name required, unique within system

### User
- **Fields**: id, tenant_id, name, email, role, created_at, last_login_at
- **Relationships**: belongs to Tenant; owns many Queries, Dashboards
- **Validation**: email unique within tenant; role in [viewer, editor, admin]

### Resource
- **Fields**: id, tenant_id, service_name, environment, attributes (map), first_seen_at
- **Relationships**: belongs to Tenant; referenced by Telemetry Signals
- **Validation**: service_name required

### Telemetry Signal (abstract)
- **Fields**: id, tenant_id, resource_id, timestamp, attributes (map)
- **Relationships**: belongs to Tenant and Resource
- **Notes**: Specialized as LogEntry, TraceSpan, MetricSeries

### LogEntry
- **Fields**: id, tenant_id, resource_id, timestamp, severity, body, attributes (map)
- **Relationships**: may reference TraceSpan via trace_id/span_id
- **Validation**: severity in known levels

### TraceSpan
- **Fields**: id, tenant_id, resource_id, trace_id, span_id, parent_span_id, name, start_time, end_time, status, attributes (map)
- **Relationships**: has many LogEntries; links to MetricSeries by resource and time
- **Validation**: trace_id and span_id required

### MetricSeries
- **Fields**: id, tenant_id, resource_id, name, type, unit, points (time,value), attributes (map)
- **Relationships**: belongs to Resource
- **Validation**: name required; type in [gauge, sum, histogram]

### Query
- **Fields**: id, tenant_id, name, description, created_by, created_at, last_run_at
- **Relationships**: has many QueryFilters; has many QueryRuns
- **Validation**: name required for saved queries

### QueryFilter
- **Fields**: id, query_id, signal_type, time_range, filters (map), limit, order_by
- **Relationships**: belongs to Query
- **Validation**: signal_type in [logs, traces, metrics]

### QueryRun
- **Fields**: id, query_id, tenant_id, status, started_at, completed_at, result_counts (map)
- **Relationships**: belongs to Query; references result sets
- **Validation**: status in [running, complete, failed]

### DashboardView
- **Fields**: id, tenant_id, name, created_by, layout, created_at
- **Relationships**: has many Visualizations
- **Validation**: name required

### Visualization
- **Fields**: id, dashboard_id, title, signal_type, query_ref, chart_type, config
- **Relationships**: belongs to DashboardView; references Query or QueryRun
- **Validation**: chart_type in [timeseries, table, histogram]

## Relationships Summary

- Tenant -> Users, Resources, Queries, Dashboards
- Resource -> LogEntries, TraceSpans, MetricSeries
- Query -> QueryFilters, QueryRuns
- DashboardView -> Visualizations
- TraceSpan -> LogEntries (via trace_id/span_id)
