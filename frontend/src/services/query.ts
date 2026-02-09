import { apiRequest } from './api';

export interface FilterItem {
  connector: 'AND' | 'OR';
  key: string;
  operator: string;
  value: string;
}

export type QueryRequest = {
  signals: string[];
  timeRange: { from: string; to: string };
  filters: Record<string, string>;
  filterList?: FilterItem[];
  page?: number;
  limit?: number;
  orderBy?: string;
};

export type Pagination = {
  page: number;
  limit: number;
  total: number;
  totalPages: number;
};

export type QueryRunResult = {
  runId: string;
  status: string;
  summary: { logCount: number; traceCount: number; metricCount: number };
  pagination?: { logs: Pagination; traces: Pagination; metrics: Pagination };
  results: { logs: any[]; traces: any[]; metrics: any[] };
};

export type SmartQueryResponse = {
  sql: string;
  request: QueryRequest;
  warnings?: string[];
};

export function runQuery(request: QueryRequest): Promise<QueryRunResult> {
  return apiRequest<QueryRunResult>('/api/query/run', {
    method: 'POST',
    body: JSON.stringify(request)
  });
}

export function generateSmartQuery(payload: { prompt: string; contextType: 'logs' | 'metrics' | 'traces' | 'auto' }): Promise<SmartQueryResponse> {
  return apiRequest<SmartQueryResponse>('/api/query/smart', {
    method: 'POST',
    body: JSON.stringify(payload)
  });
}

export function getLogAttributes(search: string): Promise<string[]> {
  const params = new URLSearchParams({ q: search });
  return apiRequest<string[]>(`/api/query/attributes?${params.toString()}`);
}
