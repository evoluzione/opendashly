import { apiRequest } from './api';

export type TelemetryCounts = {
  total: number;
  last5m: number;
  last10m: number;
  last60m: number;
};

export type StatusSummary = {
  ok: boolean;
  generatedAt: string;
  error?: string;
  checks: { database: boolean };
  counts: {
    logs: TelemetryCounts;
    traces: TelemetryCounts;
    metrics: TelemetryCounts;
  };
};

export function fetchStatusSummary(): Promise<StatusSummary> {
  return apiRequest<StatusSummary>('/api/status/summary');
}
