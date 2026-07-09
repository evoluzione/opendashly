import type { QueryRequest, QueryRunResult } from '../../services/query';

type QuerySignal = 'logs' | 'traces';

export type QueryRefreshSnapshot = {
  lastRequest: QueryRequest | null;
  autoRefreshSeconds: number | null;
  autoRefreshRangeMinutes: number | null;
  result: QueryRunResult | null;
};

function logKey(entry: any) {
  const timestamp = entry?.timestamp ?? '';
  const traceId = entry?.traceId ?? '';
  const spanId = entry?.spanId ?? '';
  const body =
    typeof entry?.body === 'string' ? entry.body : JSON.stringify(entry?.body ?? '');
  return `${timestamp}|${traceId}|${spanId}|${body}`;
}

function traceKey(entry: any) {
  return entry?.traceId ?? '';
}

function mergeByKey<T>(
  latest: T[],
  previous: T[],
  keyFn: (item: T) => string,
  limit?: number
) {
  const seen = new Set<string>();
  const merged: T[] = [];
  for (const item of latest) {
    const key = keyFn(item);
    if (seen.has(key)) continue;
    seen.add(key);
    merged.push(item);
  }
  for (const item of previous) {
    const key = keyFn(item);
    if (seen.has(key)) continue;
    seen.add(key);
    merged.push(item);
  }
  return typeof limit === 'number' && limit > 0 ? merged.slice(0, limit) : merged;
}

export function mergeResult(
  previous: QueryRunResult,
  latest: QueryRunResult,
  request: QueryRequest
): QueryRunResult {
  const logLimit = request.limit ?? latest.pagination?.logs?.limit;
  const traceLimit = request.limit ?? latest.pagination?.traces?.limit;
  return {
    ...latest,
    results: {
      ...latest.results,
      logs: mergeByKey(latest.results.logs ?? [], previous.results.logs ?? [], logKey, logLimit),
      traces: mergeByKey(
        latest.results.traces ?? [],
        previous.results.traces ?? [],
        traceKey,
        traceLimit
      )
    }
  };
}

export function replaceSingleSignalResult(
  previous: QueryRunResult,
  latest: QueryRunResult,
  signal: QuerySignal
): QueryRunResult {
  const merged: QueryRunResult = {
    ...previous,
    runId: latest.runId,
    status: latest.status,
    summary: { ...previous.summary },
    pagination: { ...previous.pagination },
    results: { ...previous.results },
    signalErrors: latest.signalErrors
  };

  if (signal === 'logs') {
    merged.results.logs = latest.results.logs ?? [];
    if (latest.pagination?.logs) {
      merged.pagination = { ...merged.pagination, logs: latest.pagination.logs };
    }
    merged.summary.logCount = latest.summary?.logCount ?? merged.results.logs.length;
    return merged;
  }

  if (signal === 'traces') {
    merged.results.traces = latest.results.traces ?? [];
    if (latest.pagination?.traces) {
      merged.pagination = { ...merged.pagination, traces: latest.pagination.traces };
    }
    merged.summary.traceCount = latest.summary?.traceCount ?? merged.results.traces.length;
    return merged;
  }

  return merged;
}

export function mergeSingleSignalResult(
  previous: QueryRunResult,
  latest: QueryRunResult,
  signal: QuerySignal,
  request: QueryRequest
): QueryRunResult {
  const merged: QueryRunResult = {
    ...previous,
    runId: latest.runId,
    status: latest.status,
    summary: { ...previous.summary },
    pagination: { ...previous.pagination },
    results: { ...previous.results }
  };

  if (signal === 'logs') {
    const logLimit = request.limit ?? latest.pagination?.logs?.limit;
    const isFirstPage = request.page === undefined || request.page === 1;
    merged.results.logs = isFirstPage
      ? mergeByKey(latest.results.logs ?? [], previous.results.logs ?? [], logKey, logLimit)
      : (latest.results.logs ?? []);
    if (latest.pagination?.logs) {
      merged.pagination = { ...merged.pagination, logs: latest.pagination.logs };
    }
    merged.summary.logCount = latest.summary?.logCount ?? merged.results.logs.length;
    return merged;
  }

  if (signal === 'traces') {
    const traceLimit = request.limit ?? latest.pagination?.traces?.limit;
    const isFirstPage = request.page === undefined || request.page === 1;
    merged.results.traces = isFirstPage
      ? mergeByKey(latest.results.traces ?? [], previous.results.traces ?? [], traceKey, traceLimit)
      : (latest.results.traces ?? []);
    if (latest.pagination?.traces) {
      merged.pagination = { ...merged.pagination, traces: latest.pagination.traces };
    }
    merged.summary.traceCount = latest.summary?.traceCount ?? merged.results.traces.length;
    return merged;
  }
  return merged;
}

export function buildRollingRangeRequest(request: QueryRequest, minutes: number): QueryRequest {
  const now = new Date();
  const from = new Date(now.getTime() - minutes * 60 * 1000);
  return {
    ...request,
    timeRange: { from: from.toISOString(), to: now.toISOString() }
  };
}

export function getMaxTimestamp(result: QueryRunResult | null): string | null {
  if (!result || !result.results) return null;
  let maxTime = '';

  if (result.results.logs) {
    for (const log of result.results.logs) {
      if (log.timestamp > maxTime) maxTime = log.timestamp;
    }
  }

  if (result.results.traces) {
    for (const trace of result.results.traces) {
      if (trace.timestamp > maxTime) maxTime = trace.timestamp;
    }
  }

  return maxTime || null;
}

export function buildAutoRefreshRequest(snapshot: QueryRefreshSnapshot): QueryRequest | null {
  if (!snapshot.lastRequest || !snapshot.autoRefreshSeconds) {
    return null;
  }

  if (snapshot.autoRefreshSeconds === 1) {
    const maxTimestamp = getMaxTimestamp(snapshot.result);
    if (maxTimestamp) {
      const fromDate = new Date(new Date(maxTimestamp).getTime() + 1);
      return {
        ...snapshot.lastRequest,
        timeRange: {
          from: fromDate.toISOString(),
          to: new Date().toISOString()
        }
      };
    }
    if (snapshot.autoRefreshRangeMinutes) {
      return buildRollingRangeRequest(snapshot.lastRequest, snapshot.autoRefreshRangeMinutes);
    }
    return snapshot.lastRequest;
  }

  if (snapshot.autoRefreshRangeMinutes) {
    return buildRollingRangeRequest(snapshot.lastRequest, snapshot.autoRefreshRangeMinutes);
  }

  return snapshot.lastRequest;
}
