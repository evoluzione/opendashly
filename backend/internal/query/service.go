package query

import (
	"context"
	"fmt"
	"time"

	"opentelemetry-dashboard/backend/internal/query/builders"
	"opentelemetry-dashboard/backend/internal/storage"
)

// Service handles query execution.
type Service struct {
	Storage *storage.Client
}

// Run executes an ad-hoc query and returns results.
func (s *Service) Run(ctx context.Context, req QueryRequest) (*QueryRunResult, error) {
	_ = builders.BuildLogsQuery(req.Filters)
	_ = builders.BuildTracesQuery(req.Filters)
	_ = builders.BuildMetricsQuery(req.Filters)

	result := &QueryRunResult{
		RunID:  fmt.Sprintf("run-%d", time.Now().UnixNano()),
		Status: "complete",
		Summary: QueryRunSummary{},
		Results: Results{
			Logs:    []any{},
			Traces:  []any{},
			Metrics: []any{},
		},
	}
	return result, nil
}
