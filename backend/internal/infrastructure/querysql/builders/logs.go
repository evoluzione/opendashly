package builders

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"opendashly/backend/internal/infrastructure/config"
)

type LogsPageCursor struct {
	Timestamp time.Time
	TraceID   string
	SpanID    string
}

func rawTelemetryQuerySettings() string {
	return fmt.Sprintf(" SETTINGS max_execution_time = %d, max_threads = 1, max_memory_usage = 67108864, max_bytes_before_external_group_by = 16777216, max_bytes_before_external_sort = 16777216", config.Auto().ClickHouseMaxExecSec)
}

// BuildLogsQuery creates a ClickHouse SQL statement for logs.
//
// Two-step pattern: an inner top-N subquery selects only the PK columns
// (Timestamp, TraceId, SpanId) needed for sorting + pagination, then the outer
// query materializes the wide Map columns (ResourceAttributes, LogAttributes)
// only for those N rows. This avoids OOM on multi-day windows where reading
// the wide columns across the full filtered set exceeds ClickHouse memory.
func BuildLogsQuery(filters map[string]string, filterList []FilterItem, from, to time.Time, limit, offset int, cursor *LogsPageCursor) string {
	filteredForLogs := filterListForSignal("logs", filterList)
	bodySearch, effectiveFilterList := extractLogsBodySearch(filteredForLogs)
	clauses := buildOtelClauses("Timestamp", filters, effectiveFilterList, from, to, "ServiceName", "TraceId", "SeverityText", []string{"ResourceAttributes", "LogAttributes"})
	if bodySearch != "" {
		if searchClause := buildLogsBodySearchClause(bodySearch); searchClause != "" {
			clauses = append(clauses, searchClause)
		}
	}
	if cursor != nil && !cursor.Timestamp.IsZero() {
		ts := formatDateTime64(cursor.Timestamp)
		traceID := EscapeLiteral(cursor.TraceID)
		spanID := EscapeLiteral(cursor.SpanID)
		clauses = append(clauses, fmt.Sprintf("(Timestamp < %s OR (Timestamp = %s AND TraceId < '%s') OR (Timestamp = %s AND TraceId = '%s' AND SpanId < '%s'))", ts, ts, traceID, ts, traceID, spanID))
	}

	whereClause := ""
	if len(clauses) > 0 {
		whereClause = " WHERE " + strings.Join(clauses, " AND ")
	}

	if limit <= 0 {
		// No pagination: keep the legacy single-pass shape.
		query := "SELECT Timestamp AS timestamp, SeverityText AS severity, Body AS body, TraceId AS traceId, SpanId AS spanId, ResourceAttributes AS resourceAttributes, LogAttributes AS logAttributes FROM telemetry.otel_logs"
		query += whereClause
		query += " ORDER BY Timestamp DESC, TraceId DESC, SpanId DESC"
		return query + rawTelemetryQuerySettings()
	}

	inner := "SELECT Timestamp, TraceId, SpanId FROM telemetry.otel_logs"
	inner += whereClause
	inner += " ORDER BY Timestamp DESC, TraceId DESC, SpanId DESC"
	inner += " LIMIT " + strconv.Itoa(limit)
	if offset > 0 && cursor == nil {
		inner += " OFFSET " + strconv.Itoa(offset)
	}

	query := "SELECT Timestamp AS timestamp, SeverityText AS severity, Body AS body, TraceId AS traceId, SpanId AS spanId, ResourceAttributes AS resourceAttributes, LogAttributes AS logAttributes FROM telemetry.otel_logs"
	query += whereClause
	if whereClause == "" {
		query += " WHERE "
	} else {
		query += " AND "
	}
	query += "(Timestamp, TraceId, SpanId) IN (" + inner + ")"
	query += " ORDER BY Timestamp DESC, TraceId DESC, SpanId DESC"
	return query + rawTelemetryQuerySettings()
}

// BuildLogsCountQuery creates a count query for logs.
func BuildLogsCountQuery(filters map[string]string, filterList []FilterItem, from, to time.Time) string {
	base := "SELECT count(*) FROM telemetry.otel_logs"
	filteredForLogs := filterListForSignal("logs", filterList)
	bodySearch, effectiveFilterList := extractLogsBodySearch(filteredForLogs)
	clauses := buildOtelClauses("Timestamp", filters, effectiveFilterList, from, to, "ServiceName", "TraceId", "SeverityText", []string{"ResourceAttributes", "LogAttributes"})
	if bodySearch != "" {
		if searchClause := buildLogsBodySearchClause(bodySearch); searchClause != "" {
			clauses = append(clauses, searchClause)
		}
	}
	query := base
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	return query + rawTelemetryQuerySettings()
}

func extractLogsBodySearch(filterList []FilterItem) (string, []FilterItem) {
	if len(filterList) == 0 {
		return "", filterList
	}
	bodySearch := ""
	filtered := make([]FilterItem, 0, len(filterList))
	for _, f := range filterList {
		switch strings.ToLower(strings.TrimSpace(f.Key)) {
		case "body", "message", "log.body", "log_message":
			if strings.EqualFold(strings.TrimSpace(f.Operator), "contains") || strings.TrimSpace(f.Operator) == "" {
				value := strings.TrimSpace(f.Value)
				if value != "" {
					bodySearch = value
				}
				continue
			}
		}
		filtered = append(filtered, f)
	}
	return bodySearch, filtered
}

func isSimpleToken(s string) bool {
	for _, c := range s {
		// ClickHouse hasTokenCaseInsensitive requires a single token needle
		// without separator characters (for example "_").
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return len(s) > 0
}

func buildLogsBodySearchClause(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	tokens := strings.Fields(trimmed)
	if len(tokens) == 0 {
		return ""
	}

	tokenClauses := make([]string, 0, len(tokens))
	for _, token := range tokens {
		escaped := EscapeLiteral(token)
		like := fmt.Sprintf("%%%s%%", escaped)
		var bodyClause string
		if isSimpleToken(token) {
			// Use hasTokenCaseInsensitive to leverage the tokenbf_v1 index on Body
			bodyClause = fmt.Sprintf("hasTokenCaseInsensitive(Body, '%s')", escaped)
		} else {
			bodyClause = fmt.Sprintf("Body ILIKE '%s'", like)
		}
		tokenClauses = append(tokenClauses,
			fmt.Sprintf("(%s OR arrayExists(v -> v ILIKE '%s', mapValues(LogAttributes)) OR arrayExists(v -> v ILIKE '%s', mapValues(ResourceAttributes)))", bodyClause, like, like),
		)
	}
	return "(" + strings.Join(tokenClauses, " AND ") + ")"
}
