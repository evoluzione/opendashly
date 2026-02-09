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
  isLiveUpdate: boolean;
};

type ServiceState = {
  services: string[];
  selectedService: string;
  selectedLogLevel: string;
  loading: boolean;
  error: string | null;
};

const initial: QueryState = {
  loading: false,
  error: null,
  result: null,
  lastRequest: null,
  autoRefreshSeconds: null,
  autoRefreshRangeMinutes: null,
  isLiveUpdate: false
};

export const queryState = writable<QueryState>(initial);

const servicesInitial: ServiceState = {
  services: [],
  selectedService: 'Tutti',
  selectedLogLevel: 'Tutti',
  loading: false,
  error: null
};

export const servicesState = writable<ServiceState>(servicesInitial);

export function selectLogLevel(value: string) {
  servicesState.update((state) => ({ ...state, selectedLogLevel: value }));
}

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
    const currentState = get(queryState);
    if (!currentState.lastRequest) {
      return;
    }

    let request = currentState.lastRequest;

    // Incremental polling optimization for "Live" mode (1s)
    if (currentState.autoRefreshSeconds === 1) {
      const maxTimestamp = getMaxTimestamp(currentState.result);
      if (maxTimestamp) {
        // Fetch only new data from last max timestamp
        // Add minimal offset to avoid duplicates if precision allows, or rely on merge dedup
        const fromDate = new Date(new Date(maxTimestamp).getTime() + 1);
        request = {
          ...currentState.lastRequest,
          timeRange: {
            from: fromDate.toISOString(),
            to: new Date().toISOString()
          }
        };
      } else if (currentState.autoRefreshRangeMinutes) {
        // Fallback to rolling if no data yet
        request = buildRollingRangeRequest(currentState.lastRequest, currentState.autoRefreshRangeMinutes);
      }
    } else if (currentState.autoRefreshRangeMinutes) {
      request = buildRollingRangeRequest(currentState.lastRequest, currentState.autoRefreshRangeMinutes);
    }

    void executeQuery(request, { retainResult: true, isBackground: true });
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

function getMaxTimestamp(result: QueryRunResult | null): string | null {
  if (!result || !result.results) return null;
  let maxTime = "";

  // Check logs
  if (result.results.logs) {
    for (const log of result.results.logs) {
      if (log.timestamp > maxTime) maxTime = log.timestamp;
    }
  }

  // Check traces
  if (result.results.traces) {
    for (const trace of result.results.traces) {
      if (trace.timestamp > maxTime) maxTime = trace.timestamp;
    }
  }

  return maxTime || null;
}

// Debug helper
function debugLog(message: string, ...args: any[]) {
  if (import.meta.env.VITE_DEBUG_QUERY === 'true') {
  }
}

export async function executeQuery(
  request: QueryRequest,
  options: { retainResult?: boolean; isBackground?: boolean } = {}
) {
  debugLog('query.execute.start', { retainResult: !!options.retainResult, isBackground: !!options.isBackground, request });
  const previousResult = get(queryState).result;

  // Set loading only if NOT a background refresh
  if (!options.isBackground) {
    queryState.update((state) => ({
      ...state,
      loading: true,
      error: null,
      result: options.retainResult ? state.result : null,
      lastRequest: request
    }));
  } else {
    // Even in background, we update lastRequest
    queryState.update((state) => ({ ...state, lastRequest: request }));
  }

  try {
    const response = await runQuery(request);
    debugLog('query.execute.success', { runId: response.runId, status: response.status });
    const shouldMerge =
      (!!options.retainResult || !!options.isBackground) &&
      !!previousResult &&
      (request.page === undefined || request.page === 1);
    const result = shouldMerge ? mergeResult(previousResult, response, request) : response;
    queryState.update((state) => ({ ...state, loading: false, error: null, result, isLiveUpdate: !!options.isBackground }));
    scheduleAutoRefresh();
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Errore sconosciuto';
    debugLog('query.execute.error', { message, error: err });
    queryState.update((state) => ({
      ...state,
      loading: false,
      error: message,
      result: options.retainResult || options.isBackground ? state.result : null
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
