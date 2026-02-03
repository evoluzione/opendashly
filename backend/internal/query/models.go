package query

import (
	"time"

	"opendashly/backend/internal/query/builders"
)

// FilterItem represents a single filter condition.
// Moved to builders package, removed from here.

// QueryRequest captures an ad-hoc query.
type QueryRequest struct {
	Signals    []string              `json:"signals"`
	TimeRange  TimeRange             `json:"timeRange"`
	Filters    map[string]string     `json:"filters"` // Deprecated: use FilterList
	FilterList []builders.FilterItem `json:"filterList"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	OrderBy    string                `json:"orderBy"`
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
	RunID      string          `json:"runId"`
	Status     string          `json:"status"`
	Summary    QueryRunSummary `json:"summary"`
	Pagination PaginationSet   `json:"pagination"`
	Results    Results         `json:"results"`
}

// Results aggregates signal results.
type Results struct {
	Logs    []any `json:"logs"`
	Traces  []any `json:"traces"`
	Metrics []any `json:"metrics"`
}

// Pagination captures paging metadata for a signal.
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// PaginationSet groups paging metadata per signal.
type PaginationSet struct {
	Logs    Pagination `json:"logs"`
	Traces  Pagination `json:"traces"`
	Metrics Pagination `json:"metrics"`
}
