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

export const DASHBOARD_GRID_COLUMNS = 6;
const MIN_W = 1;
const MAX_W = 6;
const FIXED_H = 2;

export const defaultDashboardSettings: DashboardChartSetting[] = [
  { key: 'apdex_gauge', enabled: true, order: 10, x: 0, y: 0, w: 2, h: 2 },
  { key: 'error_rate_gauge', enabled: true, order: 20, x: 2, y: 0, w: 2, h: 2 },
  { key: 'throughput_gauge', enabled: true, order: 30, x: 4, y: 0, w: 2, h: 2 },
  { key: 'latency_distribution', enabled: true, order: 40, x: 0, y: 2, w: 3, h: 2 },
  { key: 'latency_percentiles', enabled: true, order: 50, x: 3, y: 2, w: 3, h: 2 },
  { key: 'throughput_timeseries', enabled: true, order: 60, x: 0, y: 4, w: 3, h: 2 },
  { key: 'slowest_endpoints', enabled: true, order: 70, x: 3, y: 4, w: 3, h: 2 },
  { key: 'top_endpoints_throughput', enabled: true, order: 80, x: 0, y: 6, w: 3, h: 2 },
  { key: 'error_hotspots', enabled: true, order: 90, x: 3, y: 6, w: 3, h: 2 },
  { key: 'availability_trend', enabled: false, order: 100, x: 0, y: 8, w: 3, h: 2 },
  { key: 'error_budget_burn', enabled: false, order: 110, x: 3, y: 8, w: 3, h: 2 },
  { key: 'error_rate_timeseries', enabled: false, order: 120, x: 0, y: 10, w: 3, h: 2 },
  { key: 'service_latency_rank', enabled: false, order: 130, x: 3, y: 10, w: 3, h: 2 },
  { key: 'slo_compliance', enabled: false, order: 140, x: 0, y: 12, w: 3, h: 2 },
  { key: 'service_throughput', enabled: false, order: 150, x: 3, y: 12, w: 3, h: 2 }
];

const defaultSettingByKey = new Map(defaultDashboardSettings.map((setting) => [setting.key, setting]));

export const dashboardSettingsState = writable<DashboardSettingsState>(initialState);

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

function normalizeSetting(setting: DashboardChartSetting, fallback: DashboardChartSetting): DashboardChartSetting {
  const x = Number.isFinite(setting.x) ? Number(setting.x) : fallback.x;
  const y = Number.isFinite(setting.y) ? Number(setting.y) : fallback.y;
  const w = Number.isFinite(setting.w) ? Number(setting.w) : fallback.w;
  const order = Number.isFinite(setting.order) ? Number(setting.order) : fallback.order;

  let normalizedX = x < 0 ? fallback.x : x;
  let normalizedY = y < 0 ? fallback.y : y;
  let normalizedW = clamp(w, MIN_W, MAX_W);
  const normalizedH = FIXED_H;

  if (normalizedX >= DASHBOARD_GRID_COLUMNS) {
    normalizedX = fallback.x;
  }
  if (normalizedX + normalizedW > DASHBOARD_GRID_COLUMNS) {
    normalizedW = DASHBOARD_GRID_COLUMNS - normalizedX;
    if (normalizedW < MIN_W) {
      normalizedX = 0;
      normalizedW = MIN_W;
    }
  }
  if (normalizedY < 0) normalizedY = 0;

  return {
    key: fallback.key,
    enabled: setting.enabled ?? fallback.enabled,
    order: order > 0 ? order : fallback.order,
    x: normalizedX,
    y: normalizedY,
    w: normalizedW,
    h: normalizedH
  };
}

export function mergeWithDefaultDashboardSettings(
  settings: DashboardChartSetting[] | null | undefined
): DashboardChartSetting[] {
  const settingByKey = new Map((settings ?? []).map((setting) => [setting.key, setting]));
  return defaultDashboardSettings.map((fallback) => {
    const stored = settingByKey.get(fallback.key);
    return stored ? normalizeSetting(stored, fallback) : { ...fallback };
  });
}

function sortByLayout(settings: DashboardChartSetting[]): DashboardChartSetting[] {
  return [...settings].sort((a, b) => {
    if (a.y !== b.y) return a.y - b.y;
    if (a.x !== b.x) return a.x - b.x;
    return (a.order ?? 0) - (b.order ?? 0);
  });
}

export function buildOrderedDashboardSettings(
  settings: DashboardChartSetting[] | null | undefined
): DashboardChartSetting[] {
  return sortByLayout(mergeWithDefaultDashboardSettings(settings));
}

export async function loadDashboardSettings() {
  dashboardSettingsState.update((state) => ({ ...state, loading: true, error: null }));
  try {
    const response = await getDashboardSettings();
    dashboardSettingsState.update((state) => ({
      ...state,
      loading: false,
      settings: mergeWithDefaultDashboardSettings(response.settings ?? [])
    }));
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Errore caricamento impostazioni';
    dashboardSettingsState.update((state) => ({ ...state, loading: false, error: message }));
  }
}

export async function saveDashboardSettings(settings: DashboardChartSetting[]) {
  const normalized = mergeWithDefaultDashboardSettings(settings);
  dashboardSettingsState.update((state) => ({ ...state, loading: true, error: null }));
  try {
    const response = await updateDashboardSettings(normalized);
    dashboardSettingsState.update((state) => ({
      ...state,
      loading: false,
      settings: mergeWithDefaultDashboardSettings(response.settings ?? [])
    }));
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Errore salvataggio impostazioni';
    dashboardSettingsState.update((state) => ({ ...state, loading: false, error: message }));
    throw err;
  }
}

export function getDefaultDashboardSettings(): DashboardChartSetting[] {
  return defaultDashboardSettings.map((setting) => ({ ...setting }));
}

export function isChartEnabled(key: string): boolean {
  const state = get(dashboardSettingsState);
  const normalized = mergeWithDefaultDashboardSettings(state.settings);
  const match = normalized.find((setting) => setting.key === key);
  return match ? match.enabled : false;
}

export function getOrderedChartKeys(): string[] {
  return getOrderedChartSettings().map((setting) => setting.key);
}

export function getOrderedChartSettings(): DashboardChartSetting[] {
  const state = get(dashboardSettingsState);
  return buildOrderedDashboardSettings(state.settings);
}

export function getChartLayout(key: string): DashboardChartSetting | null {
  const defaultSetting = defaultSettingByKey.get(key);
  if (!defaultSetting) return null;
  const state = get(dashboardSettingsState);
  const stored = state.settings.find((setting) => setting.key === key);
  return stored ? normalizeSetting(stored, defaultSetting) : { ...defaultSetting };
}
