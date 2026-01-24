package builders

import (
	"strconv"
	"strings"
	"time"
)

// BuildTracesQuery creates a ClickHouse SQL statement for traces.
func BuildTracesQuery(filters map[string]string, from, to time.Time, limit, offset int) string {
	base := "SELECT TraceId AS traceId, any(SpanName) AS name, any(ServiceName) AS serviceName, count() AS spanCount, max(Timestamp) AS lastSeen, dateDiff('millisecond', min(Timestamp), max(addNanoseconds(Timestamp, Duration))) AS durationMs FROM telemetry.otel_traces"
	clauses := buildOtelClauses("Timestamp", filters, from, to, "ServiceName", "TraceId", "", []string{"ResourceAttributes", "SpanAttributes"})
	query := base
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " GROUP BY TraceId ORDER BY lastSeen DESC"
	if limit > 0 {
		query += " LIMIT " + strconv.Itoa(limit)
		if offset > 0 {
			query += " OFFSET " + strconv.Itoa(offset)
		}
	}
	return query
}
