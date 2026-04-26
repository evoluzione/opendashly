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
		if strings.Contains(warning, "servite da cache") {
			containsStaleWarning = true
			break
		}
	}
	if !containsStaleWarning {
		t.Fatalf("expected stale snapshot warning, got %v", result.Warnings)
	}
}

func TestGetDashboard_NonRecoverableErrorDegradesToWarning(t *testing.T) {
	// Resilience contract: any fetcher failure — recoverable or not — must
	// degrade the affected widget to a warning instead of bubbling a 5xx.
	// A single misbehaving query can never take the whole dashboard down.
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

	result, err := svc.GetDashboard(context.Background(), DashboardRequest{
		From: time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("dashboard must not bubble errors, got %v", err)
	}
	if result == nil {
		t.Fatal("expected dashboard response, got nil")
	}

	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "slowest endpoints") && strings.Contains(w, "non disponibile") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected slowest-endpoints warning, got %v", result.Warnings)
	}
	assertDashboardSlicesInitialized(t, result)
}

func TestGetDashboard_HalveOnOOMRetriesAndSucceeds(t *testing.T) {
	// When a fetcher fails with a recoverable non-pressure error, HalveOnOOM
	// retries once over the last half of the window and returns a partial
	// result instead of surfacing an error.
	recoverableErr := errors.New("timeout")
	req := DashboardRequest{
		From: time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC),
	}
	midpoint := req.To.Add(-req.To.Sub(req.From) / 2)

	calls := 0
	fetchers := testSuccessFetchers()
	fetchers.slowestEndpoints = func(_ context.Context, r DashboardRequest) ([]EndpointLatency, error) {
		calls++
		if calls == 1 {
			return nil, recoverableErr
		}
		if !r.From.Equal(midpoint) {
			t.Fatalf("expected retry to use halved window from=%s, got %s", midpoint, r.From)
		}
		return []EndpointLatency{{Endpoint: "GET /half", Service: "api", P95: 123}}, nil
	}

	svc := &Service{
		Storage:          &storage.Client{},
		QueryParallelism: 4,
		HalveOnOOM:       true,
		fetchers:         &fetchers,
	}

	result, err := svc.GetDashboard(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected exactly 2 fetcher calls (original + halved retry), got %d", calls)
	}
	if len(result.Hotspots.SlowestEndpoints) != 1 || result.Hotspots.SlowestEndpoints[0].Endpoint != "GET /half" {
		t.Fatalf("expected slowest endpoints from halved retry, got %#v", result.Hotspots.SlowestEndpoints)
	}
	partialHintFound := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "slowest endpoints") && strings.Contains(w, "partial window") {
			partialHintFound = true
			break
		}
	}
	if !partialHintFound {
		t.Fatalf("expected partial-window hint in warnings, got %v", result.Warnings)
	}
	assertDashboardSlicesInitialized(t, result)
}

func TestGetDashboard_PressureErrorOpensCircuitBreakerWithoutRetry(t *testing.T) {
	pressureErr := errors.New("memory limit exceeded: OvercommitTracker")
	req := DashboardRequest{
		From: time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC),
	}

	calls := 0
	fetchers := testSuccessFetchers()
	fetchers.slowestEndpoints = func(context.Context, DashboardRequest) ([]EndpointLatency, error) {
		calls++
		return nil, pressureErr
	}

	svc := &Service{
		Storage:          &storage.Client{},
		QueryParallelism: 1,
		HalveOnOOM:       true,
		PressureCooldown: time.Minute,
		fetchers:         &fetchers,
	}

	result, err := svc.GetDashboard(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("pressure errors must not be retried, got %d calls", calls)
	}
	if result.Health.Status != "partial" && result.Health.Status != "degraded" {
		t.Fatalf("expected partial/degraded health, got %#v", result.Health)
	}

	_, active := svc.dashboardPressureActive()
	if !active {
		t.Fatal("expected pressure circuit breaker to be active")
	}
}

func TestGetDashboard_UsesLastGoodForRollingWindowUnderPressure(t *testing.T) {
	req1 := DashboardRequest{
		From: time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC),
	}
	req2 := DashboardRequest{
		From: time.Date(2026, 4, 15, 10, 1, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 15, 11, 1, 0, 0, time.UTC),
	}

	stale := &DashboardResponse{
		Hotspots: HotspotsData{
			SlowestEndpoints: []EndpointLatency{{Endpoint: "GET /last-good", Service: "api", P95: 321}},
		},
		Satisfaction: SatisfactionData{
			Apdex: ApdexScore{Threshold: ApdexThresholdMs, Score: 0.95},
		},
		Logs:   LogsData{},
		Health: DashboardHealth{Status: "ok", Source: "rollup"},
	}

	fetchers := dashboardFetchers{
		latencyDistribution: func(context.Context, DashboardRequest) ([]LatencyBucket, error) {
			return nil, errors.New("timeout")
		},
		slowestEndpoints: func(context.Context, DashboardRequest) ([]EndpointLatency, error) { return nil, errors.New("timeout") },
		errorHotspots:    func(context.Context, DashboardRequest) ([]ErrorHotspot, error) { return nil, errors.New("timeout") },
		latencyPercentiles: func(context.Context, DashboardRequest) ([]LatencyPercentilePoint, error) {
			return nil, errors.New("timeout")
		},
		errorRateSeries: func(context.Context, DashboardRequest) ([]ErrorRatePoint, error) { return nil, errors.New("timeout") },
		statusCodeBreakdown: func(context.Context, DashboardRequest) ([]StatusCodeBreakdown, error) {
			return nil, errors.New("timeout")
		},
		topEndpoints: func(context.Context, DashboardRequest) ([]EndpointThroughput, error) {
			return nil, errors.New("timeout")
		},
		apdexScore: func(context.Context, DashboardRequest) (ApdexScore, error) {
			return ApdexScore{}, errors.New("timeout")
		},
		throughput: func(context.Context, DashboardRequest) (ThroughputSummary, []ThroughputPoint, error) {
			return ThroughputSummary{}, nil, errors.New("timeout")
		},
		logVolume: func(context.Context, DashboardRequest) ([]LogVolumePoint, error) { return nil, errors.New("timeout") },
		logLevels: func(context.Context, DashboardRequest) ([]LogLevelCount, error) { return nil, errors.New("timeout") },
		errorRate: func(context.Context, DashboardRequest) (float64, error) { return 0, errors.New("timeout") },
	}

	svc := &Service{
		Storage:          &storage.Client{},
		FreshCacheTTL:    5 * time.Millisecond,
		StaleCacheTTL:    20 * time.Millisecond,
		LastGoodCacheTTL: time.Minute,
		QueryParallelism: 1,
		fetchers:         &fetchers,
	}
	svc.getCache().set(req1, stale)
	time.Sleep(30 * time.Millisecond)

	result, err := svc.GetDashboard(context.Background(), req2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Hotspots.SlowestEndpoints) != 1 || result.Hotspots.SlowestEndpoints[0].Endpoint != "GET /last-good" {
		t.Fatalf("expected last-good fallback, got %#v", result.Hotspots.SlowestEndpoints)
	}
	if result.Health.Source != "stale_cache" {
		t.Fatalf("expected stale_cache source, got %#v", result.Health)
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
