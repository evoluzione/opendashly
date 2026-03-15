import { describe, expect, it, vi, beforeEach } from 'vitest';
import { fetchDashboardMetrics } from '../../../src/services/dashboard';
import { apiRequest } from '../../../src/services/api';

vi.mock('../../../src/services/api', () => ({
  apiRequest: vi.fn()
}));

describe('dashboard service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetchDashboardMetrics invia la richiesta corretta', async () => {
    const request = {
      from: '2026-03-15T10:00:00.000Z',
      to: '2026-03-15T11:00:00.000Z',
      serviceName: 'gateway'
    };

    const payload = {
      hotspots: {
        latencyDistribution: [],
        slowestEndpoints: [],
        errorHotspots: [],
        topEndpoints: [],
        statusCodes: []
      },
      satisfaction: {
        apdex: {
          score: 0.98,
          satisfied: 98,
          tolerating: 1,
          frustrated: 1,
          total: 100,
          threshold: 0.5
        },
        errorRate: 0.01,
        throughput: {
          totalRequests: 100,
          totalErrors: 1,
          requestsPerMin: 10,
          errorsPerMin: 0.1
        },
        timeSeries: [],
        latencySeries: [],
        errorRateSeries: []
      },
      logs: {
        volumeSeries: [],
        levels: []
      }
    };

    vi.mocked(apiRequest).mockResolvedValue(payload);

    const result = await fetchDashboardMetrics(request);

    expect(result).toEqual(payload);
    expect(apiRequest).toHaveBeenCalledWith('/api/dashboard/metrics', {
      method: 'POST',
      body: JSON.stringify(request)
    });
  });
});
