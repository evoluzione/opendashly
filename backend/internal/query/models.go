package query

import "time"

// QueryRequest captures an ad-hoc query.
type QueryRequest struct {
	Signals   []string          `json:"signals"`
	TimeRange TimeRange         `json:"timeRange"`
	Filters   map[string]string `json:"filters"`
	Limit     int               `json:"limit"`
	OrderBy   string            `json:"orderBy"`
}

// TimeRange defines a query window.
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// SavedQuery represents a stored query definition.
type SavedQuery struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Request     QueryRequest `json:"request"`
	CreatedAt   time.Time    `json:"createdAt"`
}

// QueryRunSummary summarizes result counts.
type QueryRunSummary struct {
	LogCount    int `json:"logCount"`
	TraceCount  int `json:"traceCount"`
	MetricCount int `json:"metricCount"`
}

// QueryRunResult holds query results.
type QueryRunResult struct {
	RunID   string       `json:"runId"`
	Status  string       `json:"status"`
	Summary QueryRunSummary `json:"summary"`
	Results Results      `json:"results"`
}

// Results aggregates signal results.
type Results struct {
	Logs    []any `json:"logs"`
	Traces  []any `json:"traces"`
	Metrics []any `json:"metrics"`
}
