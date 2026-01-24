import { apiRequest } from './api';

export type QueryRequest = {
  signals: string[];
  timeRange: { from: string; to: string };
  filters: Record<string, string>;
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
  console.debug('query.run.request', {
    signals: request.signals,
    timeRange: request.timeRange,
    filters: Object.keys(request.filters ?? {}).length,
    page: request.page,
    limit: request.limit,
    orderBy: request.orderBy
  });
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
