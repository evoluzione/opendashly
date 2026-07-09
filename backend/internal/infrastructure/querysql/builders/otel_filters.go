package builders

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FilterItem represents a single filter condition.
type FilterItem struct {
	Connector string `json:"connector"` // "AND", "OR"
	Key       string `json:"key"`
	Operator  string `json:"operator"` // "=", "!=", "contains", etc.
	Value     string `json:"value"`
}

// filterListForSignal removes filters that are irrelevant for a specific signal.
// This avoids expensive scans caused by applying log-only filters to traces.
func filterListForSignal(signal string, filterList []FilterItem) []FilterItem {
	if len(filterList) == 0 {
		return filterList
	}
	out := make([]FilterItem, 0, len(filterList))
	for _, item := range filterList {
		key := strings.ToLower(strings.TrimSpace(item.Key))
		skip := false

		switch signal {
		case "logs":
			skip = isTraceOnlyFilterKey(key)
		case "traces":
			skip = isLogOnlyFilterKey(key)
		}

		if !skip {
			out = append(out, item)
		}
	}
	return out
}

func isLogOnlyFilterKey(key string) bool {
	switch key {
	case "body", "message", "log.body", "log_message", "severity":
		return true
	default:
		return false
	}
}

func isTraceOnlyFilterKey(key string) bool {
	switch key {
	case "trace_error_scope", "span_name", "trace_or_span", "duration_ms":
		return true
	default:
		return false
	}
}

func buildOtelClauses(timeColumn string, filters map[string]string, filterList []FilterItem, from, to time.Time, serviceColumn, traceColumn, severityColumn string, attributeColumns []string) []string {
	clauses := []string{}
	if !from.IsZero() {
		clauses = append(clauses, timeColumn+" >= "+formatDateTime64(from))
	}
	if !to.IsZero() {
		clauses = append(clauses, timeColumn+" <= "+formatDateTime64(to))
	}
	// If we have a filterList (new mode), use that.
	if len(filterList) > 0 {
		// New logic with AND/OR
		// We treat the list as a sequence.
		// For the first item, Connector is ignored (effectively AND with time range).
		// We group the time range logic as base.
		// Actually, standard SQL: WHERE (time conditions) AND ( (filter1) OR (filter2) AND (filter3) )
		// So we construct the filter part separately.

		filterPart := ""
		for i, f := range filterList {
			clause := buildSingleClause(f.Key, f.Value, f.Operator, serviceColumn, traceColumn, severityColumn, attributeColumns)
			if clause == "" {
				continue
			}

			conn := " AND "
			if i > 0 {
				if strings.ToUpper(f.Connector) == "OR" {
					conn = " OR "
				}
			}

			filterPart += conn + clause
		}

		if filterPart != "" {
			// Remove leading AND if strictly needed, but here we append to time clauses
			// logic: (Time) AND (Filters)
			// Wait, the caller joins 'clauses' with AND.
			// If we return multiple clauses, they get ANDed.
			// So we should return one combined clause for the filters if there are mixed operators?
			// Yes.

			// Trim leading " AND " or " OR " just in case, though our loop handles i>0.
			if strings.HasPrefix(filterPart, " AND ") {
				filterPart = filterPart[5:]
			} else if strings.HasPrefix(filterPart, " OR ") {
				filterPart = filterPart[4:]
			}

			clauses = append(clauses, "("+filterPart+")")
		}

	} else {
		// Legacy map-based behavior (implicit AND)
		for k, v := range filters {
			clause := buildSingleClause(k, v, "=", serviceColumn, traceColumn, severityColumn, attributeColumns)
			if clause != "" {
				clauses = append(clauses, clause)
			}
		}
	}
	return clauses
}

func buildSingleClause(k, v, op string, serviceColumn, traceColumn, severityColumn string, attributeColumns []string) string {
	if k == "" {
		return ""
	}
	escapedKey := EscapeLiteral(k)
	escapedValue := EscapeLiteral(v)

	// Default operator to =
	if op == "" {
		op = "="
	}

	sqlOp := "="
	sqlValue := "'" + escapedValue + "'"

	switch strings.ToLower(op) {
	case "=":
		sqlOp = "="
	case "!=":
		sqlOp = "!="
	case ">":
		sqlOp = ">"
	case "<":
		sqlOp = "<"
	case ">=":
		sqlOp = ">="
	case "<=":
		sqlOp = "<="
	case "contains":
		sqlOp = "ILIKE"
		sqlValue = "'%" + escapedValue + "%'"
		// Add more as needed
	}

	switch k {
	case "service.name":
		if serviceColumn != "" {
			return fmt.Sprintf("%s %s %s", serviceColumn, sqlOp, sqlValue)
		}
	case "trace_id":
		if traceColumn != "" {
			return fmt.Sprintf("%s %s %s", traceColumn, sqlOp, sqlValue)
		}
	case "span_name":
		// Heuristic: Only applies to Traces (traceColumn is set, severityColumn is empty).
		// Logs have traceColumn set but severityColumn set.
		if traceColumn != "" && severityColumn == "" {
			if op == "contains" || op == "" {
				return fmt.Sprintf("SpanName ILIKE '%%%s%%'", escapedValue)
			}
			return fmt.Sprintf("SpanName %s %s", sqlOp, sqlValue)
		}
	case "trace_or_span":
		// Unified trace search: match TraceId exactly OR SpanName by substring.
		if traceColumn != "" && severityColumn == "" {
			return fmt.Sprintf("(%s = '%s' OR SpanName ILIKE '%%%s%%')", traceColumn, escapedValue, escapedValue)
		}
	case "duration_ms":
		// Trace duration is stored in nanoseconds as Duration.
		// Expose a manual filter in milliseconds for UX.
		if traceColumn != "" && severityColumn == "" {
			number, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil {
				return ""
			}
			switch sqlOp {
			case "=", "!=", ">", "<", ">=", "<=":
				// valid
			default:
				return ""
			}
			return fmt.Sprintf("(Duration / 1000000.0) %s %s", sqlOp, strconv.FormatFloat(number, 'f', -1, 64))
		}
	case "severity":
		if severityColumn != "" && !isAllSeverityValue(v) {
			if op == "=" || op == "" {
				parts := strings.Split(v, ",")
				severityClauses := make([]string, 0, len(parts))
				for _, part := range parts {
					raw := strings.TrimSpace(part)
					if isAllSeverityValue(raw) {
						continue
					}
					upperValue := strings.ToUpper(EscapeLiteral(raw))
					allowed := []string{upperValue}
					switch upperValue {
					case "TRACE":
						allowed = append(allowed, "VERBOSE", "TRACE1", "TRACE2", "TRACE3", "TRACE4")
					case "DEBUG":
						allowed = append(allowed, "DBG", "DEBUG1", "DEBUG2", "DEBUG3", "DEBUG4")
					case "INFO":
						allowed = append(allowed, "INFORMATION", "INFO1", "INFO2", "INFO3", "INFO4")
					case "WARN":
						allowed = append(allowed, "WARNING", "WARNINGS", "WARN1", "WARN2", "WARN3", "WARN4")
					case "ERROR":
						allowed = append(allowed, "ERR", "ERRORS", "SEVERE", "ERROR1", "ERROR2", "ERROR3", "ERROR4")
					case "FATAL":
						allowed = append(allowed, "CRITICAL", "CRIT", "ALERT", "EMERG", "EMERGENCY", "FATAL1", "FATAL2", "FATAL3", "FATAL4")
					}
					allowedList := "'" + strings.Join(allowed, "','") + "'"

					singleSeverityClauses := []string{
						fmt.Sprintf("upper(%s) IN (%s)", severityColumn, allowedList),
					}
					if upperValue == "INFO" {
						singleSeverityClauses = append(singleSeverityClauses, fmt.Sprintf("(%s = '' OR isNull(%s))", severityColumn, severityColumn))
					}
					for _, column := range attributeColumns {
						singleSeverityClauses = append(singleSeverityClauses,
							fmt.Sprintf("upper(%s['severity']) IN (%s)", column, allowedList),
							fmt.Sprintf("upper(%s['level']) IN (%s)", column, allowedList),
						)
					}
					severityClauses = append(severityClauses, "("+strings.Join(singleSeverityClauses, " OR ")+")")
				}
				if len(severityClauses) == 0 {
					return ""
				}
				return "(" + strings.Join(severityClauses, " OR ") + ")"
			} else {
				return fmt.Sprintf("%s %s %s", severityColumn, sqlOp, sqlValue)
			}
		}
	case "trace_error_scope":
		// Handled by traces query builder using HAVING on aggregated errorCount.
		return ""
	default:
		if len(attributeColumns) == 0 {
			return ""
		}
		attrClauses := make([]string, 0, len(attributeColumns))
		for _, column := range attributeColumns {
			attrClauses = append(attrClauses, fmt.Sprintf("%s['%s'] %s %s", column, escapedKey, sqlOp, sqlValue))
		}
		if len(attrClauses) == 1 {
			return attrClauses[0]
		} else {
			return "(" + strings.Join(attrClauses, " OR ") + ")"
		}
	}
	return ""
}

func formatDateTime64(value time.Time) string {
	return "toDateTime64('" + value.UTC().Format("2006-01-02 15:04:05.000000000") + "', 9)"
}

func isAllSeverityValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed == "" || strings.EqualFold(trimmed, "all") || strings.EqualFold(trimmed, "tutti")
}

func EscapeLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}

// EscapeTraceID ensures trace IDs are safe for direct SQL interpolation.
func EscapeTraceID(value string) string {
	return EscapeLiteral(value)
}
