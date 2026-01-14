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
  autoRefreshSeconds: null
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
    void executeQuery(currentState.lastRequest, { retainResult: true });
  }, currentState.autoRefreshSeconds * 1000);
}

export async function executeQuery(
  request: QueryRequest,
  options: { retainResult?: boolean } = {}
) {
  console.debug('query.execute.start', { retainResult: !!options.retainResult, request });
  queryState.update((state) => ({
    ...state,
    loading: true,
    error: null,
    result: options.retainResult ? state.result : null,
    lastRequest: request
  }));
  try {
    const result = await runQuery(request);
    console.debug('query.execute.success', { runId: result.runId, status: result.status });
    queryState.update((state) => ({ ...state, loading: false, error: null, result }));
    scheduleAutoRefresh();
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
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
    const message = err instanceof Error ? err.message : 'Failed to load services';
    servicesState.update((state) => ({ ...state, loading: false, error: message }));
  }
}

export function selectService(value: string) {
  servicesState.update((state) => ({ ...state, selectedService: value }));
}

export function setAutoRefresh(seconds: number | null) {
  queryState.update((state) => ({ ...state, autoRefreshSeconds: seconds }));
  scheduleAutoRefresh();
}

export function stopAutoRefresh() {
  resetRefreshTimer();
  queryState.update((state) => ({ ...state, autoRefreshSeconds: null }));
}
