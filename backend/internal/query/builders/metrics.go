package builders

import "strings"

// BuildMetricsQuery creates a ClickHouse SQL statement for metrics.
func BuildMetricsQuery(filters map[string]string) string {
	base := "SELECT name, unit, timestamp, value, attributes FROM telemetry.metrics"
	clauses := []string{}
	for k, v := range filters {
		clauses = append(clauses, "attributes['"+k+"'] = '"+v+"'")
	}
	if len(clauses) == 0 {
		return base
	}
	return base + " WHERE " + strings.Join(clauses, " AND ")
}
