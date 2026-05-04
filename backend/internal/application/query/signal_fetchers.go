package query

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func fetchLogs(ctx context.Context, conn driver.Conn, query string) ([]LogEntry, error) {
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()
	results := []LogEntry{}
	for rows.Next() {
		var row LogEntry
		if err := rows.Scan(&row.Timestamp, &row.Severity, &row.Body, &row.TraceID, &row.SpanID, &row.ResourceAttributes, &row.LogAttributes); err != nil {
			return nil, fmt.Errorf("scan logs: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate logs: %w", err)
	}
	return results, nil
}

func fetchTraces(ctx context.Context, conn driver.Conn, query string) ([]TraceEntry, error) {
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query traces: %w", err)
	}
	defer rows.Close()
	results := []TraceEntry{}
	for rows.Next() {
		var row TraceEntry
		if err := rows.Scan(&row.TraceID, &row.Name, &row.Service, &row.SpanCount, &row.ErrorCount, &row.LastSeen, &row.DurationMs); err != nil {
			return nil, fmt.Errorf("scan traces: %w", err)
		}
		if row.DurationMs < 0 {
			row.DurationMs = 0
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate traces: %w", err)
	}
	return results, nil
}

func trimToPage[T any](items []T, limit int) ([]T, bool) {
	if limit <= 0 {
		return items, false
	}
	if len(items) > limit {
		return items[:limit], true
	}
	return items, false
}
