package query

import (
	"context"
	"fmt"
	"time"

	"opendashly/backend/internal/infrastructure/querysql"
	"opendashly/backend/internal/infrastructure/storage"
)

// RelatedService fetches correlated telemetry for a trace.
type RelatedService struct {
	Storage *storage.Client
}

// Related returns related logs and metrics for a trace.
func (s *RelatedService) Related(ctx context.Context, traceID string) (Results, error) {
	if s.Storage == nil {
		return Results{
			Logs:    []any{},
			Traces:  []any{},
			Metrics: []any{},
		}, nil
	}

	logsQuery := querysql.BuildLogsQuery(BuildRelatedFilters(traceID), nil, time.Time{}, time.Time{}, 50, 0, nil)
	logs, err := fetchLogs(ctx, s.Storage.Conn, logsQuery)
	if err != nil {
		return Results{}, err
	}

	metricsQuery := buildRelatedMetricsQuery(traceID, 50)
	metrics, err := fetchMetrics(ctx, s.Storage.Conn, metricsQuery)
	if err != nil {
		return Results{}, err
	}

	return Results{
		Logs:    wrapAny(logs),
		Traces:  []any{},
		Metrics: wrapAny(metrics),
	}, nil
}

func buildRelatedMetricsQuery(traceID string, limit int) string {
	escapedTraceID := querysql.EscapeTraceID(traceID)
	base := "SELECT MetricName AS name, MetricUnit AS unit, TimeUnix AS timestamp, Value AS value FROM telemetry.otel_metrics_sum WHERE has(`Exemplars.TraceId`, '" + escapedTraceID + "')"
	gauge := "SELECT MetricName AS name, MetricUnit AS unit, TimeUnix AS timestamp, Value AS value FROM telemetry.otel_metrics_gauge WHERE has(`Exemplars.TraceId`, '" + escapedTraceID + "')"
	query := "SELECT name, unit, timestamp, value FROM (" + base + " UNION ALL " + gauge + ") ORDER BY timestamp DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	return query
}
