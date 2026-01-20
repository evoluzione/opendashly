import { get, writable } from 'svelte/store';
import type { QueryRunResult, QueryRequest } from '../../services/query';
import { runQuery } from '../../services/query';
import { fetchServices } from '../../services/services';

type QueryState = {
  loading: boolean;
  error: string | null;
  result: QueryRunResult | null;
  lastRequest: QueryRequest | null;
  autoRefreshSeconds: number | null;
  autoRefreshRangeMinutes: number | null;
};

type ServiceState = {
  services: string[];
  selectedService: string;
  loading: boolean;
  error: string | null;
};

const initial: QueryState = {
  loading: false,
  error: null,
  result: null,
  lastRequest: null,
  autoRefreshSeconds: null,
  autoRefreshRangeMinutes: null
};

export const queryState = writable<QueryState>(initial);

const servicesInitial: ServiceState = {
  services: [],
  selectedService: 'Tutti',
  loading: false,
  error: null
};

export const servicesState = writable<ServiceState>(servicesInitial);

let refreshTimer: ReturnType<typeof setInterval> | null = null;

function logKey(entry: any) {
  const timestamp = entry?.timestamp ?? '';
  const traceId = entry?.traceId ?? '';
  const spanId = entry?.spanId ?? '';
  const body =
    typeof entry?.body === 'string' ? entry.body : JSON.stringify(entry?.body ?? '');
  return `${timestamp}|${traceId}|${spanId}|${body}`;
}

function traceKey(entry: any) {
  return entry?.traceId ?? '';
}

function mergeByKey<T>(
  latest: T[],
  previous: T[],
  keyFn: (item: T) => string,
  limit?: number
) {
  const seen = new Set<string>();
  const merged: T[] = [];
  for (const item of latest) {
    const key = keyFn(item);
    if (seen.has(key)) continue;
    seen.add(key);
    merged.push(item);
  }
  for (const item of previous) {
    const key = keyFn(item);
    if (seen.has(key)) continue;
    seen.add(key);
    merged.push(item);
  }
  return typeof limit === 'number' && limit > 0 ? merged.slice(0, limit) : merged;
}

function mergeResult(
  previous: QueryRunResult,
  latest: QueryRunResult,
  request: QueryRequest
) {
  const logLimit = request.limit ?? latest.pagination?.logs?.limit;
  const traceLimit = request.limit ?? latest.pagination?.traces?.limit;
  return {
    ...latest,
    results: {
      ...latest.results,
      logs: mergeByKey(latest.results.logs ?? [], previous.results.logs ?? [], logKey, logLimit),
      traces: mergeByKey(
        latest.results.traces ?? [],
        previous.results.traces ?? [],
        traceKey,
        traceLimit
      )
    }
  };
}

function resetRefreshTimer() {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
}

function scheduleAutoRefresh() {
  resetRefreshTimer();
  const currentState = get(queryState);
  if (!currentState.lastRequest || !currentState.autoRefreshSeconds) {
    return;
  }
  refreshTimer = setInterval(() => {
    const latest = get(queryState);
    if (!latest.lastRequest) {
      return;
    }
    const request = latest.autoRefreshRangeMinutes
      ? buildRollingRangeRequest(latest.lastRequest, latest.autoRefreshRangeMinutes)
      : latest.lastRequest;
    void executeQuery(request, { retainResult: true });
  }, currentState.autoRefreshSeconds * 1000);
}

function buildRollingRangeRequest(request: QueryRequest, minutes: number): QueryRequest {
  const now = new Date();
  const from = new Date(now.getTime() - minutes * 60 * 1000);
  return {
    ...request,
    timeRange: { from: from.toISOString(), to: now.toISOString() }
  };
}

export async function executeQuery(
  request: QueryRequest,
  options: { retainResult?: boolean } = {}
) {
  console.debug('query.execute.start', { retainResult: !!options.retainResult, request });
  const previousResult = get(queryState).result;
  queryState.update((state) => ({
    ...state,
    loading: true,
    error: null,
    result: options.retainResult ? state.result : null,
    lastRequest: request
  }));
  try {
    const response = await runQuery(request);
    console.debug('query.execute.success', { runId: response.runId, status: response.status });
    const shouldMerge =
      !!options.retainResult &&
      !!previousResult &&
      (request.page === undefined || request.page === 1);
    const result = shouldMerge ? mergeResult(previousResult, response, request) : response;
    queryState.update((state) => ({ ...state, loading: false, error: null, result }));
    scheduleAutoRefresh();
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Errore sconosciuto';
    console.debug('query.execute.error', { message, error: err });
    queryState.update((state) => ({
      ...state,
      loading: false,
      error: message,
      result: options.retainResult ? state.result : null
    }));
  }
}

export async function loadServices() {
  servicesState.update((state) => ({ ...state, loading: true, error: null }));
  try {
    const response = await fetchServices();
    servicesState.update((state) => ({
      ...state,
      services: response.services ?? [],
      loading: false,
      error: null
    }));
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Impossibile caricare i servizi';
    servicesState.update((state) => ({ ...state, loading: false, error: message }));
  }
}

export function selectService(value: string) {
  servicesState.update((state) => ({ ...state, selectedService: value }));
}

export function setAutoRefresh(seconds: number | null, rangeMinutes: number | null = null) {
  queryState.update((state) => ({
    ...state,
    autoRefreshSeconds: seconds,
    autoRefreshRangeMinutes: rangeMinutes
  }));
  scheduleAutoRefresh();
}

export function stopAutoRefresh() {
  resetRefreshTimer();
  queryState.update((state) => ({
    ...state,
    autoRefreshSeconds: null,
    autoRefreshRangeMinutes: null
  }));
}
