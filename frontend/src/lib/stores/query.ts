import { get, writable } from 'svelte/store';
import type { QueryRunResult, QueryRequest } from '../../services/query';
import { runQuery } from '../../services/query';

type QueryState = {
  loading: boolean;
  error: string | null;
  result: QueryRunResult | null;
  lastRequest: QueryRequest | null;
  autoRefreshSeconds: number | null;
};

const initial: QueryState = {
  loading: false,
  error: null,
  result: null,
  lastRequest: null,
  autoRefreshSeconds: null
};

export const queryState = writable<QueryState>(initial);

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

export function setAutoRefresh(seconds: number | null) {
  queryState.update((state) => ({ ...state, autoRefreshSeconds: seconds }));
  scheduleAutoRefresh();
}

export function stopAutoRefresh() {
  resetRefreshTimer();
  queryState.update((state) => ({ ...state, autoRefreshSeconds: null }));
}
