import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { dashboardState, loadDashboard, resetDashboard } from '../../../src/lib/stores/dashboard';
import { fetchDashboardMetrics } from '../../../src/services/dashboard';

vi.mock('../../../src/services/dashboard', () => ({
  fetchDashboardMetrics: vi.fn()
}));

describe('dashboard store', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetDashboard();
  });

  afterEach(() => {
    resetDashboard();
  });

  it('mantiene i dati precedenti quando un refresh fallisce', async () => {
    vi.mocked(fetchDashboardMetrics).mockResolvedValueOnce({
      hotspots: {
        latencyDistribution: [],
        slowestEndpoints: [],
        errorHotspots: [],
        topEndpoints: [],
        statusCodes: []
      },
      satisfaction: {
        apdex: {
          score: 0.99,
          satisfied: 99,
          tolerating: 1,
          frustrated: 0,
          total: 100,
          threshold: 2000
        },
        errorRate: 0,
        throughput: {
          totalRequests: 100,
          totalErrors: 0,
          requestsPerMin: 10,
          errorsPerMin: 0
        },
        timeSeries: [],
        latencySeries: [],
        errorRateSeries: []
      },
      logs: {
        volumeSeries: [],
        levels: []
      },
      warnings: []
    });

    await loadDashboard({});

    vi.mocked(fetchDashboardMetrics).mockRejectedValueOnce(new Error('Gateway timeout'));
    await loadDashboard({});

    const state = get(dashboardState);
    expect(state.data).not.toBeNull();
    expect(state.error).toBe('Gateway timeout');
    expect(state.warnings).toContain('refresh failed: Gateway timeout');
  });
});
