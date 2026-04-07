package metrics

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"opendashly/backend/internal/infrastructure/storage"
)

// Service handles dashboard metrics calculations.
type Service struct {
	Storage   *storage.Client
	cache     *dashboardCache
	cacheOnce sync.Once
}

func (s *Service) getCache() *dashboardCache {
	s.cacheOnce.Do(func() {
		s.cache = newDashboardCache()
	})
	return s.cache
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

	// Check cache first
	if cached, ok := s.getCache().get(req); ok {
		log.Printf("metrics.service.GetDashboard: cache hit")
		return cached, nil
	}

	log.Printf("metrics.service.GetDashboard: from=%s to=%s service=%s",
		req.From.Format(time.RFC3339), req.To.Format(time.RFC3339), req.ServiceName)

	var (
		latencyDist    []LatencyBucket
		slowest        []EndpointLatency
		errorHotspots  []ErrorHotspot
		latencySeries  []LatencyPercentilePoint
		errorRateSeries []ErrorRatePoint
		statusCodes    []StatusCodeBreakdown
		topEndpoints   []EndpointThroughput
		apdex          ApdexScore
		throughput     ThroughputSummary
		timeSeries     []ThroughputPoint
		logVolume      []LogVolumePoint
		logLevels      []LogLevelCount
		errorRate      float64
	)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(4)

	g.Go(func() error {
		var err error
		latencyDist, err = s.getLatencyDistribution(gctx, req)
		if err != nil {
			log.Printf("metrics.service: latency distribution error: %v", err)
			return fmt.Errorf("latency distribution: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		slowest, err = s.getSlowestEndpoints(gctx, req)
		if err != nil {
			log.Printf("metrics.service: slowest endpoints error: %v", err)
			return fmt.Errorf("slowest endpoints: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		errorHotspots, err = s.getErrorHotspots(gctx, req)
		if err != nil {
			log.Printf("metrics.service: error hotspots error: %v", err)
			return fmt.Errorf("error hotspots: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		latencySeries, err = s.getLatencyPercentiles(gctx, req)
		if err != nil {
			log.Printf("metrics.service: latency percentiles error: %v", err)
			return fmt.Errorf("latency percentiles: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		errorRateSeries, err = s.getErrorRateSeries(gctx, req)
		if err != nil {
			log.Printf("metrics.service: error rate series error: %v", err)
			return fmt.Errorf("error rate series: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		statusCodes, err = s.getStatusCodeBreakdown(gctx, req)
		if err != nil {
			log.Printf("metrics.service: status code breakdown error: %v", err)
			return fmt.Errorf("status code breakdown: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		topEndpoints, err = s.getTopEndpointsThroughput(gctx, req)
		if err != nil {
			log.Printf("metrics.service: top endpoints error: %v", err)
			return fmt.Errorf("top endpoints: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		apdex, err = s.getApdexScore(gctx, req)
		if err != nil {
			log.Printf("metrics.service: apdex score error: %v", err)
			return fmt.Errorf("apdex score: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		throughput, timeSeries, err = s.getThroughput(gctx, req)
		if err != nil {
			log.Printf("metrics.service: throughput error: %v", err)
			return fmt.Errorf("throughput: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		logVolume, err = s.getLogVolume(gctx, req)
		if err != nil {
			log.Printf("metrics.service: log volume error: %v", err)
			return fmt.Errorf("log volume: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		logLevels, err = s.getLogLevels(gctx, req)
		if err != nil {
			log.Printf("metrics.service: log levels error: %v", err)
			return fmt.Errorf("log levels: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		errorRate, err = s.getErrorRate(gctx, req)
		if err != nil {
			log.Printf("metrics.service: error rate error: %v", err)
			return fmt.Errorf("error rate: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	result := &DashboardResponse{
		Hotspots: HotspotsData{
			LatencyDistribution: latencyDist,
			SlowestEndpoints:    slowest,
			ErrorHotspots:       errorHotspots,
			TopEndpoints:        topEndpoints,
			StatusCodes:         statusCodes,
		},
		Satisfaction: SatisfactionData{
			Apdex:           apdex,
			ErrorRate:       errorRate,
			Throughput:      throughput,
			TimeSeries:      timeSeries,
			LatencySeries:   latencySeries,
			ErrorRateSeries: errorRateSeries,
		},
		Logs: LogsData{
			VolumeSeries: logVolume,
			Levels:       logLevels,
		},
	}

	s.getCache().set(req, result)
	return result, nil
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
		},
		Satisfaction: SatisfactionData{
			Apdex:      ApdexScore{Threshold: ApdexThresholdMs},
			ErrorRate:  0,
			Throughput: ThroughputSummary{},
			TimeSeries: []ThroughputPoint{},
		},
	}
}
