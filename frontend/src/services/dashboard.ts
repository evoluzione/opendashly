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
}

export interface SatisfactionData {
  apdex: ApdexScore;
  errorRate: number;
  throughput: ThroughputSummary;
  timeSeries: ThroughputPoint[];
}

export interface DashboardResponse {
  hotspots: HotspotsData;
  satisfaction: SatisfactionData;
}

export interface DashboardRequest {
  from: string;
  to: string;
  serviceName?: string;
}

export async function fetchDashboardMetrics(request: DashboardRequest): Promise<DashboardResponse> {
  return apiRequest<DashboardResponse>('/api/dashboard/metrics', {
    method: 'POST',
    body: JSON.stringify(request)
  });
}
