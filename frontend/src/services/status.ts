import { apiRequest } from './api';

export type TelemetryCounts = {
  total: number;
  last5m: number;
  last10m: number;
  last60m: number;
  series?: TelemetryPoint[];
};

export type TelemetryPoint = {
  ts: string;
  count: number;
};

export type StatusSummary = {
  ok: boolean;
  generatedAt: string;
  error?: string;
  checks: { database: boolean };
  counts: {
    logs: TelemetryCounts;
    traces: TelemetryCounts;
  };
};

export type RuntimeComponentHealth = {
  name: string;
  status: 'up' | 'down' | 'degraded' | string;
  latencyMs?: number;
  error?: string;
};

export type RuntimeResourceHealth = {
  cpuCoresAvailable: number;
  cpuUsedPercentOneCore: number;
  memoryLimitBytes: number;
  memoryUsedBytes: number;
  memoryUsedPercent: number;
  goHeapAllocBytes: number;
  goRoutines: number;
  backendUptimeSeconds: number;
};

export type RuntimeQueryHealth = {
  runningNow: number;
  slowRunningNow: number;
  maxRunningElapsedSec: number;
  slowQueriesLast15m: number;
  failedQueriesLast15m: number;
  slowThresholdSec: number;
};

export type RuntimeSummary = {
  ok: boolean;
  generatedAt: string;
  components: RuntimeComponentHealth[];
  resources: RuntimeResourceHealth;
  queries: RuntimeQueryHealth;
  warnings?: string[];
};

export function fetchStatusSummary(): Promise<StatusSummary> {
  return apiRequest<StatusSummary>('/api/status/summary');
}

export function fetchRuntimeSummary(): Promise<RuntimeSummary> {
  return apiRequest<RuntimeSummary>('/api/status/runtime');
}
