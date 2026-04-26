package metrics

import "time"

// DashboardRequest contains parameters for dashboard metrics queries.
type DashboardRequest struct {
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
	ServiceName string    `json:"serviceName,omitempty"`
}

// DashboardHealth describes how fresh and complete a dashboard response is.
type DashboardHealth struct {
	Status string `json:"status"`           // ok, partial, degraded
	Source string `json:"source"`           // rollup, stale_cache, empty
	Reason string `json:"reason,omitempty"` // backend_pressure, rollup_warming, partial_failure
}

// LatencyBucket represents a histogram bucket for latency distribution.
type LatencyBucket struct {
	RangeStart int     `json:"rangeStart"`
	RangeEnd   int     `json:"rangeEnd"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
	Label      string  `json:"label"`
}

// EndpointLatency contains latency statistics for an endpoint.
type EndpointLatency struct {
	Endpoint string  `json:"endpoint"`
	Service  string  `json:"service"`
	AvgMs    float64 `json:"avgMs"`
	P50      float64 `json:"p50"`
	P95      float64 `json:"p95"`
	P99      float64 `json:"p99"`
	Count    int64   `json:"count"`
}

// ErrorHotspot contains error statistics for an endpoint.
type ErrorHotspot struct {
	Endpoint   string  `json:"endpoint"`
	Service    string  `json:"service"`
	ErrorCount int64   `json:"errorCount"`
	TotalCount int64   `json:"totalCount"`
	ErrorRate  float64 `json:"errorRate"`
}

// ApdexScore contains APDEX calculation results.
type ApdexScore struct {
	Score      float64 `json:"score"`
	Satisfied  int64   `json:"satisfied"`
	Tolerating int64   `json:"tolerating"`
	Frustrated int64   `json:"frustrated"`
	Total      int64   `json:"total"`
	Threshold  int     `json:"threshold"`
}

// ThroughputPoint represents a single point in throughput time series.
type ThroughputPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	RequestCount int64     `json:"requestCount"`
	ErrorCount   int64     `json:"errorCount"`
}

// LatencyPercentilePoint represents latency percentiles over time.
type LatencyPercentilePoint struct {
	Timestamp time.Time `json:"timestamp"`
	P50       float64   `json:"p50"`
	P95       float64   `json:"p95"`
	P99       float64   `json:"p99"`
}

// ErrorRatePoint represents error rate over time.
type ErrorRatePoint struct {
	Timestamp  time.Time `json:"timestamp"`
	ErrorRate  float64   `json:"errorRate"`
	ErrorCount int64     `json:"errorCount"`
	TotalCount int64     `json:"totalCount"`
}

// StatusCodeBreakdown summarizes status code distribution.
type StatusCodeBreakdown struct {
	Code       string  `json:"code"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// EndpointThroughput contains throughput statistics per endpoint.
type EndpointThroughput struct {
	Endpoint     string  `json:"endpoint"`
	Service      string  `json:"service"`
	RequestCount int64   `json:"requestCount"`
	ErrorCount   int64   `json:"errorCount"`
	ErrorRate    float64 `json:"errorRate"`
}

// LogVolumePoint represents log volume over time.
type LogVolumePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
}

// LogLevelCount represents log level distribution.
type LogLevelCount struct {
	Level      string  `json:"level"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// HotspotsData contains performance hotspot information.
type HotspotsData struct {
	LatencyDistribution []LatencyBucket       `json:"latencyDistribution"`
	SlowestEndpoints    []EndpointLatency     `json:"slowestEndpoints"`
	ErrorHotspots       []ErrorHotspot        `json:"errorHotspots"`
	TopEndpoints        []EndpointThroughput  `json:"topEndpoints"`
	StatusCodes         []StatusCodeBreakdown `json:"statusCodes"`
}

// SatisfactionData contains satisfaction metrics.
type SatisfactionData struct {
	Apdex           ApdexScore               `json:"apdex"`
	ErrorRate       float64                  `json:"errorRate"`
	Throughput      ThroughputSummary        `json:"throughput"`
	TimeSeries      []ThroughputPoint        `json:"timeSeries"`
	LatencySeries   []LatencyPercentilePoint `json:"latencySeries"`
	ErrorRateSeries []ErrorRatePoint         `json:"errorRateSeries"`
}

// LogsData contains log analytics.
type LogsData struct {
	VolumeSeries []LogVolumePoint `json:"volumeSeries"`
	Levels       []LogLevelCount  `json:"levels"`
}

// ThroughputSummary contains summary statistics for throughput.
type ThroughputSummary struct {
	TotalRequests  int64   `json:"totalRequests"`
	TotalErrors    int64   `json:"totalErrors"`
	RequestsPerMin float64 `json:"requestsPerMin"`
	ErrorsPerMin   float64 `json:"errorsPerMin"`
}

// DashboardResponse contains all dashboard metrics.
type DashboardResponse struct {
	Hotspots     HotspotsData     `json:"hotspots"`
	Satisfaction SatisfactionData `json:"satisfaction"`
	Logs         LogsData         `json:"logs"`
	Health       DashboardHealth  `json:"health"`
	Warnings     []string         `json:"warnings,omitempty"`
}
