package query

import (
	"context"
	"log"
)

// Run executes an ad-hoc query and returns results.
func (s *Service) Run(ctx context.Context, req QueryRequest) (*QueryRunResult, error) {
	pagination := normalizeRunPagination(req)

	if s.Debug {
		log.Printf("query.service.run start: page=%d limit=%d offset=%d filters=%d", pagination.page, pagination.limit, pagination.offset, len(req.Filters))
	}
	if s.Storage == nil {
		log.Printf("query.service.run skipped: storage not configured")
		return emptyResult(pagination.page, pagination.limit), nil
	}

	signals := requestedSignals(req.Signals)
	queries, err := buildSignalQueries(req, pagination)
	if err != nil {
		return nil, err
	}
	if s.Debug {
		log.Printf("DEBUG: executing logsQuery: %s", queries.logs)
		log.Printf("query.service.run built queries: logs=%q traces=%q", queries.logs, queries.traces)
	}
	signalResult := s.getSignalRunner().run(ctx, s.Storage.Conn, queries, signals, pagination.limit)
	signalErrors := signalResult.signalErrors
	status, err := determineQueryRunStatus(signals, signalErrors)
	if err != nil {
		return nil, err
	}
	if status == "partial" {
		log.Printf("query.service.run partial: failedSignals=%s", joinSignalErrors(signalErrors))
	}

	result := assembleQueryRunResult(pagination, status, signalResult)
	if s.Debug {
		log.Printf("query.service.run complete: runId=%s logs=%d traces=%d", result.RunID, result.Summary.LogCount, result.Summary.TraceCount)
	}
	return result, nil
}
