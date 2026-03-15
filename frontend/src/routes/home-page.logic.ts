import type { QueryRequest, QueryRunResult } from '../services/query';
import { t, type Locale, getLocaleTag } from '../lib/i18n';

export type DashboardTab = 'logs' | 'metriche' | 'tracce';

export type DashboardRangePreset =
  | '5m'
  | '15m'
  | '30m'
  | '1h'
  | '6h'
  | '24h'
  | '7d'
  | 'all'
  | 'custom';

export function getDashboardTimeRangeOptions(locale: Locale): Array<{
  value: DashboardRangePreset;
  label: string;
}> {
  return [
    { value: '5m', label: t(locale, 'range.last5m') },
    { value: '15m', label: t(locale, 'range.last15m') },
    { value: '30m', label: t(locale, 'range.last30m') },
    { value: '1h', label: t(locale, 'range.last1h') },
    { value: '6h', label: t(locale, 'range.last6h') },
    { value: '24h', label: t(locale, 'range.last24h') },
    { value: '7d', label: t(locale, 'range.last7d') },
    { value: 'all', label: t(locale, 'range.all') },
    { value: 'custom', label: t(locale, 'range.custom') }
  ];
}

const dashboardPresetMinutes: Record<Exclude<DashboardRangePreset, 'custom'>, number | null> = {
  '5m': 5,
  '15m': 15,
  '30m': 30,
  '1h': 60,
  '6h': 360,
  '24h': 1440,
  '7d': 10080,
  all: null
};

export function formatDateTimeLocal(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  const y = date.getFullYear();
  const m = pad(date.getMonth() + 1);
  const d = pad(date.getDate());
  const h = pad(date.getHours());
  const min = pad(date.getMinutes());
  return `${y}-${m}-${d}T${h}:${min}`;
}

export function toIsoFromLocal(value: string): string | null {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return date.toISOString();
}

export function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export function defaultCustomRangeInputs(now: Date = new Date()): {
  fromInput: string;
  toInput: string;
} {
  const from = new Date(now.getTime() - 6 * 60 * 60 * 1000);
  return {
    fromInput: formatDateTimeLocal(from),
    toInput: formatDateTimeLocal(now)
  };
}

export function buildDashboardRequest(args: {
  selectedService: string | null;
  rangePreset: DashboardRangePreset;
  fromInput: string;
  toInput: string;
  locale?: Locale;
}): {
  request: { from?: string; to?: string; serviceName?: string };
  error: string | null;
} {
  const activeLocale = args.locale ?? 'it';
  const request: { from?: string; to?: string; serviceName?: string } = {};

  if (args.selectedService) {
    request.serviceName = args.selectedService;
  }

  if (args.rangePreset === 'custom') {
    const fromIso = toIsoFromLocal(args.fromInput);
    const toIso = toIsoFromLocal(args.toInput);

    if (!fromIso || !toIso) {
      return {
        request,
        error: t(activeLocale, 'range.errorInvalidDate')
      };
    }

    if (new Date(toIso).getTime() <= new Date(fromIso).getTime()) {
      return {
        request,
        error: t(activeLocale, 'range.errorEndBeforeStart')
      };
    }

    request.from = fromIso;
    request.to = toIso;
    return { request, error: null };
  }

  const minutes = dashboardPresetMinutes[args.rangePreset as Exclude<DashboardRangePreset, 'custom'>];
  const now = new Date();

  if (minutes === null) {
    request.from = new Date(0).toISOString();
    request.to = now.toISOString();
  } else {
    request.from = new Date(now.getTime() - minutes * 60 * 1000).toISOString();
    request.to = now.toISOString();
  }

  return { request, error: null };
}

export function formatLastRefresh(date: Date | null, locale: Locale): string {
  if (!date) return '--';
  return date.toLocaleTimeString(getLocaleTag(locale), {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  });
}

export function countActiveDashboardFilters(
  rangePreset: DashboardRangePreset,
  selectedService: string | null
): number {
  return (rangePreset === 'all' ? 0 : 1) + (selectedService ? 1 : 0);
}

export function createInitialRunRequest(request: QueryRequest): QueryRequest {
  return {
    ...request,
    page: 1,
    logsCursor: undefined,
    tracesCursor: undefined
  };
}

export function createPageChangeRequest(args: {
  lastRequest: QueryRequest;
  signal: 'logs' | 'traces' | 'metrics';
  nextPage: number;
  logsCursorByPage: Map<number, string>;
  tracesCursorByPage: Map<number, string>;
}): QueryRequest {
  const page = args.nextPage < 1 ? 1 : args.nextPage;
  return {
    ...args.lastRequest,
    signals: [args.signal],
    page,
    logsCursor:
      args.signal === 'logs' && page > 1 ? args.logsCursorByPage.get(page) : undefined,
    tracesCursor:
      args.signal === 'traces' && page > 1
        ? args.tracesCursorByPage.get(page)
        : undefined
  };
}

export function createPageSizeRequest(lastRequest: QueryRequest, pageSize: string): QueryRequest {
  const limit = Number(pageSize) || 100;
  return {
    ...lastRequest,
    signals: ['logs', 'traces', 'metrics'],
    limit,
    page: 1,
    logsCursor: undefined,
    tracesCursor: undefined
  };
}

export function storeNextCursors(args: {
  pagination: QueryRunResult['pagination'] | undefined;
  page: number;
  logsCursorByPage: Map<number, string>;
  tracesCursorByPage: Map<number, string>;
}) {
  if (!args.pagination) return;

  if (args.pagination.logs?.hasNext && args.pagination.logs.nextCursor) {
    args.logsCursorByPage.set(args.page + 1, args.pagination.logs.nextCursor);
  } else {
    args.logsCursorByPage.delete(args.page + 1);
  }

  if (args.pagination.traces?.hasNext && args.pagination.traces.nextCursor) {
    args.tracesCursorByPage.set(args.page + 1, args.pagination.traces.nextCursor);
  } else {
    args.tracesCursorByPage.delete(args.page + 1);
  }
}
