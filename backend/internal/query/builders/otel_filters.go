package builders

import (
	"strings"
	"time"
)

func buildOtelClauses(timeColumn string, filters map[string]string, from, to time.Time, serviceColumn, traceColumn, severityColumn string, attributeColumns []string) []string {
	clauses := []string{}
	if !from.IsZero() {
		clauses = append(clauses, timeColumn+" >= "+formatDateTime64(from))
	}
	if !to.IsZero() {
		clauses = append(clauses, timeColumn+" <= "+formatDateTime64(to))
	}
	for k, v := range filters {
		escapedKey := escapeLiteral(k)
		escapedValue := escapeLiteral(v)
		switch k {
		case "service.name":
			if serviceColumn != "" {
				clauses = append(clauses, serviceColumn+" = '"+escapedValue+"'")
			}
		case "trace_id":
			if traceColumn != "" {
				clauses = append(clauses, traceColumn+" = '"+escapedValue+"'")
			}
		case "severity":
			if severityColumn != "" && v != "Tutti" {
				clauses = append(clauses, "upper("+severityColumn+") = '"+strings.ToUpper(escapedValue)+"'")
			}
		default:
			if len(attributeColumns) == 0 {
				continue
			}
			attrClauses := make([]string, 0, len(attributeColumns))
			for _, column := range attributeColumns {
				attrClauses = append(attrClauses, column+"['"+escapedKey+"'] = '"+escapedValue+"'")
			}
			if len(attrClauses) == 1 {
				clauses = append(clauses, attrClauses[0])
			} else {
				clauses = append(clauses, "("+strings.Join(attrClauses, " OR ")+")")
			}
		}
	}
	return clauses
}

func formatDateTime64(value time.Time) string {
	return "toDateTime64('" + value.UTC().Format("2006-01-02 15:04:05.000000000") + "', 9)"
}

func escapeLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}

// EscapeTraceID ensures trace IDs are safe for direct SQL interpolation.
func EscapeTraceID(value string) string {
	return escapeLiteral(value)
}
