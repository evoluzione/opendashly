import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import { executeQuery, queryState, stopAutoRefresh } from '../../../src/lib/stores/query';
import { runQuery } from '../../../src/services/query';

vi.mock('../../../src/services/query', () => ({
  runQuery: vi.fn()
}));

vi.mock('../../../src/services/services', () => ({
  fetchServices: vi.fn()
}));

const baseRequest = {
  signals: ['logs', 'traces', 'metrics'],
  timeRange: {
    from: '2026-04-15T10:00:00.000Z',
    to: '2026-04-15T11:00:00.000Z'
  },
  filters: {}
};

const baseResult = {
  runId: 'run-1',
  status: 'complete',
  summary: {
    logCount: 1,
    traceCount: 0,
    metricCount: 0
  },
  results: {
    logs: [{ body: 'ok', timestamp: '2026-04-15T10:10:00.000Z' }],
    traces: [],
    metrics: []
  },
  signalErrors: {}
};

describe('query store', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    stopAutoRefresh();
    queryState.set({
      loading: false,
      error: null,
      warnings: [],
      result: null,
      lastRequest: null,
      autoRefreshSeconds: null,
      autoRefreshRangeMinutes: null,
      isLiveUpdate: false
    });
  });

  afterEach(() => {
    stopAutoRefresh();
  });

  it('mantiene i risultati esistenti su errore transient durante refresh', async () => {
    vi.mocked(runQuery).mockResolvedValueOnce(baseResult as any);
    await executeQuery(baseRequest as any);

    vi.mocked(runQuery).mockRejectedValueOnce(new Error('Request timed out after 10000ms'));
    await executeQuery(baseRequest as any, { retainResult: true, isBackground: true });

    const state = get(queryState);
    expect(state.result).not.toBeNull();
    expect(state.result?.results.logs).toHaveLength(1);
    expect(state.error).toBe('Request timed out after 10000ms');
    expect(state.warnings).toContain('refresh failed: Request timed out after 10000ms');
  });
});
