package query

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"opendashly/backend/internal/query/builders"
	"opendashly/backend/internal/storage"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// TraceSpansService fetches spans for a trace.
type TraceSpansService struct {
	Storage *storage.Client
}

// TraceSpanEntry represents a span row for the timeline.
type TraceSpanEntry struct {
	TraceID      string            `json:"traceId"`
	SpanID       string            `json:"spanId"`
	ParentSpanID string            `json:"parentSpanId,omitempty"`
	Name         string            `json:"name"`
	Service      string            `json:"service,omitempty"`
	Source       string            `json:"source,omitempty"`
	StartTime    time.Time         `json:"startTime"`
	EndTime      time.Time         `json:"endTime"`
	Duration     int64             `json:"duration"`
	Status       string            `json:"status,omitempty"`
	SpanKind     *int              `json:"spanKind,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
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
	return "SELECT TraceId AS traceId, SpanId AS spanId, ParentSpanId AS parentSpanId, SpanName AS name, ServiceName AS serviceName, ServiceName AS source, Timestamp AS startTime, Duration AS duration, toString(ifNull(StatusCode, 0)) AS status, toInt32OrNull(SpanKind) AS spanKind, CAST(SpanAttributes, 'Map(String, String)') AS attributes FROM telemetry.otel_traces WHERE TraceId = '" + escapedTraceID + "' ORDER BY Timestamp ASC"
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
		var statusValue string
		var spanKindValue sql.NullInt32
		if err := rows.Scan(&row.TraceID, &row.SpanID, &row.ParentSpanID, &row.Name, &row.Service, &row.Source, &row.StartTime, &duration, &statusValue, &spanKindValue, &row.Attributes); err != nil {
			return nil, fmt.Errorf("scan trace spans: %w", err)
		}
		if duration < 0 {
			duration = 0
		}
		row.Status = normalizeStatus(statusValue)
		if spanKindValue.Valid {
			value := int(spanKindValue.Int32)
			row.SpanKind = &value
		} else {
			row.SpanKind = nil
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

func normalizeStatus(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case uint8:
		return statusLabel(int64(v))
	case uint16:
		return statusLabel(int64(v))
	case uint32:
		return statusLabel(int64(v))
	case uint64:
		return statusLabel(int64(v))
	case int8:
		return statusLabel(int64(v))
	case int16:
		return statusLabel(int64(v))
	case int32:
		return statusLabel(int64(v))
	case int64:
		return statusLabel(v)
	case int:
		return statusLabel(int64(v))
	case float32:
		return statusLabel(int64(v))
	case float64:
		return statusLabel(int64(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

func statusLabel(code int64) string {
	switch code {
	case 0:
		return "UNSET"
	case 1:
		return "OK"
	case 2:
		return "ERROR"
	default:
		return fmt.Sprintf("%d", code)
	}
}
