package query

import (
	"strings"

	"opendashly/backend/internal/query/builders"
)

var defaultLogAttributeSeed = []string{
	"service.name",
	"severity",
	"trace_id",
	"span_id",
	"http.method",
	"http.route",
	"http.status_code",
	"error.type",
	"error.message",
}

func buildLogAttributeKeysQuery(search string) string {
	query := "SELECT DISTINCT key FROM (" +
		"SELECT arrayJoin(mapKeys(ResourceAttributes)) AS key FROM (" +
		"SELECT ResourceAttributes FROM telemetry.otel_logs ORDER BY Timestamp DESC LIMIT 50000" +
		") " +
		"UNION ALL " +
		"SELECT arrayJoin(mapKeys(LogAttributes)) AS key FROM (" +
		"SELECT LogAttributes FROM telemetry.otel_logs ORDER BY Timestamp DESC LIMIT 50000" +
		") " +
		"UNION ALL SELECT 'service.name' AS key " +
		"UNION ALL SELECT 'severity' AS key " +
		"UNION ALL SELECT 'trace_id' AS key " +
		"UNION ALL SELECT 'span_id' AS key " +
		"UNION ALL SELECT 'http.method' AS key " +
		"UNION ALL SELECT 'http.route' AS key " +
		"UNION ALL SELECT 'http.status_code' AS key " +
		"UNION ALL SELECT 'error.type' AS key " +
		"UNION ALL SELECT 'error.message' AS key " +
		") WHERE key != ''"
	if search != "" {
		escaped := builders.EscapeLiteral(search)
		query += " AND key ILIKE '%" + escaped + "%'"
	}
	query += " ORDER BY key LIMIT 100"
	return query
}

func fallbackLogAttributeKeys(search string) []string {
	searchLower := strings.ToLower(strings.TrimSpace(search))
	filtered := make([]string, 0, len(defaultLogAttributeSeed))
	for _, key := range defaultLogAttributeSeed {
		if searchLower == "" || strings.Contains(strings.ToLower(key), searchLower) {
			filtered = append(filtered, key)
		}
	}
	return filtered
}
