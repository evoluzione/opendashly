package query

import "time"

type LogEntry struct {
	Timestamp          time.Time         `json:"timestamp"`
	Severity           string            `json:"severity"`
	Body               string            `json:"body"`
	TraceID            string            `json:"traceId,omitempty"`
	SpanID             string            `json:"spanId,omitempty"`
	ResourceAttributes map[string]string `json:"resourceAttributes,omitempty"`
	LogAttributes      map[string]string `json:"logAttributes,omitempty"`
}

type TraceEntry struct {
	TraceID    string    `json:"traceId"`
	Name       string    `json:"name"`
	Service    string    `json:"service,omitempty"`
	SpanCount  uint64    `json:"spanCount,omitempty"`
	ErrorCount uint64    `json:"errorCount"`
	LastSeen   time.Time `json:"lastSeen,omitempty"`
	DurationMs float64   `json:"durationMs,omitempty"`
}

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type MetricSeries struct {
	Name   string        `json:"name"`
	Unit   string        `json:"unit,omitempty"`
	Points []MetricPoint `json:"points"`
}
