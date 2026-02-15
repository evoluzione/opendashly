package builders

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type LogsPageCursor struct {
	Timestamp time.Time
	TraceID   string
	SpanID    string
}

// BuildLogsQuery creates a ClickHouse SQL statement for logs.
func BuildLogsQuery(filters map[string]string, filterList []FilterItem, from, to time.Time, limit, offset int, cursor *LogsPageCursor) string {
	base := "SELECT Timestamp AS timestamp, SeverityText AS severity, Body AS body, TraceId AS traceId, SpanId AS spanId, ResourceAttributes AS resourceAttributes, LogAttributes AS logAttributes FROM telemetry.otel_logs"
	clauses := buildOtelClauses("Timestamp", filters, filterList, from, to, "ServiceName", "TraceId", "SeverityText", []string{"ResourceAttributes", "LogAttributes"})
	if cursor != nil && !cursor.Timestamp.IsZero() {
		ts := formatDateTime64(cursor.Timestamp)
		traceID := EscapeLiteral(cursor.TraceID)
		spanID := EscapeLiteral(cursor.SpanID)
		clauses = append(clauses, fmt.Sprintf("(Timestamp < %s OR (Timestamp = %s AND TraceId < '%s') OR (Timestamp = %s AND TraceId = '%s' AND SpanId < '%s'))", ts, ts, traceID, ts, traceID, spanID))
	}
	query := base
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY Timestamp DESC, TraceId DESC, SpanId DESC"
	if limit > 0 {
		query += " LIMIT " + strconv.Itoa(limit)
		if offset > 0 && cursor == nil {
			query += " OFFSET " + strconv.Itoa(offset)
		}
	}
	return query
}

// BuildLogsCountQuery creates a count query for logs.
func BuildLogsCountQuery(filters map[string]string, filterList []FilterItem, from, to time.Time) string {
	base := "SELECT count(*) FROM telemetry.otel_logs"
	clauses := buildOtelClauses("Timestamp", filters, filterList, from, to, "ServiceName", "TraceId", "SeverityText", []string{"ResourceAttributes", "LogAttributes"})
	query := base
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	return query
}
