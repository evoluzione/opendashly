import { apiRequest } from './api';

export type RelatedTelemetry = {
  logs: any[];
  metrics: any[];
};

export type TraceSpan = {
  traceId: string;
  spanId: string;
  parentSpanId?: string;
  name: string;
  service?: string;
  source?: string;
  startTime: string;
  endTime: string;
  duration: number;
  status?: string;
  spanKind?: number | string;
};

export function fetchRelated(traceId: string): Promise<RelatedTelemetry> {
  return apiRequest<RelatedTelemetry>(`/api/traces/${traceId}/related`);
}

export function fetchTraceSpans(traceId: string): Promise<TraceSpan[]> {
  return apiRequest<TraceSpan[]>(`/api/traces/${traceId}/spans`);
}
