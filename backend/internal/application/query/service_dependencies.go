package query

import (
	"opendashly/backend/internal/infrastructure/storage"
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
