import { describe, expect, it } from 'vitest';
import {
  buildChangePasswordPayload,
  buildFirstLoginPasswordPayload,
  buildLoginPayload
} from '../../../src/services/auth.mapper';
import { buildDashboardMetricsPayload } from '../../../src/services/dashboard.mapper';
import { buildAttributesQuery, toStringList } from '../../../src/services/query.mapper';

describe('service mappers', () => {
  it('toStringList estrae solo stringhe', () => {
    expect(toStringList(['a', 10, 'b', null])).toEqual(['a', 'b']);
    expect(toStringList({ items: [] })).toEqual([]);
  });

  it('buildAttributesQuery serializza il parametro q', () => {
    expect(buildAttributesQuery('service.name')).toBe('q=service.name');
  });

  it('buildLoginPayload e buildFirstLoginPasswordPayload producono body coerenti', () => {
    expect(buildLoginPayload('admin', 'secret')).toEqual({ username: 'admin', password: 'secret' });
    expect(buildFirstLoginPasswordPayload('new-secret')).toEqual({ newPassword: 'new-secret' });
  });

  it('buildChangePasswordPayload include currentPassword solo quando presente', () => {
    expect(buildChangePasswordPayload('new-secret', 'old-secret')).toEqual({
      newPassword: 'new-secret',
      currentPassword: 'old-secret'
    });
    expect(buildChangePasswordPayload('new-secret')).toEqual({ newPassword: 'new-secret' });
  });

  it('buildDashboardMetricsPayload normalizza il payload request', () => {
    expect(
      buildDashboardMetricsPayload({
        from: '2026-03-15T10:00:00.000Z',
        to: '2026-03-15T11:00:00.000Z',
        serviceName: 'api'
      })
    ).toEqual({
      from: '2026-03-15T10:00:00.000Z',
      to: '2026-03-15T11:00:00.000Z',
      serviceName: 'api'
    });
  });
});
