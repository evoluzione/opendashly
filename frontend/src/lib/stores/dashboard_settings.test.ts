import { describe, expect, it } from 'vitest';
import type { DashboardChartSetting } from '../../services/dashboard_settings';
import {
  buildOrderedDashboardSettings,
  defaultDashboardSettings,
  mergeWithDefaultDashboardSettings
} from './dashboard_settings';

describe('dashboard_settings store helpers', () => {
  it('fills missing layout values from defaults', () => {
    const input: DashboardChartSetting[] = [
      {
        key: 'apdex_gauge',
        enabled: false,
        order: 0,
        x: -2,
        y: -1,
        w: 0,
        h: 0
      }
    ];

    const merged = mergeWithDefaultDashboardSettings(input);
    const apdex = merged.find((setting) => setting.key === 'apdex_gauge');
    expect(apdex).toBeTruthy();
    expect(apdex).toMatchObject({
      enabled: false,
      order: 10,
      x: 0,
      y: 0,
      w: 2,
      h: 2
    });
  });

  it('returns defaults when settings are empty', () => {
    const merged = mergeWithDefaultDashboardSettings([]);
    expect(merged).toHaveLength(defaultDashboardSettings.length);
    expect(merged[0]).toMatchObject(defaultDashboardSettings[0]);
  });

  it('orders widgets by y/x and uses order as fallback', () => {
    const ordered = buildOrderedDashboardSettings([
      {
        key: 'apdex_gauge',
        enabled: true,
        order: 30,
        x: 3,
        y: 2,
        w: 3,
        h: 2
      },
      {
        key: 'error_rate_gauge',
        enabled: true,
        order: 10,
        x: 0,
        y: 2,
        w: 2,
        h: 2
      }
    ]);

    const apdexIndex = ordered.findIndex((setting) => setting.key === 'apdex_gauge');
    const errorRateIndex = ordered.findIndex((setting) => setting.key === 'error_rate_gauge');
    expect(apdexIndex).toBeGreaterThan(errorRateIndex);
  });
});
