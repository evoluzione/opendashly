import { get, writable } from 'svelte/store';
import type { DashboardChartSetting } from '../../services/dashboard_settings';
import { getDashboardSettings, updateDashboardSettings } from '../../services/dashboard_settings';

type DashboardSettingsState = {
  loading: boolean;
  error: string | null;
  settings: DashboardChartSetting[];
};

const initialState: DashboardSettingsState = {
  loading: false,
  error: null,
  settings: []
};

export const dashboardSettingsState = writable<DashboardSettingsState>(initialState);

const defaultOrderKeys = [
  'apdex_gauge',
  'error_rate_gauge',
  'throughput_gauge',
  'latency_distribution',
  'latency_percentiles',
  'throughput_timeseries',
  'error_rate_timeseries',
  'status_codes',
  'slowest_endpoints',
  'top_endpoints_throughput',
  'error_hotspots'
];

export async function loadDashboardSettings() {
  dashboardSettingsState.update((state) => ({ ...state, loading: true, error: null }));
  try {
    const response = await getDashboardSettings();
    dashboardSettingsState.update((state) => ({
      ...state,
      loading: false,
      settings: response.settings ?? []
    }));
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Errore caricamento impostazioni';
    dashboardSettingsState.update((state) => ({ ...state, loading: false, error: message }));
  }
}

export async function saveDashboardSettings(settings: DashboardChartSetting[]) {
  dashboardSettingsState.update((state) => ({ ...state, loading: true, error: null }));
  try {
    const response = await updateDashboardSettings(settings);
    dashboardSettingsState.update((state) => ({
      ...state,
      loading: false,
      settings: response.settings ?? []
    }));
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Errore salvataggio impostazioni';
    dashboardSettingsState.update((state) => ({ ...state, loading: false, error: message }));
  }
}

export function isChartEnabled(key: string): boolean {
  const state = get(dashboardSettingsState);
  if (!state.settings || state.settings.length === 0) {
    return true;
  }
  const match = state.settings.find((setting) => setting.key === key);
  return match ? match.enabled : true;
}

export function getOrderedChartKeys(): string[] {
  const state = get(dashboardSettingsState);
  if (!state.settings || state.settings.length === 0) {
    return [...defaultOrderKeys];
  }
  const orderMap = new Map<string, number>();
  defaultOrderKeys.forEach((key, index) => orderMap.set(key, (index + 1) * 10));
  for (const setting of state.settings) {
    if (typeof setting.order === 'number' && setting.order > 0) {
      orderMap.set(setting.key, setting.order);
    }
  }
  return [...defaultOrderKeys].sort((a, b) => {
    return (orderMap.get(a) ?? 0) - (orderMap.get(b) ?? 0);
  });
}
