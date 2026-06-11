package status

import (
	"context"
	"fmt"
	"time"

	"opendashly/backend/internal/infrastructure/storage"
)

type TelemetryCounts struct {
	Total   uint64           `json:"total"`
	Last5m  uint64           `json:"last5m"`
	Last10m uint64           `json:"last10m"`
	Last60m uint64           `json:"last60m"`
	Series  []TelemetryPoint `json:"series,omitempty"`
}

type TelemetryPoint struct {
	Ts    time.Time `json:"ts"`
	Count uint64    `json:"count"`
}

type Checks struct {
	Database bool `json:"database"`
}

type Summary struct {
	Ok          bool          `json:"ok"`
	GeneratedAt time.Time     `json:"generatedAt"`
	Error       string        `json:"error,omitempty"`
	Warnings    []string      `json:"warnings,omitempty"`
	Checks      Checks        `json:"checks"`
	Counts      SummaryCounts `json:"counts"`
}

type SummaryCounts struct {
	Logs   TelemetryCounts `json:"logs"`
	Traces TelemetryCounts `json:"traces"`
}

type Service struct {
	Storage            *storage.Client
	MaintenanceStorage *storage.Client
	CollectorHealthURL string

	pingDatabase  func(context.Context) error
	countsFetcher func(context.Context, *storage.Client, string) (TelemetryCounts, error)
	seriesFetcher func(context.Context, *storage.Client, string, time.Time, int, int) ([]TelemetryPoint, error)
}

const (
	seriesBucketMinutes = 5
	seriesPointCount    = 12
)

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
	if err := s.getDatabasePinger()(ctx); err != nil {
		summary.Error = fmt.Sprintf("database unavailable: %v", err)
		return summary
	}
	summary.Checks.Database = true

	warnings := make([]string, 0, 4)
	logs, err := s.getCountsFetcher()(ctx, s.Storage, logsCountsQuery)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("logs counts failed: %v", err))
	}
	traces, err := s.getCountsFetcher()(ctx, s.Storage, tracesCountsQuery)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("traces counts failed: %v", err))
	}
	anchor := summary.GeneratedAt.UTC().Truncate(time.Duration(seriesBucketMinutes) * time.Minute)
	logs.Series, err = s.getSeriesFetcher()(ctx, s.Storage, logsSeriesQuery, anchor, seriesBucketMinutes, seriesPointCount)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("logs series failed: %v", err))
	}
	traces.Series, err = s.getSeriesFetcher()(ctx, s.Storage, tracesSeriesQuery, anchor, seriesBucketMinutes, seriesPointCount)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("traces series failed: %v", err))
	}
	summary.Counts = SummaryCounts{
		Logs:   logs,
		Traces: traces,
	}
	summary.Ok = summary.Checks.Database
	if len(warnings) > 0 {
		summary.Warnings = warnings
	}
	return summary
}

// databaseProbeQuery is the health probe for ClickHouse. A protocol-level
// Ping (or an untracked SELECT 1) succeeds even when the server's (total)
// memory tracker is past its cap and every real query fails with code 241,
// so the probe is a real query forced through the tracker.
const databaseProbeQuery = `
	SELECT count()
	FROM numbers(65536)
	SETTINGS max_untracked_memory = 0, max_threads = 1, max_memory_usage = 33554432, max_execution_time = 2
`

func (s *Service) getDatabasePinger() func(context.Context) error {
	if s.pingDatabase != nil {
		return s.pingDatabase
	}
	return func(ctx context.Context) error {
		if s.Storage == nil || s.Storage.Conn == nil {
			return fmt.Errorf("storage not configured")
		}
		pingCtx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
		defer cancel()
		var probed uint64
		return s.Storage.Conn.QueryRow(pingCtx, databaseProbeQuery).Scan(&probed)
	}
}

func (s *Service) getCountsFetcher() func(context.Context, *storage.Client, string) (TelemetryCounts, error) {
	if s.countsFetcher != nil {
		return s.countsFetcher
	}
	return func(ctx context.Context, store *storage.Client, query string) (TelemetryCounts, error) {
		queryCtx, cancel := context.WithTimeout(ctx, 1200*time.Millisecond)
		defer cancel()
		return fetchCounts(queryCtx, store, query)
	}
}

func (s *Service) getSeriesFetcher() func(context.Context, *storage.Client, string, time.Time, int, int) ([]TelemetryPoint, error) {
	if s.seriesFetcher != nil {
		return s.seriesFetcher
	}
	return func(ctx context.Context, store *storage.Client, query string, anchor time.Time, bucketMinutes int, pointCount int) ([]TelemetryPoint, error) {
		queryCtx, cancel := context.WithTimeout(ctx, 1200*time.Millisecond)
		defer cancel()
		return fetchSeries(queryCtx, store, query, anchor, bucketMinutes, pointCount)
	}
}

func fetchCounts(ctx context.Context, store *storage.Client, query string) (TelemetryCounts, error) {
	var counts TelemetryCounts
	row := store.Conn.QueryRow(ctx, query)
	if err := row.Scan(&counts.Total, &counts.Last5m, &counts.Last10m, &counts.Last60m); err != nil {
		return TelemetryCounts{}, err
	}
	return counts, nil
}

func fetchSeries(ctx context.Context, store *storage.Client, query string, anchor time.Time, bucketMinutes int, pointCount int) ([]TelemetryPoint, error) {
	bucketSize := time.Duration(bucketMinutes) * time.Minute
	rows, err := store.Conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	countsByBucket := make(map[int64]uint64, pointCount)
	for rows.Next() {
		var bucket time.Time
		var count uint64
		if err := rows.Scan(&bucket, &count); err != nil {
			return nil, err
		}
		bucketUnix := bucket.UTC().Truncate(bucketSize).Unix()
		countsByBucket[bucketUnix] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	series := make([]TelemetryPoint, 0, pointCount)
	start := anchor.UTC().Truncate(bucketSize).Add(time.Duration(-(pointCount - 1)) * bucketSize)
	for i := 0; i < pointCount; i++ {
		ts := start.Add(time.Duration(i) * bucketSize)
		series = append(series, TelemetryPoint{
			Ts:    ts,
			Count: countsByBucket[ts.Unix()],
		})
	}

	return series, nil
}

const logsCountsQuery = `
	WITH (
		SELECT toUInt64(ifNull(sum(rows), 0))
		FROM system.parts
		WHERE active AND database = 'telemetry' AND table = 'otel_logs'
	) AS total_rows
	SELECT
		total_rows AS total,
		toUInt64(ifNull(sumIf(total, time_bucket >= now() - INTERVAL 5 MINUTE), 0)) AS last5m,
		toUInt64(ifNull(sumIf(total, time_bucket >= now() - INTERVAL 10 MINUTE), 0)) AS last10m,
		toUInt64(ifNull(sum(total), 0)) AS last60m
	FROM telemetry.status_logs_1m
	WHERE time_bucket >= now() - INTERVAL 60 MINUTE
	SETTINGS max_execution_time = 2, max_threads = 1, max_memory_usage = 33554432
`

const tracesCountsQuery = `
	WITH (
		SELECT toUInt64(ifNull(sum(rows), 0))
		FROM system.parts
		WHERE active AND database = 'telemetry' AND table = 'otel_traces'
	) AS total_rows
	SELECT
		total_rows AS total,
		toUInt64(ifNull(sumIf(total, time_bucket >= now() - INTERVAL 5 MINUTE), 0)) AS last5m,
		toUInt64(ifNull(sumIf(total, time_bucket >= now() - INTERVAL 10 MINUTE), 0)) AS last10m,
		toUInt64(ifNull(sum(total), 0)) AS last60m
	FROM telemetry.status_traces_1m
	WHERE time_bucket >= now() - INTERVAL 60 MINUTE
	SETTINGS max_execution_time = 2, max_threads = 1, max_memory_usage = 33554432
`

const logsSeriesQuery = `
	SELECT
		toStartOfInterval(time_bucket, INTERVAL 5 MINUTE) AS bucket,
		toUInt64(sum(total)) AS count
	FROM telemetry.status_logs_1m
	WHERE time_bucket >= now() - INTERVAL 60 MINUTE
	GROUP BY bucket
	ORDER BY bucket
	SETTINGS max_execution_time = 2, max_threads = 1, max_memory_usage = 33554432
`

const tracesSeriesQuery = `
	SELECT
		toStartOfInterval(time_bucket, INTERVAL 5 MINUTE) AS bucket,
		toUInt64(sum(total)) AS count
	FROM telemetry.status_traces_1m
	WHERE time_bucket >= now() - INTERVAL 60 MINUTE
	GROUP BY bucket
	ORDER BY bucket
	SETTINGS max_execution_time = 2, max_threads = 1, max_memory_usage = 33554432
`
