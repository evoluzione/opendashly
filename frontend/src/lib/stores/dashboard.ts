import { get, writable } from 'svelte/store';
import type { DashboardResponse, DashboardRequest } from '../../services/dashboard';
import { fetchDashboardMetrics } from '../../services/dashboard';

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
    // Refresh with rolling time range
    const now = new Date();
    const originalDuration = new Date(latest.lastRequest.to).getTime() - new Date(latest.lastRequest.from).getTime();
    const from = new Date(now.getTime() - originalDuration);
    void loadDashboard({
      ...latest.lastRequest,
      from: from.toISOString(),
      to: now.toISOString()
    });
  }, currentState.autoRefreshSeconds * 1000);
}

export async function loadDashboard(request: DashboardRequest) {
  console.debug('dashboard.load.start', { request });
  dashboardState.update((state) => ({
    ...state,
    loading: true,
    error: null,
    lastRequest: request
  }));
  try {
    const response = await fetchDashboardMetrics(request);
    console.debug('dashboard.load.success', { response });
    dashboardState.update((state) => ({
      ...state,
      loading: false,
      error: null,
      data: response
    }));
    scheduleAutoRefresh();
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Errore sconosciuto';
    console.debug('dashboard.load.error', { message, error: err });
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

export function selectDashboardService(serviceName: string | null) {
  dashboardState.update((state) => ({
    ...state,
    selectedService: serviceName
  }));

  // Ricarica la dashboard con il nuovo filtro servizio
  const currentState = get(dashboardState);
  if (currentState.lastRequest) {
    void loadDashboard({
      ...currentState.lastRequest,
      serviceName: serviceName || undefined
    });
  }
}

