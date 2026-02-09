package metrics

import (
	"context"
	"fmt"
	"log"
	"time"

	"opendashly/backend/internal/storage"
)

// Service handles dashboard metrics calculations.
type Service struct {
	Storage *storage.Client
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

	log.Printf("metrics.service.GetDashboard: from=%s to=%s service=%s",
		req.From.Format(time.RFC3339), req.To.Format(time.RFC3339), req.ServiceName)

	// Fetch all metrics
	latencyDist, err := s.getLatencyDistribution(ctx, req)
	if err != nil {
		log.Printf("metrics.service: latency distribution error: %v", err)
		return nil, fmt.Errorf("latency distribution: %w", err)
	}

	slowest, err := s.getSlowestEndpoints(ctx, req)
	if err != nil {
		log.Printf("metrics.service: slowest endpoints error: %v", err)
		return nil, fmt.Errorf("slowest endpoints: %w", err)
	}

	errorHotspots, err := s.getErrorHotspots(ctx, req)
	if err != nil {
		log.Printf("metrics.service: error hotspots error: %v", err)
		return nil, fmt.Errorf("error hotspots: %w", err)
	}

	apdex, err := s.getApdexScore(ctx, req)
	if err != nil {
		log.Printf("metrics.service: apdex score error: %v", err)
		return nil, fmt.Errorf("apdex score: %w", err)
	}

	throughput, timeSeries, err := s.getThroughput(ctx, req)
	if err != nil {
		log.Printf("metrics.service: throughput error: %v", err)
		return nil, fmt.Errorf("throughput: %w", err)
	}

	errorRate, err := s.getErrorRate(ctx, req)
	if err != nil {
		log.Printf("metrics.service: error rate error: %v", err)
		return nil, fmt.Errorf("error rate: %w", err)
	}

	return &DashboardResponse{
		Hotspots: HotspotsData{
			LatencyDistribution: latencyDist,
			SlowestEndpoints:    slowest,
			ErrorHotspots:       errorHotspots,
		},
		Satisfaction: SatisfactionData{
			Apdex:      apdex,
			ErrorRate:  errorRate,
			Throughput: throughput,
			TimeSeries: timeSeries,
		},
	}, nil
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

	// Calculate requests per minute
	durationMinutes := req.To.Sub(req.From).Minutes()
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
