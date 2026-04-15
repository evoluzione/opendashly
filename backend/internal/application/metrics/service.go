package metrics

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"opendashly/backend/internal/infrastructure/storage"
)

// Service handles dashboard metrics calculations.
type Service struct {
	Storage          *storage.Client
	FreshCacheTTL    time.Duration
	StaleCacheTTL    time.Duration
	QueryParallelism int

	fetchers  *dashboardFetchers
	cache     *dashboardCache
	cacheOnce sync.Once
}

type dashboardFetchers struct {
	latencyDistribution func(context.Context, DashboardRequest) ([]LatencyBucket, error)
	slowestEndpoints    func(context.Context, DashboardRequest) ([]EndpointLatency, error)
	errorHotspots       func(context.Context, DashboardRequest) ([]ErrorHotspot, error)
	latencyPercentiles  func(context.Context, DashboardRequest) ([]LatencyPercentilePoint, error)
	errorRateSeries     func(context.Context, DashboardRequest) ([]ErrorRatePoint, error)
	statusCodeBreakdown func(context.Context, DashboardRequest) ([]StatusCodeBreakdown, error)
	topEndpoints        func(context.Context, DashboardRequest) ([]EndpointThroughput, error)
	apdexScore          func(context.Context, DashboardRequest) (ApdexScore, error)
	throughput          func(context.Context, DashboardRequest) (ThroughputSummary, []ThroughputPoint, error)
	logVolume           func(context.Context, DashboardRequest) ([]LogVolumePoint, error)
	logLevels           func(context.Context, DashboardRequest) ([]LogLevelCount, error)
	errorRate           func(context.Context, DashboardRequest) (float64, error)
}

func (s *Service) getCache() *dashboardCache {
	s.cacheOnce.Do(func() {
		s.cache = newDashboardCache(s.FreshCacheTTL, s.StaleCacheTTL)
	})
	return s.cache
}

func (s *Service) getQueryParallelism() int {
	if s.QueryParallelism <= 0 {
		return 2
	}
	return s.QueryParallelism
}

func (s *Service) getFetchers() dashboardFetchers {
	if s.fetchers != nil {
		return *s.fetchers
	}
	return dashboardFetchers{
		latencyDistribution: s.getLatencyDistribution,
		slowestEndpoints:    s.getSlowestEndpoints,
		errorHotspots:       s.getErrorHotspots,
		latencyPercentiles:  s.getLatencyPercentiles,
		errorRateSeries:     s.getErrorRateSeries,
		statusCodeBreakdown: s.getStatusCodeBreakdown,
		topEndpoints:        s.getTopEndpointsThroughput,
		apdexScore:          s.getApdexScore,
		throughput:          s.getThroughput,
		logVolume:           s.getLogVolume,
		logLevels:           s.getLogLevels,
		errorRate:           s.getErrorRate,
	}
}

// toInt converts various numeric types to int for ClickHouse compatibility
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	default:
		return 0
	}
}

// GetDashboard retrieves all dashboard metrics.
func (s *Service) GetDashboard(ctx context.Context, req DashboardRequest) (*DashboardResponse, error) {
	if s.Storage == nil {
		log.Printf("metrics.service: storage not configured")
		return emptyResponse(), nil
	}

	cache := s.getCache()
	if cached, ok := cache.getFresh(req); ok {
		log.Printf("metrics.service.GetDashboard: cache hit")
		return cached, nil
	}
	staleSnapshot, hasStale := cache.getStale(req)

	log.Printf("metrics.service.GetDashboard: from=%s to=%s service=%s",
		req.From.Format(time.RFC3339), req.To.Format(time.RFC3339), req.ServiceName)

	latencyDist := []LatencyBucket{}
	slowest := []EndpointLatency{}
	errorHotspots := []ErrorHotspot{}
	latencySeries := []LatencyPercentilePoint{}
	errorRateSeries := []ErrorRatePoint{}
	statusCodes := []StatusCodeBreakdown{}
	topEndpoints := []EndpointThroughput{}
	apdex := ApdexScore{Threshold: ApdexThresholdMs}
	throughput := ThroughputSummary{}
	timeSeries := []ThroughputPoint{}
	logVolume := []LogVolumePoint{}
	logLevels := []LogLevelCount{}
	errorRate := float64(0)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(s.getQueryParallelism())
	fetchers := s.getFetchers()

	warnings := make([]string, 0, 4)
	var warningsMu sync.Mutex
	successfulSections := 0
	var successMu sync.Mutex

	markSuccess := func() {
		successMu.Lock()
		successfulSections++
		successMu.Unlock()
	}

	g.Go(func() error {
		section, err := fetchers.latencyDistribution(gctx, req)
		if err != nil {
			log.Printf("metrics.service: latency distribution error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					latencyDist = cloneOrEmptySlice(staleSnapshot.Hotspots.LatencyDistribution)
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "latency distribution", err)
				return nil
			}
			return fmt.Errorf("latency distribution: %w", err)
		}
		latencyDist = cloneOrEmptySlice(section)
		markSuccess()
		return nil
	})

	g.Go(func() error {
		section, err := fetchers.slowestEndpoints(gctx, req)
		if err != nil {
			log.Printf("metrics.service: slowest endpoints error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					slowest = cloneOrEmptySlice(staleSnapshot.Hotspots.SlowestEndpoints)
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "slowest endpoints", err)
				return nil
			}
			return fmt.Errorf("slowest endpoints: %w", err)
		}
		slowest = cloneOrEmptySlice(section)
		markSuccess()
		return nil
	})

	g.Go(func() error {
		section, err := fetchers.errorHotspots(gctx, req)
		if err != nil {
			log.Printf("metrics.service: error hotspots error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					errorHotspots = cloneOrEmptySlice(staleSnapshot.Hotspots.ErrorHotspots)
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "error hotspots", err)
				return nil
			}
			return fmt.Errorf("error hotspots: %w", err)
		}
		errorHotspots = cloneOrEmptySlice(section)
		markSuccess()
		return nil
	})

	g.Go(func() error {
		section, err := fetchers.latencyPercentiles(gctx, req)
		if err != nil {
			log.Printf("metrics.service: latency percentiles error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					latencySeries = cloneOrEmptySlice(staleSnapshot.Satisfaction.LatencySeries)
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "latency percentiles", err)
				return nil
			}
			return fmt.Errorf("latency percentiles: %w", err)
		}
		latencySeries = cloneOrEmptySlice(section)
		markSuccess()
		return nil
	})

	g.Go(func() error {
		section, err := fetchers.errorRateSeries(gctx, req)
		if err != nil {
			log.Printf("metrics.service: error rate series error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					errorRateSeries = cloneOrEmptySlice(staleSnapshot.Satisfaction.ErrorRateSeries)
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "error rate series", err)
				return nil
			}
			return fmt.Errorf("error rate series: %w", err)
		}
		errorRateSeries = cloneOrEmptySlice(section)
		markSuccess()
		return nil
	})

	g.Go(func() error {
		section, err := fetchers.statusCodeBreakdown(gctx, req)
		if err != nil {
			log.Printf("metrics.service: status code breakdown error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					statusCodes = cloneOrEmptySlice(staleSnapshot.Hotspots.StatusCodes)
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "status code breakdown", err)
				return nil
			}
			return fmt.Errorf("status code breakdown: %w", err)
		}
		statusCodes = cloneOrEmptySlice(section)
		markSuccess()
		return nil
	})

	g.Go(func() error {
		section, err := fetchers.topEndpoints(gctx, req)
		if err != nil {
			log.Printf("metrics.service: top endpoints error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					topEndpoints = cloneOrEmptySlice(staleSnapshot.Hotspots.TopEndpoints)
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "top endpoints", err)
				return nil
			}
			return fmt.Errorf("top endpoints: %w", err)
		}
		topEndpoints = cloneOrEmptySlice(section)
		markSuccess()
		return nil
	})

	g.Go(func() error {
		section, err := fetchers.apdexScore(gctx, req)
		if err != nil {
			log.Printf("metrics.service: apdex score error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					apdex = staleSnapshot.Satisfaction.Apdex
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "apdex score", err)
				return nil
			}
			return fmt.Errorf("apdex score: %w", err)
		}
		apdex = section
		markSuccess()
		return nil
	})

	g.Go(func() error {
		throughputSection, timeSeriesSection, err := fetchers.throughput(gctx, req)
		if err != nil {
			log.Printf("metrics.service: throughput error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					throughput = staleSnapshot.Satisfaction.Throughput
					timeSeries = cloneOrEmptySlice(staleSnapshot.Satisfaction.TimeSeries)
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "throughput", err)
				return nil
			}
			return fmt.Errorf("throughput: %w", err)
		}
		throughput = throughputSection
		timeSeries = cloneOrEmptySlice(timeSeriesSection)
		markSuccess()
		return nil
	})

	g.Go(func() error {
		section, err := fetchers.logVolume(gctx, req)
		if err != nil {
			log.Printf("metrics.service: log volume error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					logVolume = cloneOrEmptySlice(staleSnapshot.Logs.VolumeSeries)
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "log volume", err)
				return nil
			}
			return fmt.Errorf("log volume: %w", err)
		}
		logVolume = cloneOrEmptySlice(section)
		markSuccess()
		return nil
	})

	g.Go(func() error {
		section, err := fetchers.logLevels(gctx, req)
		if err != nil {
			log.Printf("metrics.service: log levels error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					logLevels = cloneOrEmptySlice(staleSnapshot.Logs.Levels)
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "log levels", err)
				return nil
			}
			return fmt.Errorf("log levels: %w", err)
		}
		logLevels = cloneOrEmptySlice(section)
		markSuccess()
		return nil
	})

	g.Go(func() error {
		section, err := fetchers.errorRate(gctx, req)
		if err != nil {
			log.Printf("metrics.service: error rate error: %v", err)
			if isRecoverableDashboardError(err) {
				if hasStale {
					errorRate = staleSnapshot.Satisfaction.ErrorRate
				}
				appendRecoverableDashboardWarning(&warningsMu, &warnings, "error rate", err)
				return nil
			}
			return fmt.Errorf("error rate: %w", err)
		}
		errorRate = section
		markSuccess()
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	result := &DashboardResponse{
		Hotspots: HotspotsData{
			LatencyDistribution: cloneOrEmptySlice(latencyDist),
			SlowestEndpoints:    cloneOrEmptySlice(slowest),
			ErrorHotspots:       cloneOrEmptySlice(errorHotspots),
			TopEndpoints:        cloneOrEmptySlice(topEndpoints),
			StatusCodes:         cloneOrEmptySlice(statusCodes),
		},
		Satisfaction: SatisfactionData{
			Apdex:           apdex,
			ErrorRate:       errorRate,
			Throughput:      throughput,
			TimeSeries:      cloneOrEmptySlice(timeSeries),
			LatencySeries:   cloneOrEmptySlice(latencySeries),
			ErrorRateSeries: cloneOrEmptySlice(errorRateSeries),
		},
		Logs: LogsData{
			VolumeSeries: cloneOrEmptySlice(logVolume),
			Levels:       cloneOrEmptySlice(logLevels),
		},
		Warnings: cloneOrEmptySlice(warnings),
	}

	if successfulSections == 0 && len(warnings) > 0 && hasStale {
		staleResult := cloneDashboardResponse(staleSnapshot)
		staleResult.Warnings = append(staleResult.Warnings, "serving stale dashboard snapshot due to temporary backend pressure")
		staleResult.Warnings = append(staleResult.Warnings, warnings...)
		return staleResult, nil
	}

	cache.set(req, result)
	return result, nil
}

func appendRecoverableDashboardWarning(mu *sync.Mutex, warnings *[]string, section string, err error) {
	if err == nil {
		return
	}
	mu.Lock()
	*warnings = append(*warnings, fmt.Sprintf("%s unavailable: %v", section, err))
	mu.Unlock()
}

func isRecoverableDashboardError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	patterns := []string{
		"memory limit exceeded",
		"overcommittracker",
		"timeout",
		"deadline exceeded",
		"temporarily unavailable",
		"context canceled",
	}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

func (s *Service) getLatencyDistribution(ctx context.Context, req DashboardRequest) ([]LatencyBucket, error) {
	query := BuildLatencyDistributionQuery(req.From, req.To, req.ServiceName)
	log.Printf("metrics.service.getLatencyDistribution: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	bucketCounts := make(map[int]int64)
	var total int64

	for rows.Next() {
		var bucketStart, bucketEnd int32
		var count uint64
		if err := rows.Scan(&bucketStart, &bucketEnd, &count); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		bucketCounts[int(bucketStart)] = int64(count)
		total += int64(count)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}

	// Build result with all buckets (even if zero)
	bucketDefs := GetLatencyBuckets()
	result := make([]LatencyBucket, len(bucketDefs))
	for i, def := range bucketDefs {
		count := bucketCounts[def.Start]
		var pct float64
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		result[i] = LatencyBucket{
			RangeStart: def.Start,
			RangeEnd:   def.End,
			Count:      count,
			Percentage: pct,
			Label:      def.Label,
		}
	}

	return result, nil
}

func (s *Service) getSlowestEndpoints(ctx context.Context, req DashboardRequest) ([]EndpointLatency, error) {
	query := BuildSlowestEndpointsQuery(req.From, req.To, req.ServiceName, 10)
	log.Printf("metrics.service.getSlowestEndpoints: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var results []EndpointLatency
	for rows.Next() {
		var endpoint, service string
		var avgMs, p50, p95, p99 float64
		var count uint64
		if err := rows.Scan(&endpoint, &service, &avgMs, &p50, &p95, &p99, &count); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, EndpointLatency{
			Endpoint: endpoint,
			Service:  service,
			AvgMs:    avgMs,
			P50:      p50,
			P95:      p95,
			P99:      p99,
			Count:    int64(count),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}

	if results == nil {
		results = []EndpointLatency{}
	}
	return results, nil
}

func (s *Service) getErrorHotspots(ctx context.Context, req DashboardRequest) ([]ErrorHotspot, error) {
	query := BuildErrorHotspotsQuery(req.From, req.To, req.ServiceName, 10)
	log.Printf("metrics.service.getErrorHotspots: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var results []ErrorHotspot
	for rows.Next() {
		var endpoint, service string
		var errorCount, totalCount uint64
		var errorRate float64
		if err := rows.Scan(&endpoint, &service, &errorCount, &totalCount, &errorRate); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, ErrorHotspot{
			Endpoint:   endpoint,
			Service:    service,
			ErrorCount: int64(errorCount),
			TotalCount: int64(totalCount),
			ErrorRate:  errorRate,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}

	if results == nil {
		results = []ErrorHotspot{}
	}
	return results, nil
}

func (s *Service) getLatencyPercentiles(ctx context.Context, req DashboardRequest) ([]LatencyPercentilePoint, error) {
	query := BuildLatencyPercentilesQuery(req.From, req.To, req.ServiceName)
	log.Printf("metrics.service.getLatencyPercentiles: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	results := []LatencyPercentilePoint{}
	for rows.Next() {
		var bucket time.Time
		var p50, p95, p99 float64
		if err := rows.Scan(&bucket, &p50, &p95, &p99); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, LatencyPercentilePoint{
			Timestamp: bucket,
			P50:       p50,
			P95:       p95,
			P99:       p99,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	return results, nil
}

func (s *Service) getErrorRateSeries(ctx context.Context, req DashboardRequest) ([]ErrorRatePoint, error) {
	query := BuildErrorRateTimeSeriesQuery(req.From, req.To, req.ServiceName)
	log.Printf("metrics.service.getErrorRateSeries: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	results := []ErrorRatePoint{}
	for rows.Next() {
		var bucket time.Time
		var errorCount, totalCount uint64
		var errorRate float64
		if err := rows.Scan(&bucket, &errorCount, &totalCount, &errorRate); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, ErrorRatePoint{
			Timestamp:  bucket,
			ErrorRate:  errorRate,
			ErrorCount: int64(errorCount),
			TotalCount: int64(totalCount),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	return results, nil
}

func (s *Service) getStatusCodeBreakdown(ctx context.Context, req DashboardRequest) ([]StatusCodeBreakdown, error) {
	query := BuildStatusCodeBreakdownQuery(req.From, req.To, req.ServiceName)
	log.Printf("metrics.service.getStatusCodeBreakdown: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	results := []StatusCodeBreakdown{}
	var total int64
	for rows.Next() {
		var code string
		var count uint64
		if err := rows.Scan(&code, &count); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, StatusCodeBreakdown{
			Code:  code,
			Count: int64(count),
		})
		total += int64(count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	for i := range results {
		if total > 0 {
			results[i].Percentage = float64(results[i].Count) / float64(total) * 100
		}
	}
	return results, nil
}

func (s *Service) getTopEndpointsThroughput(ctx context.Context, req DashboardRequest) ([]EndpointThroughput, error) {
	query := BuildTopEndpointsThroughputQuery(req.From, req.To, req.ServiceName, 10)
	log.Printf("metrics.service.getTopEndpointsThroughput: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	results := []EndpointThroughput{}
	for rows.Next() {
		var endpoint, service string
		var requestCount, errorCount uint64
		var errorRate float64
		if err := rows.Scan(&endpoint, &service, &requestCount, &errorCount, &errorRate); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, EndpointThroughput{
			Endpoint:     endpoint,
			Service:      service,
			RequestCount: int64(requestCount),
			ErrorCount:   int64(errorCount),
			ErrorRate:    errorRate,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	return results, nil
}

func (s *Service) getLogVolume(ctx context.Context, req DashboardRequest) ([]LogVolumePoint, error) {
	query := BuildLogVolumeQuery(req.From, req.To, req.ServiceName)
	log.Printf("metrics.service.getLogVolume: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	results := []LogVolumePoint{}
	for rows.Next() {
		var bucket time.Time
		var count uint64
		if err := rows.Scan(&bucket, &count); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, LogVolumePoint{Timestamp: bucket, Count: int64(count)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	return results, nil
}

func (s *Service) getLogLevels(ctx context.Context, req DashboardRequest) ([]LogLevelCount, error) {
	query := BuildLogLevelsQuery(req.From, req.To, req.ServiceName)
	log.Printf("metrics.service.getLogLevels: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	results := []LogLevelCount{}
	var total int64
	for rows.Next() {
		var level string
		var count uint64
		if err := rows.Scan(&level, &count); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, LogLevelCount{Level: level, Count: int64(count)})
		total += int64(count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	for i := range results {
		if total > 0 {
			results[i].Percentage = float64(results[i].Count) / float64(total) * 100
		}
	}
	return results, nil
}

func (s *Service) getApdexScore(ctx context.Context, req DashboardRequest) (ApdexScore, error) {
	query := BuildApdexQuery(req.From, req.To, req.ServiceName)
	log.Printf("metrics.service.getApdexScore: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return ApdexScore{}, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var satisfied, tolerating, frustrated, total uint64
	if rows.Next() {
		if err := rows.Scan(&satisfied, &tolerating, &frustrated, &total); err != nil {
			return ApdexScore{}, fmt.Errorf("scan: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return ApdexScore{}, fmt.Errorf("iterate: %w", err)
	}

	var score float64
	if total > 0 {
		// APDEX formula: (Satisfied + Tolerating/2) / Total
		score = (float64(satisfied) + float64(tolerating)/2) / float64(total)
	}

	return ApdexScore{
		Score:      score,
		Satisfied:  int64(satisfied),
		Tolerating: int64(tolerating),
		Frustrated: int64(frustrated),
		Total:      int64(total),
		Threshold:  ApdexThresholdMs,
	}, nil
}

func (s *Service) getThroughput(ctx context.Context, req DashboardRequest) (ThroughputSummary, []ThroughputPoint, error) {
	query := BuildThroughputQuery(req.From, req.To, req.ServiceName)
	log.Printf("metrics.service.getThroughput: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return ThroughputSummary{}, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var points []ThroughputPoint
	var totalRequests, totalErrors int64

	for rows.Next() {
		var timestamp time.Time
		var requestCount, errorCount uint64
		if err := rows.Scan(&timestamp, &requestCount, &errorCount); err != nil {
			return ThroughputSummary{}, nil, fmt.Errorf("scan: %w", err)
		}
		points = append(points, ThroughputPoint{
			Timestamp:    timestamp,
			RequestCount: int64(requestCount),
			ErrorCount:   int64(errorCount),
		})
		totalRequests += int64(requestCount)
		totalErrors += int64(errorCount)
	}

	if err := rows.Err(); err != nil {
		return ThroughputSummary{}, nil, fmt.Errorf("iterate: %w", err)
	}

	// Calculate requests/errors per minute using observed span first.
	// This avoids flattening values when requested range is very large.
	durationMinutes := req.To.Sub(req.From).Minutes()
	if len(points) > 1 {
		observedMinutes := points[len(points)-1].Timestamp.Sub(points[0].Timestamp).Minutes()
		if observedMinutes > 0 {
			durationMinutes = observedMinutes
		}
	} else if len(points) == 1 {
		durationMinutes = 1
	}
	var reqPerMin, errPerMin float64
	if durationMinutes > 0 {
		reqPerMin = float64(totalRequests) / durationMinutes
		errPerMin = float64(totalErrors) / durationMinutes
	}

	if points == nil {
		points = []ThroughputPoint{}
	}

	return ThroughputSummary{
		TotalRequests:  totalRequests,
		TotalErrors:    totalErrors,
		RequestsPerMin: reqPerMin,
		ErrorsPerMin:   errPerMin,
	}, points, nil
}

func (s *Service) getErrorRate(ctx context.Context, req DashboardRequest) (float64, error) {
	query := BuildErrorRateQuery(req.From, req.To, req.ServiceName)
	log.Printf("metrics.service.getErrorRate: executing query")

	rows, err := s.Storage.Conn.Query(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var errorCount, totalCount uint64
	var errorRate float64
	if rows.Next() {
		if err := rows.Scan(&errorCount, &totalCount, &errorRate); err != nil {
			return 0, fmt.Errorf("scan: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate: %w", err)
	}

	return errorRate, nil
}

func emptyResponse() *DashboardResponse {
	return &DashboardResponse{
		Hotspots: HotspotsData{
			LatencyDistribution: []LatencyBucket{},
			SlowestEndpoints:    []EndpointLatency{},
			ErrorHotspots:       []ErrorHotspot{},
			TopEndpoints:        []EndpointThroughput{},
			StatusCodes:         []StatusCodeBreakdown{},
		},
		Satisfaction: SatisfactionData{
			Apdex:           ApdexScore{Threshold: ApdexThresholdMs},
			ErrorRate:       0,
			Throughput:      ThroughputSummary{},
			TimeSeries:      []ThroughputPoint{},
			LatencySeries:   []LatencyPercentilePoint{},
			ErrorRateSeries: []ErrorRatePoint{},
		},
		Logs: LogsData{
			VolumeSeries: []LogVolumePoint{},
			Levels:       []LogLevelCount{},
		},
		Warnings: []string{},
	}
}
