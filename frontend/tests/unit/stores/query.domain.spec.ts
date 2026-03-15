import { describe, expect, it, vi } from 'vitest';
import {
  buildAutoRefreshRequest,
  buildRollingRangeRequest,
  getMaxTimestamp,
  mergeResult,
  mergeSingleSignalResult
} from '../../../src/lib/stores/query.domain';
import type { QueryRequest, QueryRunResult } from '../../../src/services/query';

function makeRequest(partial: Partial<QueryRequest> = {}): QueryRequest {
  return {
    signals: ['logs', 'traces', 'metrics'],
    timeRange: {
      from: '2026-03-15T10:00:00.000Z',
      to: '2026-03-15T10:05:00.000Z'
    },
    filters: {},
    ...partial
  };
}

function makeResult(partial: Partial<QueryRunResult> = {}): QueryRunResult {
  return {
    runId: 'run-1',
    status: 'done',
    summary: {
      logCount: 0,
      traceCount: 0,
      metricCount: 0
    },
    results: {
      logs: [],
      traces: [],
      metrics: []
    },
    ...partial
  };
}

describe('query.domain', () => {
  it('mergeResult deduplica logs e traces', () => {
    const previous = makeResult({
      results: {
        logs: [{ timestamp: '1', traceId: 'a', spanId: 's1', body: 'x' }],
        traces: [{ traceId: 't1', timestamp: '1' }],
        metrics: []
      }
    });

    const latest = makeResult({
      results: {
        logs: [
          { timestamp: '1', traceId: 'a', spanId: 's1', body: 'x' },
          { timestamp: '2', traceId: 'b', spanId: 's2', body: 'y' }
        ],
        traces: [
          { traceId: 't1', timestamp: '1' },
          { traceId: 't2', timestamp: '2' }
        ],
        metrics: []
      }
    });

    const merged = mergeResult(previous, latest, makeRequest());

    expect(merged.results.logs).toHaveLength(2);
    expect(merged.results.traces).toHaveLength(2);
  });

  it('mergeSingleSignalResult aggiorna solo il segnale richiesto', () => {
    const previous = makeResult({
      summary: { logCount: 1, traceCount: 1, metricCount: 1 },
      results: {
        logs: [{ timestamp: '1', traceId: 'a', spanId: 's1', body: 'x' }],
        traces: [{ traceId: 't1', timestamp: '1' }],
        metrics: [{ k: 1 }]
      }
    });

    const latest = makeResult({
      summary: { logCount: 2, traceCount: 0, metricCount: 0 },
      results: {
        logs: [{ timestamp: '2', traceId: 'b', spanId: 's2', body: 'y' }],
        traces: [],
        metrics: []
      }
    });

    const merged = mergeSingleSignalResult(previous, latest, 'logs', makeRequest({ signals: ['logs'] }));

    expect(merged.summary.logCount).toBe(2);
    expect(merged.results.logs).toHaveLength(2);
    expect(merged.results.traces).toHaveLength(1);
    expect(merged.results.metrics).toHaveLength(1);
  });

  it('buildRollingRangeRequest genera un intervallo temporale coerente', () => {
    const request = buildRollingRangeRequest(makeRequest(), 5);
    expect(new Date(request.timeRange.to).getTime()).toBeGreaterThan(
      new Date(request.timeRange.from).getTime()
    );
  });

  it('getMaxTimestamp ritorna massimo tra logs e traces', () => {
    const result = makeResult({
      results: {
        logs: [{ timestamp: '2026-03-15T10:01:00.000Z' }],
        traces: [{ timestamp: '2026-03-15T10:03:00.000Z' }],
        metrics: []
      }
    });

    expect(getMaxTimestamp(result)).toBe('2026-03-15T10:03:00.000Z');
  });

  it('buildAutoRefreshRequest usa il delta incrementale in live mode', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-03-15T10:10:00.000Z'));

    const request = buildAutoRefreshRequest({
      lastRequest: makeRequest(),
      autoRefreshSeconds: 1,
      autoRefreshRangeMinutes: 5,
      result: makeResult({
        results: {
          logs: [{ timestamp: '2026-03-15T10:09:00.000Z' }],
          traces: [],
          metrics: []
        }
      })
    });

    expect(request?.timeRange.from).toBe('2026-03-15T10:09:00.001Z');
    expect(request?.timeRange.to).toBe('2026-03-15T10:10:00.000Z');

    vi.useRealTimers();
  });
});
