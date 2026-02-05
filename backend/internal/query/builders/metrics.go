package builders

import (
	"strconv"
	"strings"
	"time"
)

// BuildMetricsQuery creates a ClickHouse SQL statement for metrics.
func BuildMetricsQuery(filters map[string]string, filterList []FilterItem, from, to time.Time, limit, offset int) string {
	// Strategy: Union sum and gauge tables
	sumBase := "SELECT MetricName AS name, MetricUnit AS unit, TimeUnix AS timestamp, Value AS value FROM telemetry.otel_metrics_sum"
	gaugeBase := "SELECT MetricName AS name, MetricUnit AS unit, TimeUnix AS timestamp, Value AS value FROM telemetry.otel_metrics_gauge"
	clauses := buildOtelClauses("TimeUnix", filters, filterList, from, to, "ServiceName", "", "", []string{"ResourceAttributes", "Attributes"})
	sumQuery := sumBase
	gaugeQuery := gaugeBase
	if len(clauses) > 0 {
		where := " WHERE " + strings.Join(clauses, " AND ")
		sumQuery += where
		gaugeQuery += where
	}
	query := "SELECT name, unit, timestamp, value FROM (" + sumQuery + " UNION ALL " + gaugeQuery + ")"
	query += " ORDER BY timestamp DESC"
	if limit > 0 {
		query += " LIMIT " + strconv.Itoa(limit)
		if offset > 0 {
			query += " OFFSET " + strconv.Itoa(offset)
		}
	}
	return query
}

// BuildMetricsCountQuery creates a count query for metrics.
func BuildMetricsCountQuery(filters map[string]string, filterList []FilterItem, from, to time.Time) string {
	// Strategy: Union sum and gauge tables for counting
	sumBase := "SELECT MetricName, MetricUnit FROM telemetry.otel_metrics_sum"
	gaugeBase := "SELECT MetricName, MetricUnit FROM telemetry.otel_metrics_gauge"
	clauses := buildOtelClauses("TimeUnix", filters, filterList, from, to, "ServiceName", "", "", []string{"ResourceAttributes", "Attributes"})

	sumQuery := sumBase
	gaugeQuery := gaugeBase
	if len(clauses) > 0 {
		where := " WHERE " + strings.Join(clauses, " AND ")
		sumQuery += where
		gaugeQuery += where
	}
	// We count distinct series (Name + Unit)
	query := "SELECT uniqExact(tuple(MetricName, MetricUnit)) FROM (" + sumQuery + " UNION ALL " + gaugeQuery + ")"
	return query
}
