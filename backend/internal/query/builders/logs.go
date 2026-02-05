package builders

import (
	"strconv"
	"strings"
	"time"
)

// BuildLogsQuery creates a ClickHouse SQL statement for logs.
func BuildLogsQuery(filters map[string]string, filterList []FilterItem, from, to time.Time, limit, offset int) string {
	base := "SELECT Timestamp AS timestamp, SeverityText AS severity, Body AS body, TraceId AS traceId, SpanId AS spanId, ResourceAttributes AS resourceAttributes, LogAttributes AS logAttributes FROM telemetry.otel_logs"
	clauses := buildOtelClauses("Timestamp", filters, filterList, from, to, "ServiceName", "TraceId", "SeverityText", []string{"ResourceAttributes", "LogAttributes"})
	query := base
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY Timestamp DESC"
	if limit > 0 {
		query += " LIMIT " + strconv.Itoa(limit)
		if offset > 0 {
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
