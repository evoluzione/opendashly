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
  logsCursor?: string;
  tracesCursor?: string;
  page?: number;
  limit?: number;
  orderBy?: string;
};

export type Pagination = {
  page: number;
  limit: number;
  total: number;
  totalPages: number;
  hasNext: boolean;
  nextCursor?: string;
};

export type QueryRunResult = {
  runId: string;
  status: string;
  summary: { logCount: number; traceCount: number; metricCount: number };
  pagination?: { logs: Pagination; traces: Pagination; metrics: Pagination };
  results: { logs: any[]; traces: any[]; metrics: any[] };
  signalErrors?: Record<string, string>;
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
  return apiRequest<unknown>(`/api/query/attributes?${params.toString()}`).then((payload) => {
    if (!Array.isArray(payload)) return [];
    return payload.filter((item): item is string => typeof item === 'string');
  });
}
