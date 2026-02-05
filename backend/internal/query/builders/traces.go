package builders

import (
	"strconv"
	"strings"
	"time"
)

// BuildTracesQuery creates a ClickHouse SQL statement for traces.
func BuildTracesQuery(filters map[string]string, filterList []FilterItem, from, to time.Time, limit, offset int) string {
	base := "SELECT TraceId AS traceId, argMin(SpanName, Timestamp) AS name, argMin(ServiceName, Timestamp) AS service, count() AS spanCount, countIf(StatusCode = 'STATUS_CODE_ERROR') AS errorCount, max(Timestamp) AS lastSeen, max(Duration) / 1000000 AS durationMs FROM telemetry.otel_traces"
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

// BuildTracesCountQuery creates a count query for traces (counting unique TraceIds).
func BuildTracesCountQuery(filters map[string]string, filterList []FilterItem, from, to time.Time) string {
	// We need to count unique TraceIds that satisfy the conditions.
	// Since traces are group by TraceId, we can count the number of groups.
	// Accurate count: SELECT count(DISTINCT TraceId) FROM ...
	base := "SELECT uniqExact(TraceId) FROM telemetry.otel_traces"
	clauses := buildOtelClauses("Timestamp", filters, filterList, from, to, "ServiceName", "TraceId", "", []string{"ResourceAttributes", "SpanAttributes"})
	query := base
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	// No group by needed for uniqExact on the whole set, unless we had having clauses which we don't for now.
	return query
}
