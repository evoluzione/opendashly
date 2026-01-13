import { apiRequest } from './api';

export type QueryRequest = {
  signals: string[];
  timeRange: { from: string; to: string };
  filters: Record<string, string>;
  limit?: number;
  orderBy?: string;
};

export type QueryRunResult = {
  runId: string;
  status: string;
  summary: { logCount: number; traceCount: number; metricCount: number };
  results: { logs: any[]; traces: any[]; metrics: any[] };
};

export function runQuery(request: QueryRequest): Promise<QueryRunResult> {
  return apiRequest<QueryRunResult>('/api/query/run', {
    method: 'POST',
    body: JSON.stringify(request)
  });
}
