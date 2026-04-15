package metrics

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"opendashly/backend/internal/infrastructure/storage"
)

func TestGetDashboard_UsesStaleSectionFallbackOnRecoverableErrors(t *testing.T) {
	recoverableErr := errors.New("memory limit exceeded")
	req := DashboardRequest{
		From:        time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC),
		To:          time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC),
		ServiceName: "api",
	}

	stale := &DashboardResponse{
		Hotspots: HotspotsData{
			SlowestEndpoints: []EndpointLatency{{Endpoint: "GET /stale", Service: "api", P95: 999}},
		},
		Satisfaction: SatisfactionData{
			Apdex:     ApdexScore{Threshold: ApdexThresholdMs},
			ErrorRate: 42.5,
		},
		Logs: LogsData{},
	}

	fetchers := testSuccessFetchers()
	fetchers.slowestEndpoints = func(context.Context, DashboardRequest) ([]EndpointLatency, error) {
		return nil, recoverableErr
	}
	fetchers.errorRate = func(context.Context, DashboardRequest) (float64, error) {
		return 0, recoverableErr
	}

	svc := &Service{
		Storage:          &storage.Client{},
		FreshCacheTTL:    5 * time.Millisecond,
		StaleCacheTTL:    time.Minute,
		QueryParallelism: 4,
		fetchers:         &fetchers,
	}
	svc.getCache().set(req, stale)
	time.Sleep(12 * time.Millisecond)

	result, err := svc.GetDashboard(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(result.Hotspots.SlowestEndpoints); got != 1 {
		t.Fatalf("expected stale slowest endpoints fallback, got len=%d", got)
	}
	if result.Hotspots.SlowestEndpoints[0].Endpoint != "GET /stale" {
		t.Fatalf("unexpected stale fallback endpoint: %#v", result.Hotspots.SlowestEndpoints[0])
	}
	if result.Satisfaction.ErrorRate != 42.5 {
		t.Fatalf("expected stale error rate fallback, got %v", result.Satisfaction.ErrorRate)
	}
	if len(result.Warnings) < 2 {
		t.Fatalf("expected at least 2 warnings, got %v", result.Warnings)
	}

	assertDashboardSlicesInitialized(t, result)
}

func TestGetDashboard_AllRecoverableFailuresReturnStaleSnapshot(t *testing.T) {
	recoverableErr := errors.New("timeout")
	req := DashboardRequest{
		From: time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC),
	}

	stale := &DashboardResponse{
		Hotspots: HotspotsData{
			LatencyDistribution: []LatencyBucket{{Label: "0-100ms", Count: 10}},
			TopEndpoints:        []EndpointThroughput{{Endpoint: "GET /health", Service: "api", RequestCount: 12}},
		},
		Satisfaction: SatisfactionData{
			Apdex:      ApdexScore{Threshold: ApdexThresholdMs, Score: 0.9},
			ErrorRate:  1.2,
			TimeSeries: []ThroughputPoint{{Timestamp: time.Date(2026, 4, 15, 10, 30, 0, 0, time.UTC), RequestCount: 100}},
		},
		Logs: LogsData{
			VolumeSeries: []LogVolumePoint{{Timestamp: time.Date(2026, 4, 15, 10, 30, 0, 0, time.UTC), Count: 55}},
		},
	}

	fetchers := dashboardFetchers{
		latencyDistribution: func(context.Context, DashboardRequest) ([]LatencyBucket, error) { return nil, recoverableErr },
		slowestEndpoints:    func(context.Context, DashboardRequest) ([]EndpointLatency, error) { return nil, recoverableErr },
		errorHotspots:       func(context.Context, DashboardRequest) ([]ErrorHotspot, error) { return nil, recoverableErr },
		latencyPercentiles:  func(context.Context, DashboardRequest) ([]LatencyPercentilePoint, error) { return nil, recoverableErr },
		errorRateSeries:     func(context.Context, DashboardRequest) ([]ErrorRatePoint, error) { return nil, recoverableErr },
		statusCodeBreakdown: func(context.Context, DashboardRequest) ([]StatusCodeBreakdown, error) {
			return nil, recoverableErr
		},
		topEndpoints: func(context.Context, DashboardRequest) ([]EndpointThroughput, error) { return nil, recoverableErr },
		apdexScore:   func(context.Context, DashboardRequest) (ApdexScore, error) { return ApdexScore{}, recoverableErr },
		throughput: func(context.Context, DashboardRequest) (ThroughputSummary, []ThroughputPoint, error) {
			return ThroughputSummary{}, nil, recoverableErr
		},
		logVolume: func(context.Context, DashboardRequest) ([]LogVolumePoint, error) { return nil, recoverableErr },
		logLevels: func(context.Context, DashboardRequest) ([]LogLevelCount, error) { return nil, recoverableErr },
		errorRate: func(context.Context, DashboardRequest) (float64, error) { return 0, recoverableErr },
	}

	svc := &Service{
		Storage:          &storage.Client{},
		FreshCacheTTL:    5 * time.Millisecond,
		StaleCacheTTL:    time.Minute,
		QueryParallelism: 4,
		fetchers:         &fetchers,
	}
	svc.getCache().set(req, stale)
	time.Sleep(12 * time.Millisecond)

	result, err := svc.GetDashboard(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Hotspots.LatencyDistribution) != 1 || result.Hotspots.LatencyDistribution[0].Count != 10 {
		t.Fatalf("expected stale latency distribution, got %#v", result.Hotspots.LatencyDistribution)
	}
	if result.Satisfaction.Apdex.Score != 0.9 {
		t.Fatalf("expected stale apdex score, got %v", result.Satisfaction.Apdex.Score)
	}

	containsStaleWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "stale dashboard snapshot") {
			containsStaleWarning = true
			break
		}
	}
	if !containsStaleWarning {
		t.Fatalf("expected stale snapshot warning, got %v", result.Warnings)
	}
}

func TestGetDashboard_NonRecoverableErrorStillFails(t *testing.T) {
	nonRecoverableErr := errors.New("syntax error")
	fetchers := testSuccessFetchers()
	fetchers.slowestEndpoints = func(context.Context, DashboardRequest) ([]EndpointLatency, error) {
		return nil, nonRecoverableErr
	}

	svc := &Service{
		Storage:          &storage.Client{},
		QueryParallelism: 2,
		fetchers:         &fetchers,
	}

	_, err := svc.GetDashboard(context.Background(), DashboardRequest{
		From: time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected non recoverable error")
	}
	if !strings.Contains(err.Error(), "slowest endpoints") {
		t.Fatalf("expected non recoverable slowest endpoints error, got %v", err)
	}
}

func assertDashboardSlicesInitialized(t *testing.T, result *DashboardResponse) {
	t.Helper()
	if result.Hotspots.LatencyDistribution == nil {
		t.Fatal("latencyDistribution must be non-nil")
	}
	if result.Hotspots.SlowestEndpoints == nil {
		t.Fatal("slowestEndpoints must be non-nil")
	}
	if result.Hotspots.ErrorHotspots == nil {
		t.Fatal("errorHotspots must be non-nil")
	}
	if result.Hotspots.TopEndpoints == nil {
		t.Fatal("topEndpoints must be non-nil")
	}
	if result.Hotspots.StatusCodes == nil {
		t.Fatal("statusCodes must be non-nil")
	}
	if result.Satisfaction.TimeSeries == nil {
		t.Fatal("timeSeries must be non-nil")
	}
	if result.Satisfaction.LatencySeries == nil {
		t.Fatal("latencySeries must be non-nil")
	}
	if result.Satisfaction.ErrorRateSeries == nil {
		t.Fatal("errorRateSeries must be non-nil")
	}
	if result.Logs.VolumeSeries == nil {
		t.Fatal("logs.volumeSeries must be non-nil")
	}
	if result.Logs.Levels == nil {
		t.Fatal("logs.levels must be non-nil")
	}
}

func testSuccessFetchers() dashboardFetchers {
	return dashboardFetchers{
		latencyDistribution: func(context.Context, DashboardRequest) ([]LatencyBucket, error) {
			return []LatencyBucket{}, nil
		},
		slowestEndpoints: func(context.Context, DashboardRequest) ([]EndpointLatency, error) {
			return []EndpointLatency{}, nil
		},
		errorHotspots: func(context.Context, DashboardRequest) ([]ErrorHotspot, error) {
			return []ErrorHotspot{}, nil
		},
		latencyPercentiles: func(context.Context, DashboardRequest) ([]LatencyPercentilePoint, error) {
			return []LatencyPercentilePoint{}, nil
		},
		errorRateSeries: func(context.Context, DashboardRequest) ([]ErrorRatePoint, error) {
			return []ErrorRatePoint{}, nil
		},
		statusCodeBreakdown: func(context.Context, DashboardRequest) ([]StatusCodeBreakdown, error) {
			return []StatusCodeBreakdown{}, nil
		},
		topEndpoints: func(context.Context, DashboardRequest) ([]EndpointThroughput, error) {
			return []EndpointThroughput{}, nil
		},
		apdexScore: func(context.Context, DashboardRequest) (ApdexScore, error) {
			return ApdexScore{Threshold: ApdexThresholdMs}, nil
		},
		throughput: func(context.Context, DashboardRequest) (ThroughputSummary, []ThroughputPoint, error) {
			return ThroughputSummary{}, []ThroughputPoint{}, nil
		},
		logVolume: func(context.Context, DashboardRequest) ([]LogVolumePoint, error) {
			return []LogVolumePoint{}, nil
		},
		logLevels: func(context.Context, DashboardRequest) ([]LogLevelCount, error) {
			return []LogLevelCount{}, nil
		},
		errorRate: func(context.Context, DashboardRequest) (float64, error) {
			return 0, nil
		},
	}
}
