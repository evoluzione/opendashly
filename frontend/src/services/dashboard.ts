import { apiRequest } from './api';
import { buildDashboardMetricsPayload } from './dashboard.mapper';

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
  health: DashboardHealth;
  warnings?: string[];
}

export interface DashboardHealth {
  status: 'ok' | 'partial' | 'degraded';
  source: 'rollup' | 'stale_cache' | 'empty';
  reason?: 'backend_pressure' | 'rollup_warming' | 'partial_failure' | 'storage_unavailable' | string;
}

export interface DashboardRequest {
  from?: string;
  to?: string;
  serviceName?: string;
}

const DASHBOARD_REQUEST_TIMEOUT_MS = 40000;

export async function fetchDashboardMetrics(request: DashboardRequest): Promise<DashboardResponse> {
  const payload = await apiRequest<DashboardResponse>('/api/dashboard/metrics', {
    method: 'POST',
    body: JSON.stringify(buildDashboardMetricsPayload(request)),
    timeoutMs: DASHBOARD_REQUEST_TIMEOUT_MS
  });
  return normalizeDashboardResponse(payload);
}

function normalizeDashboardResponse(payload: DashboardResponse | null | undefined): DashboardResponse {
  const safeHotspots = payload?.hotspots ?? ({} as HotspotsData);
  const safeSatisfaction = payload?.satisfaction ?? ({} as SatisfactionData);
  const safeLogs = payload?.logs ?? ({} as LogsData);
  const health = normalizeDashboardHealth(payload?.health);

  return {
    hotspots: {
      latencyDistribution: normalizeArray(safeHotspots.latencyDistribution),
      slowestEndpoints: normalizeArray(safeHotspots.slowestEndpoints),
      errorHotspots: normalizeArray(safeHotspots.errorHotspots),
      topEndpoints: normalizeArray(safeHotspots.topEndpoints),
      statusCodes: normalizeArray(safeHotspots.statusCodes)
    },
    satisfaction: {
      apdex: safeSatisfaction.apdex ?? {
        score: 0,
        satisfied: 0,
        tolerating: 0,
        frustrated: 0,
        total: 0,
        threshold: 2000
      },
      errorRate: safeSatisfaction.errorRate ?? 0,
      throughput: safeSatisfaction.throughput ?? {
        totalRequests: 0,
        totalErrors: 0,
        requestsPerMin: 0,
        errorsPerMin: 0
      },
      timeSeries: normalizeArray(safeSatisfaction.timeSeries),
      latencySeries: normalizeArray(safeSatisfaction.latencySeries),
      errorRateSeries: normalizeArray(safeSatisfaction.errorRateSeries)
    },
    logs: {
      volumeSeries: normalizeArray(safeLogs.volumeSeries),
      levels: normalizeArray(safeLogs.levels)
    },
    health,
    warnings: normalizeDashboardWarnings(payload?.warnings, health)
  };
}

function normalizeArray<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : [];
}

function normalizeDashboardHealth(value: DashboardHealth | null | undefined): DashboardHealth {
  const status = value?.status;
  const source = value?.source;
  return {
    status: status === 'partial' || status === 'degraded' ? status : 'ok',
    source: source === 'stale_cache' || source === 'empty' ? source : 'rollup',
    reason: value?.reason
  };
}

function normalizeDashboardWarnings(
  value: string[] | null | undefined,
  health: DashboardHealth
): string[] {
  if (health.source === 'stale_cache' && health.reason === 'backend_pressure') {
    return ['Metriche temporaneamente servite da cache: backend sotto pressione.'];
  }
  if (health.status === 'degraded' && health.reason === 'backend_pressure') {
    return ['Metriche temporaneamente non disponibili: backend sotto pressione.'];
  }

  const warnings = normalizeArray(value)
    .map((warning) => sanitizeDashboardWarning(warning))
    .filter((warning, index, list) => warning && list.indexOf(warning) === index);
  return warnings.slice(0, 3);
}

function sanitizeDashboardWarning(warning: string): string {
  const lower = warning.toLowerCase();
  if (lower.includes('memory limit exceeded') || lower.includes('overcommittracker')) {
    return 'Metriche temporaneamente non disponibili: backend sotto pressione.';
  }
  if (warning.length > 180) {
    return `${warning.slice(0, 177)}...`;
  }
  return warning;
}
