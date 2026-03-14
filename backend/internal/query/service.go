package query

import (
	"context"
	"log"
	"opendashly/backend/internal/storage"
	"sync"
)

// Service handles query execution.
type Service struct {
	Storage *storage.Client
	Debug   bool

	cacheInit    sync.Once
	cache        *cacheManager
	runnerInit   sync.Once
	signalRunner signalRunner
}

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
		log.Printf("query.service.run built queries: logs=%q traces=%q metrics=%q", queries.logs, queries.traces, queries.metrics)
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
		log.Printf("query.service.run complete: runId=%s logs=%d traces=%d metrics=%d", result.RunID, result.Summary.LogCount, result.Summary.TraceCount, result.Summary.MetricCount)
	}
	return result, nil
}

func (s *Service) getCacheManager() *cacheManager {
	s.cacheInit.Do(func() {
		s.cache = newCacheManager(servicesCacheTTL, attributesCacheTTL)
	})
	return s.cache
}

func (s *Service) getSignalRunner() signalRunner {
	if s.signalRunner != nil {
		return s.signalRunner
	}
	s.runnerInit.Do(func() {
		if s.signalRunner == nil {
			s.signalRunner = defaultSignalOrchestrator()
		}
	})
	return s.signalRunner
}

func (s *Service) getServicesFromCache() ([]string, bool) {
	return s.getCacheManager().getServices()
}

func (s *Service) setServicesCache(values []string) {
	s.getCacheManager().setServices(values)
}

func (s *Service) getAttributesFromCache(search string) ([]string, bool) {
	return s.getCacheManager().getAttributes(search)
}

func (s *Service) setAttributesCache(search string, values []string) {
	s.getCacheManager().setAttributes(search, values)
}
