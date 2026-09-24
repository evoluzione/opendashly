import { describe, expect, it } from 'vitest';
import { linkFilters, linkUrl, toCsv, type AssistantLink } from '../../../src/services/assistantLinks';

const now = new Date('2026-09-24T15:00:00Z');

describe('assistant links', () => {
  it('uses a relative preset for "the last hour" up to now', () => {
    const link: AssistantLink = {
      label: 'Log di errore',
      kind: 'logs',
      service: 'payment-service',
      severity: 'ERROR,FATAL',
      from: '2026-09-24T14:00:00Z',
      to: '2026-09-24T15:00:00Z',
    };
    const params = new URL(linkUrl(link, now), 'http://x').searchParams;
    expect(params.get('tab')).toBe('logs');
    expect(params.get('service')).toBe('payment-service');
    expect(params.get('severity')).toBe('ERROR,FATAL');
    expect(params.get('range')).toBe('1h');
    expect(params.get('autorun')).toBe('1');
  });

  it('uses a custom range for past windows such as yesterday', () => {
    const link: AssistantLink = {
      label: 'Trace in errore',
      kind: 'traces',
      errorsOnly: true,
      search: '/search',
      from: '2026-09-23T00:00:00Z',
      to: '2026-09-24T00:00:00Z',
    };
    const params = new URL(linkUrl(link, now), 'http://x').searchParams;
    expect(params.get('tab')).toBe('tracce');
    expect(params.get('range')).toBe('custom');
    expect(params.get('errors')).toBe('with_errors');
    expect(params.get('search')).toBe('/search');
    expect(new Date(params.get('from')!).toISOString()).toBe('2026-09-23T00:00:00.000Z');
  });

  it('opens a single trace by id', () => {
    const params = new URL(linkUrl({ label: 'Trace', kind: 'trace', traceId: 'abc' }, now), 'http://x').searchParams;
    expect(params.get('traceId')).toBe('abc');
    expect(params.get('tab')).toBe('tracce');
  });

  it('builds the same filters the search page would send', () => {
    expect(linkFilters({ label: '', kind: 'traces', service: 'cart-service', errorsOnly: true, search: 'GET' })).toEqual([
      { connector: 'AND', key: 'service.name', operator: '=', value: 'cart-service' },
      { connector: 'AND', key: 'trace_error_scope', operator: '=', value: 'with_errors' },
      { connector: 'AND', key: 'trace_or_span', operator: 'contains', value: 'GET' },
    ]);
  });

  it('escapes CSV cells', () => {
    expect(toCsv(['a', 'b'], [{ a: 'x,y', b: 'say "hi"\nbye' }])).toBe('a,b\n"x,y","say ""hi""\nbye"\n');
  });
});
