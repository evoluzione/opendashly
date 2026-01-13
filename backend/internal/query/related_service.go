package query

import (
	"context"

	"opentelemetry-dashboard/backend/internal/query/builders"
)

// RelatedService fetches correlated telemetry for a trace.
type RelatedService struct{}

// Related returns related logs and metrics for a trace.
func (s *RelatedService) Related(ctx context.Context, traceID string) (Results, error) {
	_ = builders.BuildLogsQuery(BuildRelatedFilters(traceID))
	_ = builders.BuildMetricsQuery(BuildRelatedFilters(traceID))

	return Results{
		Logs:    []any{},
		Traces:  []any{},
		Metrics: []any{},
	}, nil
}
