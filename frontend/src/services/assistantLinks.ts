import { runQuery, type FilterItem } from './query';
import { fetchTraceSpans } from './traces';

// AssistantLink points at the data behind an assistant answer. The search page
// URL and the downloads are built here, in the viewer's time zone.
export type AssistantLink = {
  label: string;
  kind: 'logs' | 'traces' | 'trace';
  service?: string;
  from?: string;
  to?: string;
  severity?: string;
  errorsOnly?: boolean;
  search?: string;
  traceId?: string;
};

export const DOWNLOAD_LIMIT = 1000;

// Relative presets of the search page: used when a link covers "the last N"
// up to now, so the link stays meaningful when opened later.
const PRESETS: Array<[string, number]> = [
  ['5m', 5],
  ['15m', 15],
  ['30m', 30],
  ['1h', 60],
  ['6h', 360],
  ['24h', 1440],
  ['7d', 10080],
];

function localInputValue(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

// linkUrl returns the search page URL with the link's filters preselected.
export function linkUrl(link: AssistantLink, now: Date = new Date()): string {
  const params = new URLSearchParams();
  if (link.kind === 'trace') {
    params.set('traceId', link.traceId ?? '');
    params.set('tab', 'tracce');
  } else {
    params.set('tab', link.kind === 'logs' ? 'logs' : 'tracce');
    if (link.service) params.set('service', link.service);
    if (link.severity) params.set('severity', link.severity);
    if (link.errorsOnly) params.set('errors', 'with_errors');
    if (link.search) params.set('search', link.search);
    const from = link.from ? new Date(link.from) : null;
    const to = link.to ? new Date(link.to) : null;
    if (from && to && !Number.isNaN(from.getTime()) && !Number.isNaN(to.getTime())) {
      const minutes = Math.round((to.getTime() - from.getTime()) / 60000);
      const recent = Math.abs(now.getTime() - to.getTime()) < 2 * 60000;
      const preset = PRESETS.find(([, m]) => m === minutes);
      if (recent && preset) {
        params.set('range', preset[0]);
      } else {
        params.set('range', 'custom');
        params.set('from', localInputValue(from));
        params.set('to', localInputValue(to));
      }
    }
  }
  params.set('mode', 'manual');
  params.set('autorun', '1');
  return `/?${params.toString()}`;
}

// linkFilters mirrors what the search page sends for the same URL.
export function linkFilters(link: AssistantLink): FilterItem[] {
  const filters: FilterItem[] = [];
  const add = (key: string, operator: string, value: string) =>
    filters.push({ connector: 'AND', key, operator, value });
  if (link.service) add('service.name', '=', link.service);
  if (link.kind === 'logs') {
    if (link.severity) add('severity', '=', link.severity);
    if (link.search) add('body', 'contains', link.search);
  } else {
    if (link.errorsOnly) add('trace_error_scope', '=', 'with_errors');
    if (link.search) add('trace_or_span', 'contains', link.search);
  }
  return filters;
}

function csvCell(value: unknown): string {
  const text = value === null || value === undefined ? '' : typeof value === 'object' ? JSON.stringify(value) : String(value);
  return /[",\n\r]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text;
}

export function toCsv(columns: string[], rows: Array<Record<string, unknown>>): string {
  const lines = [columns.join(',')];
  for (const row of rows) {
    lines.push(columns.map((c) => csvCell(row[c])).join(','));
  }
  return lines.join('\n') + '\n';
}

export function logRows(logs: any[]): Array<Record<string, unknown>> {
  return logs.map((l) => ({
    timestamp: l.timestamp,
    severity: l.severity,
    service: l.resourceAttributes?.['service.name'] ?? '',
    traceId: l.traceId ?? '',
    spanId: l.spanId ?? '',
    body: l.body,
  }));
}

export const LOG_COLUMNS = ['timestamp', 'severity', 'service', 'traceId', 'spanId', 'body'];
export const TRACE_COLUMNS = ['traceId', 'name', 'service', 'spanCount', 'errorCount', 'durationMs', 'lastSeen'];

function saveFile(content: string, filename: string, type: string) {
  const blob = new Blob([content], { type });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

function fileStamp(link: AssistantLink): string {
  const parts = [link.kind, link.service, link.from?.slice(0, 16)].filter(Boolean) as string[];
  return parts.join('-').replace(/[^a-zA-Z0-9-]+/g, '-');
}

// downloadLink saves the link's data: a CSV of up to DOWNLOAD_LIMIT logs or
// traces, or the spans of a single trace as JSON. It returns the row count.
export async function downloadLink(link: AssistantLink): Promise<number> {
  if (link.kind === 'trace') {
    const spans = await fetchTraceSpans(link.traceId ?? '');
    saveFile(JSON.stringify(spans, null, 2), `trace-${link.traceId}.json`, 'application/json');
    return spans.length;
  }
  const signal = link.kind === 'logs' ? 'logs' : 'traces';
  const result = await runQuery({
    signals: [signal],
    timeRange: { from: link.from ?? new Date(0).toISOString(), to: link.to ?? new Date().toISOString() },
    filters: {},
    filterList: linkFilters(link),
    page: 1,
    limit: DOWNLOAD_LIMIT,
  });
  const csv =
    signal === 'logs'
      ? toCsv(LOG_COLUMNS, logRows(result.results.logs ?? []))
      : toCsv(TRACE_COLUMNS, result.results.traces ?? []);
  saveFile(csv, `${fileStamp(link)}.csv`, 'text/csv;charset=utf-8');
  return signal === 'logs' ? (result.results.logs ?? []).length : (result.results.traces ?? []).length;
}
