package query

import (
	"fmt"
	"time"

	"opendashly/backend/internal/infrastructure/querysql"
)

type runPagination struct {
	page      int
	limit     int
	offset    int
	readLimit int
}

func normalizeRunPagination(req QueryRequest) runPagination {
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 100
	}
	return runPagination{
		page:      page,
		limit:     limit,
		offset:    (page - 1) * limit,
		readLimit: limit + 1,
	}
}

func buildSignalQueries(req QueryRequest, pagination runPagination) (signalQueries, error) {
	logsCursor, err := decodeLogsCursor(req.LogsCursor)
	if err != nil {
		return signalQueries{}, fmt.Errorf("decode logs cursor: %w", err)
	}
	tracesCursor, err := decodeTracesCursor(req.TracesCursor)
	if err != nil {
		return signalQueries{}, fmt.Errorf("decode traces cursor: %w", err)
	}

	return signalQueries{
		logs: querysql.BuildLogsQuery(
			req.Filters,
			req.FilterList,
			req.TimeRange.From,
			req.TimeRange.To,
			pagination.readLimit,
			pagination.offset,
			logsCursor,
		),
		traces: querysql.BuildTracesQuery(
			req.Filters,
			req.FilterList,
			req.TimeRange.From,
			req.TimeRange.To,
			pagination.readLimit,
			pagination.offset,
			tracesCursor,
		),
		metrics: querysql.BuildMetricsQuery(
			req.Filters,
			req.FilterList,
			req.TimeRange.From,
			req.TimeRange.To,
			pagination.readLimit,
			pagination.offset,
		),
	}, nil
}

func assembleQueryRunResult(pagination runPagination, status string, signalResult signalExecutionResult) *QueryRunResult {
	result := &QueryRunResult{
		RunID:  fmt.Sprintf("run-%d", time.Now().UnixNano()),
		Status: status,
		Pagination: PaginationSet{
			Logs:    buildPagination(pagination.page, pagination.limit, signalResult.logsHasNext, signalResult.logsNextCursor),
			Traces:  buildPagination(pagination.page, pagination.limit, signalResult.tracesHasNext, signalResult.tracesNextCursor),
			Metrics: buildPagination(pagination.page, pagination.limit, signalResult.metricsHasNext, ""),
		},
		Results: Results{
			Logs:    wrapAny(signalResult.logs),
			Traces:  wrapAny(signalResult.traces),
			Metrics: wrapAny(signalResult.metrics),
		},
	}
	if len(signalResult.signalErrors) > 0 {
		result.SignalErrors = signalResult.signalErrors
	}
	result.Summary = QueryRunSummary{
		LogCount:    len(signalResult.logs),
		TraceCount:  len(signalResult.traces),
		MetricCount: len(signalResult.metrics),
	}
	return result
}
