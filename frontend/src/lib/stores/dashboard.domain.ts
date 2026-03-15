import type { DashboardRequest } from '../../services/dashboard';

export function buildRollingDashboardRequest(lastRequest: DashboardRequest): DashboardRequest {
  if (!lastRequest.from || !lastRequest.to) {
    return { ...lastRequest };
  }

  const now = new Date();
  const originalDuration =
    new Date(lastRequest.to).getTime() - new Date(lastRequest.from).getTime();

  if (!Number.isFinite(originalDuration) || originalDuration <= 0) {
    return { ...lastRequest };
  }

  const from = new Date(now.getTime() - originalDuration);
  return {
    ...lastRequest,
    from: from.toISOString(),
    to: now.toISOString()
  };
}
