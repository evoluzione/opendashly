import { apiRequest } from './api';

export interface LatencyBucket {
  rangeStart: number;
  rangeEnd: number;
  count: number;
  percentage: number;
  label: string;
}

export interface EndpointLatency {
  endpoint: string;
  service: string;
  avgMs: number;
  p50: number;
  p95: number;
  p99: number;
  count: number;
}

export interface ErrorHotspot {
  endpoint: string;
  service: string;
  errorCount: number;
  totalCount: number;
  errorRate: number;
}

export interface ApdexScore {
  score: number;
  satisfied: number;
  tolerating: number;
  frustrated: number;
  total: number;
  threshold: number;
}

export interface ThroughputPoint {
  timestamp: string;
  requestCount: number;
  errorCount: number;
}

export interface LatencyPercentilePoint {
  timestamp: string;
  p50: number;
  p95: number;
  p99: number;
}

export interface ErrorRatePoint {
  timestamp: string;
  errorRate: number;
  errorCount: number;
  totalCount: number;
}

export interface StatusCodeBreakdown {
  code: string;
  count: number;
  percentage: number;
}

export interface EndpointThroughput {
  endpoint: string;
  service: string;
  requestCount: number;
  errorCount: number;
  errorRate: number;
}

export interface LogVolumePoint {
  timestamp: string;
  count: number;
}

export interface LogLevelCount {
  level: string;
  count: number;
  percentage: number;
}

export interface ThroughputSummary {
  totalRequests: number;
  totalErrors: number;
  requestsPerMin: number;
  errorsPerMin: number;
}

export interface HotspotsData {
  latencyDistribution: LatencyBucket[];
  slowestEndpoints: EndpointLatency[];
  errorHotspots: ErrorHotspot[];
  topEndpoints: EndpointThroughput[];
  statusCodes: StatusCodeBreakdown[];
}

export interface SatisfactionData {
  apdex: ApdexScore;
  errorRate: number;
  throughput: ThroughputSummary;
  timeSeries: ThroughputPoint[];
  latencySeries: LatencyPercentilePoint[];
  errorRateSeries: ErrorRatePoint[];
}

export interface LogsData {
  volumeSeries: LogVolumePoint[];
  levels: LogLevelCount[];
}

export interface DashboardResponse {
  hotspots: HotspotsData;
  satisfaction: SatisfactionData;
  logs: LogsData;
}

export interface DashboardRequest {
  from?: string;
  to?: string;
  serviceName?: string;
}

export async function fetchDashboardMetrics(request: DashboardRequest): Promise<DashboardResponse> {
  return apiRequest<DashboardResponse>('/api/dashboard/metrics', {
    method: 'POST',
    body: JSON.stringify(request)
  });
}
