package builders

import (
	"strconv"
	"strings"
	"time"
)

// BuildTracesQuery creates a ClickHouse SQL statement for traces.
func BuildTracesQuery(filters map[string]string, filterList []FilterItem, from, to time.Time, limit, offset int) string {
	base := "SELECT TraceId AS traceId, any(SpanName) AS name, any(ServiceName) AS service, count() AS spanCount, countIf(StatusCode = 'STATUS_CODE_ERROR') AS errorCount, max(Timestamp) AS lastSeen, max(Duration) / 1000000 AS durationMs FROM telemetry.otel_traces"
	clauses := buildOtelClauses("Timestamp", filters, filterList, from, to, "ServiceName", "TraceId", "", []string{"ResourceAttributes", "SpanAttributes"})
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
