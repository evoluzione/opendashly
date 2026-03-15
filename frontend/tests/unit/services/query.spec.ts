import { describe, expect, it, vi, beforeEach } from 'vitest';
import { getLogAttributes, runQuery, type QueryRequest } from '../../../src/services/query';
import { apiRequest } from '../../../src/services/api';

vi.mock('../../../src/services/api', () => ({
  apiRequest: vi.fn()
}));

describe('query service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('runQuery invia la richiesta corretta', async () => {
    const request: QueryRequest = {
      signals: ['logs'],
      timeRange: {
        from: '2026-03-15T10:00:00.000Z',
        to: '2026-03-15T10:05:00.000Z'
      },
      filters: { service: 'api' }
    };

    const payload = {
      runId: 'run-1',
      status: 'done',
      summary: { logCount: 1, traceCount: 0, metricCount: 0 },
      results: { logs: [], traces: [], metrics: [] }
    };

    vi.mocked(apiRequest).mockResolvedValue(payload);

    const result = await runQuery(request);

    expect(result).toEqual(payload);
    expect(apiRequest).toHaveBeenCalledWith('/api/query/run', {
      method: 'POST',
      body: JSON.stringify(request)
    });
  });

  it('getLogAttributes filtra valori non stringa', async () => {
    vi.mocked(apiRequest).mockResolvedValue(['service.name', 123, 'body', null] as unknown as never);

    const result = await getLogAttributes('serv');

    expect(result).toEqual(['service.name', 'body']);
    expect(apiRequest).toHaveBeenCalledWith('/api/query/attributes?q=serv');
  });

  it('getLogAttributes ritorna array vuoto su payload non array', async () => {
    vi.mocked(apiRequest).mockResolvedValue({ items: [] } as never);

    const result = await getLogAttributes('x');

    expect(result).toEqual([]);
  });
});
