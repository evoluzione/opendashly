import { get, writable } from 'svelte/store';
import type { QueryRunResult, QueryRequest } from '../../services/query';
import { runQuery } from '../../services/query';
import { fetchServices } from '../../services/services';
import {
  buildAutoRefreshRequest,
  mergeResult,
  mergeSingleSignalResult,
  replaceSingleSignalResult
} from './query.domain';

type QueryState = {
  loading: boolean;
  error: string | null;
  warnings: string[];
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
  warnings: [],
  result: null,
  lastRequest: null,
  autoRefreshSeconds: null,
  autoRefreshRangeMinutes: null,
  isLiveUpdate: false
};

export const queryState = writable<QueryState>(initial);

const servicesInitial: ServiceState = {
  services: [],
  selectedService: '',
  selectedLogLevel: '',
  loading: false,
  error: null
};

export const servicesState = writable<ServiceState>(servicesInitial);

function normalizeAllSelection(value: string): string {
  const trimmed = String(value ?? '').trim();
  return trimmed === 'Tutti' || trimmed.toLowerCase() === 'all' ? '' : trimmed;
}

export function selectLogLevel(value: string) {
  servicesState.update((state) => ({ ...state, selectedLogLevel: normalizeAllSelection(value) }));
}

let refreshTimer: ReturnType<typeof setInterval> | null = null;

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
    const request = buildAutoRefreshRequest({
      lastRequest: latest.lastRequest,
      autoRefreshSeconds: latest.autoRefreshSeconds,
      autoRefreshRangeMinutes: latest.autoRefreshRangeMinutes,
      result: latest.result
    });

    if (!request) {
      return;
    }

    void executeQuery(request, { retainResult: true, isBackground: true });
  }, currentState.autoRefreshSeconds * 1000);
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
  const requestedSignal = request.signals?.length === 1 ? request.signals[0] : null;
  const singleSignal =
    requestedSignal === 'logs' || requestedSignal === 'traces' ? requestedSignal : null;

  // Set loading only if NOT a background refresh
  if (!options.isBackground) {
    queryState.update((state) => ({
      ...state,
      loading: true,
      error: null,
      warnings: [],
      result: options.retainResult || singleSignal ? state.result : null,
      lastRequest: request
    }));
  } else {
    // Even in background, we update lastRequest
    queryState.update((state) => ({ ...state, lastRequest: request }));
  }

  try {
    const response = await runQuery(request);
    debugLog('query.execute.success', { runId: response.runId, status: response.status });
    const isRetained = !!options.retainResult || !!options.isBackground;

    let result = response;
    if (isRetained && previousResult && singleSignal) {
      result = mergeSingleSignalResult(previousResult, response, singleSignal, request);
    } else if (!isRetained && previousResult && singleSignal) {
      result = replaceSingleSignalResult(previousResult, response, singleSignal);
    } else if (
      isRetained &&
      previousResult &&
      (request.page === undefined || request.page === 1)
    ) {
      result = mergeResult(previousResult, response, request);
    }
    const warnings =
      response.signalErrors && Object.keys(response.signalErrors).length > 0
        ? Object.entries(response.signalErrors).map(([signal, reason]) => `${signal}: ${reason}`)
        : [];

    queryState.update((state) => ({
      ...state,
      loading: false,
      error: null,
      warnings,
      result,
      isLiveUpdate: !!options.isBackground
    }));
    scheduleAutoRefresh();
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Errore sconosciuto';
    debugLog('query.execute.error', { message, error: err });
    const refreshWarning = `refresh failed: ${message}`;
    queryState.update((state) => ({
      ...state,
      loading: false,
      error: message,
      warnings:
        (options.retainResult || options.isBackground) &&
        state.result &&
        !state.warnings.includes(refreshWarning)
          ? [...state.warnings, refreshWarning]
          : state.warnings,
      result: options.retainResult || options.isBackground || singleSignal ? state.result : null
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
  servicesState.update((state) => ({ ...state, selectedService: normalizeAllSelection(value) }));
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
