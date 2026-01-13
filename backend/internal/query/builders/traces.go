package builders

import "strings"

// BuildTracesQuery creates a ClickHouse SQL statement for traces.
func BuildTracesQuery(filters map[string]string) string {
	base := "SELECT trace_id, span_id, parent_span_id, name, start_time, end_time, status, attributes FROM telemetry.traces"
	clauses := []string{}
	for k, v := range filters {
		clauses = append(clauses, "attributes['"+k+"'] = '"+v+"'")
	}
	if len(clauses) == 0 {
		return base
	}
	return base + " WHERE " + strings.Join(clauses, " AND ")
}
