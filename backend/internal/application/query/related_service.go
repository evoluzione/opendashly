package query

import (
	"context"
	"time"

	"opendashly/backend/internal/infrastructure/querysql"
	"opendashly/backend/internal/infrastructure/storage"
)

// RelatedService fetches correlated telemetry for a trace.
type RelatedService struct {
	Storage *storage.Client
}

// Related returns related logs for a trace.
func (s *RelatedService) Related(ctx context.Context, traceID string) (Results, error) {
	if s.Storage == nil {
		return Results{
			Logs:    []any{},
			Traces:  []any{},
		}, nil
	}

	logsQuery := querysql.BuildLogsQuery(BuildRelatedFilters(traceID), nil, time.Time{}, time.Time{}, 50, 0, nil)
	logs, err := fetchLogs(ctx, s.Storage.Conn, logsQuery)
	if err != nil {
		return Results{}, err
	}

	return Results{
		Logs:   wrapAny(logs),
		Traces: []any{},
	}, nil
}
