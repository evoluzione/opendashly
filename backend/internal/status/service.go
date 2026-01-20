package status

import (
	"context"
	"fmt"
	"time"

	"opendashly/backend/internal/storage"
)

type TelemetryCounts struct {
	Total  uint64 `json:"total"`
	Last5m uint64 `json:"last5m"`
	Last10m uint64 `json:"last10m"`
	Last60m uint64 `json:"last60m"`
}

type Checks struct {
	Database bool `json:"database"`
}

type Summary struct {
	Ok          bool            `json:"ok"`
	GeneratedAt time.Time       `json:"generatedAt"`
	Error       string          `json:"error,omitempty"`
	Checks      Checks          `json:"checks"`
	Counts      SummaryCounts   `json:"counts"`
}

type SummaryCounts struct {
	Logs    TelemetryCounts `json:"logs"`
	Traces  TelemetryCounts `json:"traces"`
	Metrics TelemetryCounts `json:"metrics"`
}

type Service struct {
	Storage *storage.Client
}

func (s *Service) Summary(ctx context.Context) Summary {
	summary := Summary{
		Ok:          false,
		GeneratedAt: time.Now().UTC(),
		Checks:      Checks{Database: false},
	}
	if s.Storage == nil {
		summary.Error = "storage not configured"
		return summary
	}
	if err := s.Storage.Conn.Ping(ctx); err != nil {
		summary.Error = fmt.Sprintf("database unavailable: %v", err)
		return summary
	}
	summary.Checks.Database = true

	logs, err := fetchCounts(ctx, s.Storage, logsCountsQuery)
	if err != nil {
		summary.Error = fmt.Sprintf("logs counts failed: %v", err)
		return summary
	}
	traces, err := fetchCounts(ctx, s.Storage, tracesCountsQuery)
	if err != nil {
		summary.Error = fmt.Sprintf("traces counts failed: %v", err)
		return summary
	}
	metrics, err := fetchCounts(ctx, s.Storage, metricsCountsQuery)
	if err != nil {
		summary.Error = fmt.Sprintf("metrics counts failed: %v", err)
		return summary
	}

	summary.Counts = SummaryCounts{
		Logs:    logs,
		Traces:  traces,
		Metrics: metrics,
	}
	summary.Ok = summary.Checks.Database
	return summary
}

func fetchCounts(ctx context.Context, store *storage.Client, query string) (TelemetryCounts, error) {
	var counts TelemetryCounts
	row := store.Conn.QueryRow(ctx, query)
	if err := row.Scan(&counts.Total, &counts.Last5m, &counts.Last10m, &counts.Last60m); err != nil {
		return TelemetryCounts{}, err
	}
	return counts, nil
}

const logsCountsQuery = `
	SELECT
		count() AS total,
		countIf(Timestamp >= now() - INTERVAL 5 MINUTE) AS last5m,
		countIf(Timestamp >= now() - INTERVAL 10 MINUTE) AS last10m,
		countIf(Timestamp >= now() - INTERVAL 60 MINUTE) AS last60m
	FROM telemetry.otel_logs
`

const tracesCountsQuery = `
	SELECT
		uniqExact(TraceId) AS total,
		uniqExactIf(TraceId, Timestamp >= now() - INTERVAL 5 MINUTE) AS last5m,
		uniqExactIf(TraceId, Timestamp >= now() - INTERVAL 10 MINUTE) AS last10m,
		uniqExactIf(TraceId, Timestamp >= now() - INTERVAL 60 MINUTE) AS last60m
	FROM telemetry.otel_traces
`

const metricsCountsQuery = `
	SELECT
		sum(total) AS total,
		sum(last5m) AS last5m,
		sum(last10m) AS last10m,
		sum(last60m) AS last60m
	FROM (
		SELECT
			count() AS total,
			countIf(TimeUnix >= now() - INTERVAL 5 MINUTE) AS last5m,
			countIf(TimeUnix >= now() - INTERVAL 10 MINUTE) AS last10m,
			countIf(TimeUnix >= now() - INTERVAL 60 MINUTE) AS last60m
		FROM telemetry.otel_metrics_sum
		UNION ALL
		SELECT
			count() AS total,
			countIf(TimeUnix >= now() - INTERVAL 5 MINUTE) AS last5m,
			countIf(TimeUnix >= now() - INTERVAL 10 MINUTE) AS last10m,
			countIf(TimeUnix >= now() - INTERVAL 60 MINUTE) AS last60m
		FROM telemetry.otel_metrics_gauge
	)
`
