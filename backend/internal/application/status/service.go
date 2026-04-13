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
	Checks      Checks        `json:"checks"`
	Counts      SummaryCounts `json:"counts"`
}

type SummaryCounts struct {
	Logs    TelemetryCounts `json:"logs"`
	Traces  TelemetryCounts `json:"traces"`
	Metrics TelemetryCounts `json:"metrics"`
}

type Service struct {
	Storage            *storage.Client
	CollectorHealthURL string
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

	anchor := summary.GeneratedAt.UTC().Truncate(time.Duration(seriesBucketMinutes) * time.Minute)
	logs.Series, err = fetchSeries(ctx, s.Storage, logsSeriesQuery, anchor, seriesBucketMinutes, seriesPointCount)
	if err != nil {
		summary.Error = fmt.Sprintf("logs series failed: %v", err)
		return summary
	}
	traces.Series, err = fetchSeries(ctx, s.Storage, tracesSeriesQuery, anchor, seriesBucketMinutes, seriesPointCount)
	if err != nil {
		summary.Error = fmt.Sprintf("traces series failed: %v", err)
		return summary
	}
	metrics.Series, err = fetchSeries(ctx, s.Storage, metricsSeriesQuery, anchor, seriesBucketMinutes, seriesPointCount)
	if err != nil {
		summary.Error = fmt.Sprintf("metrics series failed: %v", err)
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
		countIf(Timestamp >= now() - INTERVAL 5 MINUTE) AS last5m,
		countIf(Timestamp >= now() - INTERVAL 10 MINUTE) AS last10m,
		count() AS last60m
	FROM telemetry.otel_logs
	WHERE Timestamp >= now() - INTERVAL 60 MINUTE
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

const logsSeriesQuery = `
	SELECT
		toStartOfInterval(Timestamp, INTERVAL 5 MINUTE) AS bucket,
		count() AS count
	FROM telemetry.otel_logs
	WHERE Timestamp >= now() - INTERVAL 60 MINUTE
	GROUP BY bucket
	ORDER BY bucket
`

const tracesSeriesQuery = `
	SELECT
		toStartOfInterval(Timestamp, INTERVAL 5 MINUTE) AS bucket,
		uniqExact(TraceId) AS count
	FROM telemetry.otel_traces
	WHERE Timestamp >= now() - INTERVAL 60 MINUTE
	GROUP BY bucket
	ORDER BY bucket
`

const metricsSeriesQuery = `
	SELECT
		bucket,
		sum(count) AS count
	FROM (
		SELECT
			toStartOfInterval(TimeUnix, INTERVAL 5 MINUTE) AS bucket,
			count() AS count
		FROM telemetry.otel_metrics_sum
		WHERE TimeUnix >= now() - INTERVAL 60 MINUTE
		GROUP BY bucket
		UNION ALL
		SELECT
			toStartOfInterval(TimeUnix, INTERVAL 5 MINUTE) AS bucket,
			count() AS count
		FROM telemetry.otel_metrics_gauge
		WHERE TimeUnix >= now() - INTERVAL 60 MINUTE
		GROUP BY bucket
	)
	GROUP BY bucket
	ORDER BY bucket
`
