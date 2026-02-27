package query

import (
	"context"
	"fmt"
	"math"
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
	StatusMessage string           `json:"statusMessage,omitempty"`
	SpanKind     any               `json:"spanKind,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Events       []TraceSpanEvent  `json:"events,omitempty"`
}

// TraceSpanEvent represents a single span event.
type TraceSpanEvent struct {
	Name       string            `json:"name"`
	Timestamp  time.Time         `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
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
	return "SELECT TraceId AS traceId, SpanId AS spanId, ParentSpanId AS parentSpanId, SpanName AS name, ServiceName AS serviceName, ServiceName AS source, Timestamp AS startTime, Duration AS duration, StatusCode AS status, StatusMessage AS statusMessage, SpanKind AS spanKind, SpanAttributes AS attributes, `Events.Timestamp` AS eventTimestamps, `Events.Name` AS eventNames, `Events.Attributes` AS eventAttributes FROM telemetry.otel_traces WHERE TraceId = '" + escapedTraceID + "' ORDER BY Timestamp ASC"
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
		var duration uint64
		var statusValue string
		var statusMessage string
		var spanKindValue string
		var eventTimestamps []time.Time
		var eventNames []string
		var eventAttributes []map[string]string
		if err := rows.Scan(&row.TraceID, &row.SpanID, &row.ParentSpanID, &row.Name, &row.Service, &row.Source, &row.StartTime, &duration, &statusValue, &statusMessage, &spanKindValue, &row.Attributes, &eventTimestamps, &eventNames, &eventAttributes); err != nil {
			return nil, fmt.Errorf("scan trace spans: %w", err)
		}
		durationForResponse := duration
		if durationForResponse > uint64(math.MaxInt64) {
			durationForResponse = uint64(math.MaxInt64)
		}
		row.Status = normalizeStatus(statusValue)
		if statusMessage != "" {
			row.StatusMessage = statusMessage
		}
		if spanKindValue != "" {
			row.SpanKind = spanKindValue
		} else {
			row.SpanKind = nil
		}
		row.Events = zipSpanEvents(eventTimestamps, eventNames, eventAttributes)
		row.Duration = int64(durationForResponse)
		row.EndTime = row.StartTime.Add(time.Duration(durationForResponse))
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trace spans: %w", err)
	}
	return results, nil
}

func zipSpanEvents(timestamps []time.Time, names []string, attributes []map[string]string) []TraceSpanEvent {
	maxLen := len(timestamps)
	if len(names) > maxLen {
		maxLen = len(names)
	}
	if len(attributes) > maxLen {
		maxLen = len(attributes)
	}
	if maxLen == 0 {
		return nil
	}

	events := make([]TraceSpanEvent, 0, maxLen)
	for i := 0; i < maxLen; i++ {
		event := TraceSpanEvent{}
		if i < len(names) {
			event.Name = names[i]
		}
		if i < len(timestamps) {
			event.Timestamp = timestamps[i]
		}
		if i < len(attributes) {
			event.Attributes = attributes[i]
		}
		events = append(events, event)
	}
	return events
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
