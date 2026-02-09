import { apiRequest } from './api';

export type DashboardChartSetting = {
  key: string;
  enabled: boolean;
  order?: number;
};

export type DashboardSettingsResponse = {
  settings: DashboardChartSetting[];
};

export function getDashboardSettings(): Promise<DashboardSettingsResponse> {
  return apiRequest<DashboardSettingsResponse>('/api/dashboard/settings');
}

export function updateDashboardSettings(
  settings: DashboardChartSetting[]
): Promise<DashboardSettingsResponse> {
  return apiRequest<DashboardSettingsResponse>('/api/admin/dashboard/settings', {
    method: 'PUT',
    body: JSON.stringify({ settings })
  });
}
