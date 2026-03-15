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
	return query
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
		tokenClauses = append(tokenClauses,
			fmt.Sprintf("(Body ILIKE '%s' OR arrayExists(v -> v ILIKE '%s', mapValues(LogAttributes)) OR arrayExists(v -> v ILIKE '%s', mapValues(ResourceAttributes)))", like, like, like),
		)
	}
	return "(" + strings.Join(tokenClauses, " AND ") + ")"
}
