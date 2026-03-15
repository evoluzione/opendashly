package telemetry

import "time"

// LogEntry represents a log record.
type LogEntry struct {
	Timestamp  time.Time         `json:"timestamp"`
	Severity   string            `json:"severity"`
	Body       string            `json:"body"`
	Attributes map[string]string `json:"attributes"`
	TraceID    string            `json:"traceId"`
	SpanID     string            `json:"spanId"`
}

// TraceSpan represents a trace span.
type TraceSpan struct {
	TraceID      string            `json:"traceId"`
	SpanID       string            `json:"spanId"`
	ParentSpanID string            `json:"parentSpanId"`
	Name         string            `json:"name"`
	StartTime    time.Time         `json:"startTime"`
	EndTime      time.Time         `json:"endTime"`
	Status       string            `json:"status"`
	Attributes   map[string]string `json:"attributes"`
}

// MetricPoint represents a metric point.
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// MetricSeries represents a metric time series.
type MetricSeries struct {
	Name       string            `json:"name"`
	Unit       string            `json:"unit"`
	Points     []MetricPoint     `json:"points"`
	Attributes map[string]string `json:"attributes"`
}
