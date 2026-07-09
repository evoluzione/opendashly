import { describe, expect, it, vi } from 'vitest';
import {
  buildDashboardRequest,
  countActiveDashboardFilters,
  createInitialRunRequest,
  createPageChangeRequest,
  createPageSizeRequest,
  defaultCustomRangeInputs,
  formatDateTimeLocal,
  storeNextCursors,
  toIsoFromLocal
} from '../../../src/routes/home-page.logic';

describe('home-page.logic', () => {
  it('formatDateTimeLocal produce formato datetime-local', () => {
    const value = formatDateTimeLocal(new Date('2026-03-15T08:07:00.000Z'));
    expect(value).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
  });

  it('toIsoFromLocal gestisce input valido e non valido', () => {
    expect(toIsoFromLocal('invalid')).toBeNull();
    expect(toIsoFromLocal('2026-03-15T10:00')).toContain('2026-03-15T');
  });

  it('defaultCustomRangeInputs ritorna un range coerente di 6 ore', () => {
    const now = new Date('2026-03-15T12:00:00.000Z');
    const result = defaultCustomRangeInputs(now);
    expect(result.fromInput).toBeTruthy();
    expect(result.toInput).toBeTruthy();
  });

  it('buildDashboardRequest valida custom range', () => {
    const invalid = buildDashboardRequest({
      selectedService: null,
      rangePreset: 'custom',
      fromInput: '2026-03-15T12:00',
      toInput: '2026-03-15T11:00'
    });

    expect(invalid.error).toContain('successiva');

    const valid = buildDashboardRequest({
      selectedService: 'gateway',
      rangePreset: 'custom',
      fromInput: '2026-03-15T10:00',
      toInput: '2026-03-15T11:00'
    });

    expect(valid.error).toBeNull();
    expect(valid.request.serviceName).toBe('gateway');
    expect(valid.request.from).toBeTruthy();
    expect(valid.request.to).toBeTruthy();
  });

  it('countActiveDashboardFilters conta preset e servizio', () => {
    expect(countActiveDashboardFilters('all', null)).toBe(0);
    expect(countActiveDashboardFilters('6h', null)).toBe(1);
    expect(countActiveDashboardFilters('6h', 'api')).toBe(2);
  });

  it('createInitialRunRequest resetta paginazione e cursori', () => {
    const result = createInitialRunRequest({
      signals: ['logs'],
      timeRange: { from: 'a', to: 'b' },
      filters: {},
      page: 9,
      logsCursor: 'x',
      tracesCursor: 'y'
    });

    expect(result.page).toBe(1);
    expect(result.logsCursor).toBeUndefined();
    expect(result.tracesCursor).toBeUndefined();
  });

  it('createPageChangeRequest usa i cursori della pagina richiesta', () => {
    const logsMap = new Map<number, string>([[2, 'log-cursor-2']]);
    const tracesMap = new Map<number, string>([[2, 'trace-cursor-2']]);

    const logsRequest = createPageChangeRequest({
      lastRequest: {
        signals: ['logs', 'traces', 'metrics'],
        timeRange: { from: 'a', to: 'b' },
        filters: {}
      },
      signal: 'logs',
      nextPage: 2,
      logsCursorByPage: logsMap,
      tracesCursorByPage: tracesMap
    });

    expect(logsRequest.logsCursor).toBe('log-cursor-2');
    expect(logsRequest.tracesCursor).toBeUndefined();
  });

  it('createPageSizeRequest resetta cursori e imposta limit', () => {
    const request = createPageSizeRequest(
      {
        signals: ['logs'],
        timeRange: { from: 'a', to: 'b' },
        filters: {}
      },
      '200'
    );

    expect(request.signals).toEqual(['logs']);
    expect(request.limit).toBe(200);
    expect(request.page).toBe(1);
    expect(request.logsCursor).toBeUndefined();
    expect(request.tracesCursor).toBeUndefined();
  });

  it('storeNextCursors aggiorna map in base alla paginazione', () => {
    const logsMap = new Map<number, string>();
    const tracesMap = new Map<number, string>();

    storeNextCursors({
      pagination: {
        logs: {
          page: 1,
          limit: 10,
          total: 20,
          totalPages: 2,
          hasNext: true,
          nextCursor: 'n-log'
        },
        traces: {
          page: 1,
          limit: 10,
          total: 20,
          totalPages: 2,
          hasNext: true,
          nextCursor: 'n-trace'
        },
        metrics: {
          page: 1,
          limit: 10,
          total: 10,
          totalPages: 1,
          hasNext: false
        }
      },
      page: 1,
      logsCursorByPage: logsMap,
      tracesCursorByPage: tracesMap
    });

    expect(logsMap.get(2)).toBe('n-log');
    expect(tracesMap.get(2)).toBe('n-trace');
  });

  it('storeNextCursors può aggiornare solo il segnale richiesto', () => {
    const logsMap = new Map<number, string>([[2, 'old-log']]);
    const tracesMap = new Map<number, string>([[2, 'old-trace']]);

    storeNextCursors({
      pagination: {
        logs: { page: 1, limit: 10, total: 10, totalPages: 1, hasNext: false },
        traces: {
          page: 1,
          limit: 10,
          total: 20,
          totalPages: 2,
          hasNext: true,
          nextCursor: 'new-trace'
        }
      },
      page: 1,
      logsCursorByPage: logsMap,
      tracesCursorByPage: tracesMap,
      signals: ['traces']
    });

    expect(logsMap.get(2)).toBe('old-log');
    expect(tracesMap.get(2)).toBe('new-trace');
  });

  it('buildDashboardRequest preset all copre tutto storico', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-03-15T12:00:00.000Z'));

    const result = buildDashboardRequest({
      selectedService: null,
      rangePreset: 'all',
      fromInput: '',
      toInput: ''
    });

    expect(result.error).toBeNull();
    expect(result.request.from).toBe('1970-01-01T00:00:00.000Z');
    expect(result.request.to).toBe('2026-03-15T12:00:00.000Z');

    vi.useRealTimers();
  });
});
