import { describe, expect, it, vi } from 'vitest';
import { buildRollingDashboardRequest } from '../../../src/lib/stores/dashboard.domain';

describe('dashboard.domain', () => {
  it('ritorna la request invariata se from/to non sono presenti', () => {
    expect(buildRollingDashboardRequest({ serviceName: 'api' })).toEqual({ serviceName: 'api' });
  });

  it('calcola rolling range mantenendo la durata originale', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-03-15T12:00:00.000Z'));

    const result = buildRollingDashboardRequest({
      from: '2026-03-15T11:00:00.000Z',
      to: '2026-03-15T11:30:00.000Z',
      serviceName: 'gateway'
    });

    expect(result.from).toBe('2026-03-15T11:30:00.000Z');
    expect(result.to).toBe('2026-03-15T12:00:00.000Z');
    expect(result.serviceName).toBe('gateway');

    vi.useRealTimers();
  });
});
