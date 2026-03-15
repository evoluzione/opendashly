import type { DashboardRequest } from './dashboard';

export function buildDashboardMetricsPayload(request: DashboardRequest): DashboardRequest {
  return {
    from: request.from,
    to: request.to,
    serviceName: request.serviceName
  };
}
