package builders

import "strings"

// BuildLogsQuery creates a ClickHouse SQL statement for logs.
func BuildLogsQuery(filters map[string]string) string {
	base := "SELECT timestamp, severity, body, attributes, trace_id, span_id FROM telemetry.logs"
	clauses := []string{}
	for k, v := range filters {
		clauses = append(clauses, "attributes['"+k+"'] = '"+v+"'")
	}
	if len(clauses) == 0 {
		return base
	}
	return base + " WHERE " + strings.Join(clauses, " AND ")
}
