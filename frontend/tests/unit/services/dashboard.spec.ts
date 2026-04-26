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
      },
      health: {
        status: 'ok',
        source: 'rollup'
      },
      warnings: []
    };

    vi.mocked(apiRequest).mockResolvedValue(payload);

    const result = await fetchDashboardMetrics(request);

    expect(result).toEqual(payload);
    expect(apiRequest).toHaveBeenCalledWith('/api/dashboard/metrics', {
      method: 'POST',
      body: JSON.stringify(request)
    });
  });

  it('normalizza il payload quando il backend ritorna array null', async () => {
    vi.mocked(apiRequest).mockResolvedValue({
      hotspots: {
        latencyDistribution: null,
        slowestEndpoints: null,
        errorHotspots: null,
        topEndpoints: null,
        statusCodes: null
      },
      satisfaction: {
        apdex: null,
        errorRate: null,
        throughput: null,
        timeSeries: null,
        latencySeries: null,
        errorRateSeries: null
      },
      logs: {
        volumeSeries: null,
        levels: null
      },
      health: null,
      warnings: null
    } as any);

    const result = await fetchDashboardMetrics({});

    expect(result.hotspots.latencyDistribution).toEqual([]);
    expect(result.hotspots.slowestEndpoints).toEqual([]);
    expect(result.hotspots.errorHotspots).toEqual([]);
    expect(result.hotspots.topEndpoints).toEqual([]);
    expect(result.hotspots.statusCodes).toEqual([]);
    expect(result.satisfaction.timeSeries).toEqual([]);
    expect(result.satisfaction.latencySeries).toEqual([]);
    expect(result.satisfaction.errorRateSeries).toEqual([]);
    expect(result.logs.volumeSeries).toEqual([]);
    expect(result.logs.levels).toEqual([]);
    expect(result.health).toEqual({ status: 'ok', source: 'rollup', reason: undefined });
    expect(result.warnings).toEqual([]);
  });

  it('compatta warning tecnici di pressione ClickHouse', async () => {
    vi.mocked(apiRequest).mockResolvedValue({
      hotspots: {},
      satisfaction: {},
      logs: {},
      health: {
        status: 'degraded',
        source: 'empty',
        reason: 'backend_pressure'
      },
      warnings: [
        'latency distribution unavailable: iterate: code: 241, message: memory limit exceeded: OvercommitTracker'
      ]
    } as any);

    const result = await fetchDashboardMetrics({});

    expect(result.warnings).toEqual([
      'Metriche temporaneamente non disponibili: backend sotto pressione.'
    ]);
  });
});
