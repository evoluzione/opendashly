import { apiRequest } from './api';

export type RelatedTelemetry = {
  logs: any[];
  metrics: any[];
};

export function fetchRelated(traceId: string): Promise<RelatedTelemetry> {
  return apiRequest<RelatedTelemetry>(`/api/traces/${traceId}/related`);
}
