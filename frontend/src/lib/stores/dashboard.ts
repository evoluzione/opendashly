import { get, writable } from 'svelte/store';
import type { DashboardResponse, DashboardRequest } from '../../services/dashboard';
import { fetchDashboardMetrics } from '../../services/dashboard';
import { buildRollingDashboardRequest } from './dashboard.domain';

type DashboardState = {
  loading: boolean;
  error: string | null;
  data: DashboardResponse | null;
  lastRequest: DashboardRequest | null;
  autoRefreshSeconds: number | null;
  selectedService: string | null;
};

const initial: DashboardState = {
  loading: false,
  error: null,
  data: null,
  lastRequest: null,
  autoRefreshSeconds: null,
  selectedService: null
};

export const dashboardState = writable<DashboardState>(initial);

let refreshTimer: ReturnType<typeof setInterval> | null = null;

function resetRefreshTimer() {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
}

function scheduleAutoRefresh() {
  resetRefreshTimer();
  const currentState = get(dashboardState);
  if (!currentState.lastRequest || !currentState.autoRefreshSeconds) {
    return;
  }
  refreshTimer = setInterval(() => {
    const latest = get(dashboardState);
    if (!latest.lastRequest) {
      return;
    }
    void loadDashboard(buildRollingDashboardRequest(latest.lastRequest));
  }, currentState.autoRefreshSeconds * 1000);
}

export async function loadDashboard(request: DashboardRequest) {
  dashboardState.update((state) => ({
    ...state,
    loading: true,
    error: null,
    lastRequest: request
  }));
  try {
    const response = await fetchDashboardMetrics(request);
    dashboardState.update((state) => ({
      ...state,
      loading: false,
      error: null,
      data: response
    }));
    scheduleAutoRefresh();
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Errore sconosciuto';
    dashboardState.update((state) => ({
      ...state,
      loading: false,
      error: message
    }));
  }
}

export function setDashboardAutoRefresh(seconds: number | null) {
  dashboardState.update((state) => ({
    ...state,
    autoRefreshSeconds: seconds
  }));
  scheduleAutoRefresh();
}

export function stopDashboardAutoRefresh() {
  resetRefreshTimer();
  dashboardState.update((state) => ({
    ...state,
    autoRefreshSeconds: null
  }));
}

export function resetDashboard() {
  resetRefreshTimer();
  dashboardState.set(initial);
}

export function selectDashboardService(
  serviceName: string | null,
  options: { reload?: boolean } = {}
) {
  dashboardState.update((state) => ({
    ...state,
    selectedService: serviceName
  }));

  const shouldReload = options.reload ?? true;
  if (!shouldReload) {
    return;
  }

  // Ricarica la dashboard con il nuovo filtro servizio
  const currentState = get(dashboardState);
  if (currentState.lastRequest) {
    void loadDashboard({
      ...currentState.lastRequest,
      serviceName: serviceName || undefined
    });
  }
}
