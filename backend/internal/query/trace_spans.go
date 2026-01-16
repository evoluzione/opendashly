package query

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"opentelemetry-dashboard/backend/internal/query/builders"
	"opentelemetry-dashboard/backend/internal/storage"
)

// TraceSpansService fetches spans for a trace.
type TraceSpansService struct {
	Storage *storage.Client
}

// TraceSpanEntry represents a span row for the timeline.
type TraceSpanEntry struct {
	TraceID      string    `json:"traceId"`
	SpanID       string    `json:"spanId"`
	ParentSpanID string    `json:"parentSpanId,omitempty"`
	Name         string    `json:"name"`
	Service      string    `json:"service,omitempty"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	Duration     int64     `json:"duration"`
	Status       string    `json:"status,omitempty"`
}

// Spans returns spans for a trace.
func (s *TraceSpansService) Spans(ctx context.Context, traceID string) ([]TraceSpanEntry, error) {
	if s.Storage == nil {
		return []TraceSpanEntry{}, nil
	}
	query := buildTraceSpansQuery(traceID)
	return fetchTraceSpans(ctx, s.Storage.Conn, query)
}

func buildTraceSpansQuery(traceID string) string {
	escapedTraceID := builders.EscapeTraceID(traceID)
	return "SELECT TraceId AS traceId, SpanId AS spanId, ParentSpanId AS parentSpanId, SpanName AS name, ServiceName AS serviceName, Timestamp AS startTime, Duration AS duration, StatusCode AS status FROM telemetry.otel_traces WHERE TraceId = '" + escapedTraceID + "' ORDER BY Timestamp ASC"
}

func fetchTraceSpans(ctx context.Context, conn driver.Conn, query string) ([]TraceSpanEntry, error) {
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query trace spans: %w", err)
	}
	defer rows.Close()
	results := []TraceSpanEntry{}
	for rows.Next() {
		var row TraceSpanEntry
		var duration int64
		if err := rows.Scan(&row.TraceID, &row.SpanID, &row.ParentSpanID, &row.Name, &row.Service, &row.StartTime, &duration, &row.Status); err != nil {
			return nil, fmt.Errorf("scan trace spans: %w", err)
		}
		if duration < 0 {
			duration = 0
		}
		row.Duration = duration
		row.EndTime = row.StartTime.Add(time.Duration(duration))
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trace spans: %w", err)
	}
	return results, nil
}
