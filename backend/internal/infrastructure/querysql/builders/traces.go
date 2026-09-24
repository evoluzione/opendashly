package builders

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type TracesPageCursor struct {
	LastSeen time.Time
	TraceID  string
}

// traceErrorStatusExpr matches an errored span. StatusCode is stored as 'Error',
// '2' or 'STATUS_CODE_ERROR' depending on the OTel ingestion path, so all three
// representations must be checked — consistent with the span detail view and the
// metrics queries. Matching only one of them silently undercounts errors.
const traceErrorStatusExpr = "toString(StatusCode) IN ('Error', '2', 'STATUS_CODE_ERROR')"

// BuildTracesQuery creates a ClickHouse SQL statement for traces.
func BuildTracesQuery(filters map[string]string, filterList []FilterItem, from, to time.Time, limit, offset int, cursor *TracesPageCursor) string {
	filteredForTraces := filterListForSignal("traces", filterList)
	traceErrorScope, effectiveFilterList := extractTraceErrorScope(filteredForTraces)
	base := "SELECT TraceId AS traceId, argMin(SpanName, Timestamp) AS name, argMin(ServiceName, Timestamp) AS service, count() AS spanCount, countIf(" + traceErrorStatusExpr + ") AS errorCount, max(Timestamp) AS lastSeen, max(Duration) / 1000000 AS durationMs FROM telemetry.otel_traces"
	clauses := buildOtelClauses("Timestamp", filters, effectiveFilterList, from, to, "ServiceName", "TraceId", "", []string{"ResourceAttributes", "SpanAttributes"})
	clauses = withErrorTracePrefilter(traceErrorScope, clauses)
	query := base
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " GROUP BY TraceId"
	switch traceErrorScope {
	case "with_errors":
		query += " HAVING errorCount > 0"
	case "without_errors":
		query += " HAVING errorCount = 0"
	}
	if cursor != nil && !cursor.LastSeen.IsZero() {
		ts := formatDateTime64(cursor.LastSeen)
		traceID := EscapeLiteral(cursor.TraceID)
		cursorClause := fmt.Sprintf("(lastSeen < %s OR (lastSeen = %s AND TraceId < '%s'))", ts, ts, traceID)
		if strings.Contains(query, " HAVING ") {
			query += " AND " + cursorClause
		} else {
			query += " HAVING " + cursorClause
		}
	}
	query += " ORDER BY lastSeen DESC, TraceId DESC"
	if limit > 0 {
		query += " LIMIT " + strconv.Itoa(limit)
		if offset > 0 && cursor == nil {
			query += " OFFSET " + strconv.Itoa(offset)
		}
	}
	query += rawTelemetryQuerySettings()
	return query
}

// BuildTracesCountQuery creates a count query for traces (counting unique TraceIds).
func BuildTracesCountQuery(filters map[string]string, filterList []FilterItem, from, to time.Time) string {
	filteredForTraces := filterListForSignal("traces", filterList)
	traceErrorScope, effectiveFilterList := extractTraceErrorScope(filteredForTraces)
	base := "SELECT TraceId, countIf(" + traceErrorStatusExpr + ") AS errorCount FROM telemetry.otel_traces"
	clauses := buildOtelClauses("Timestamp", filters, effectiveFilterList, from, to, "ServiceName", "TraceId", "", []string{"ResourceAttributes", "SpanAttributes"})
	clauses = withErrorTracePrefilter(traceErrorScope, clauses)
	query := base
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " GROUP BY TraceId"
	switch traceErrorScope {
	case "with_errors":
		query += " HAVING errorCount > 0"
	case "without_errors":
		query += " HAVING errorCount = 0"
	}
	return "SELECT count() FROM (" + query + ")" + rawTelemetryQuerySettings()
}

func extractTraceErrorScope(filterList []FilterItem) (string, []FilterItem) {
	scope := "all"
	if len(filterList) == 0 {
		return scope, filterList
	}

	filtered := make([]FilterItem, 0, len(filterList))
	for _, f := range filterList {
		if f.Key == "trace_error_scope" {
			switch strings.ToLower(strings.TrimSpace(f.Value)) {
			case "with_errors":
				scope = "with_errors"
			case "without_errors":
				scope = "without_errors"
			default:
				scope = "all"
			}
			continue
		}
		filtered = append(filtered, f)
	}
	return scope, filtered
}

// withErrorTracePrefilter narrows a "with errors" search to the traces that
// have an errored span matching the same filters, before grouping. Grouping
// every trace of a long window and dropping the clean ones afterwards
// (HAVING) exceeds the per-query memory budget; the result is the same.
func withErrorTracePrefilter(scope string, clauses []string) []string {
	if scope != "with_errors" {
		return clauses
	}
	inner := append(append([]string{}, clauses...), traceErrorStatusExpr)
	return append(clauses, "TraceId IN (SELECT TraceId FROM telemetry.otel_traces WHERE "+strings.Join(inner, " AND ")+")")
}
