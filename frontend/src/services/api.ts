const envBaseUrl = import.meta.env.VITE_API_BASE ?? '';
const envTimeoutMs = Number(import.meta.env.VITE_API_TIMEOUT_MS ?? '10000');

export class ApiError extends Error {
  status: number;
  path: string;
  method: string;

  constructor(message: string, status: number, path: string, method: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.path = path;
    this.method = method;
  }
}

export type ApiRequestOptions = RequestInit & {
  timeoutMs?: number;
  retries?: number;
  retryDelayMs?: number;
  retryOnStatuses?: number[];
};

const DEFAULT_TIMEOUT_MS = Number.isFinite(envTimeoutMs) && envTimeoutMs > 0 ? envTimeoutMs : 10000;
const DEFAULT_RETRY_STATUSES = [502, 503, 504];

const baseUrl = (() => {
  if (envBaseUrl && envBaseUrl !== 'http://localhost:8080') {
    return envBaseUrl;
  }
  if (typeof window === 'undefined') {
    return envBaseUrl;
  }
  const host = window.location.hostname;
  const protocol = window.location.protocol;
  return `${protocol}//${host}:8080`;
})();

function isIdempotentMethod(method: string): boolean {
  return method === 'GET' || method === 'HEAD' || method === 'OPTIONS';
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function mergeAbortSignals(
  outerSignal: AbortSignal | undefined,
  timeoutSignal: AbortSignal
): AbortSignal {
  if (!outerSignal) return timeoutSignal;
  const controller = new AbortController();
  const onAbort = () => controller.abort();
  outerSignal.addEventListener('abort', onAbort, { once: true });
  timeoutSignal.addEventListener('abort', onAbort, { once: true });
  return controller.signal;
}

async function fetchWithTimeout(url: string, options: RequestInit, timeoutMs: number): Promise<Response> {
  const timeoutController = new AbortController();
  const timer = setTimeout(() => timeoutController.abort(), timeoutMs);
  try {
    const signal = mergeAbortSignals(options.signal, timeoutController.signal);
    return await fetch(url, {
      ...options,
      signal
    });
  } finally {
    clearTimeout(timer);
  }
}

async function toApiError(response: Response, path: string, method: string): Promise<ApiError> {
  let bodyText = '';
  try {
    bodyText = await response.text();
  } catch {
    bodyText = '';
  }

  let message = `Request failed: ${response.status}`;
  if (bodyText) {
    try {
      const bodyJson = JSON.parse(bodyText);
      if (bodyJson && typeof bodyJson === 'object') {
        const error = (bodyJson as Record<string, unknown>).error;
        const fallback = (bodyJson as Record<string, unknown>).message;
        if (typeof error === 'string' && error.length > 0) {
          message = error;
        } else if (typeof fallback === 'string' && fallback.length > 0) {
          message = fallback;
        }
      }
    } catch {
      const plain = bodyText.trim();
      if (plain) {
        message = plain;
      }
    }
  }

  return new ApiError(message, response.status, path, method);
}

function shouldRetryRequest(
  method: string,
  attempt: number,
  retries: number,
  retryOnStatuses: number[],
  error?: unknown,
  status?: number
): boolean {
  if (attempt >= retries || !isIdempotentMethod(method)) {
    return false;
  }
  if (typeof status === 'number') {
    return retryOnStatuses.includes(status);
  }
  return error instanceof TypeError;
}

export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  const url = `${baseUrl}${path}`;
  const method = (options.method ?? 'GET').toUpperCase();
  const retries = options.retries ?? 0;
  const retryDelayMs = options.retryDelayMs ?? 250;
  const retryOnStatuses = options.retryOnStatuses ?? DEFAULT_RETRY_STATUSES;
  const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;

  const { retries: _r, retryDelayMs: _d, retryOnStatuses: _s, timeoutMs: _t, ...fetchOptions } = options;

  let attempt = 0;
  while (true) {
    try {
      const response = await fetchWithTimeout(
        url,
        {
          headers: {
            'Content-Type': 'application/json',
            ...(fetchOptions.headers || {})
          },
          credentials: 'include',
          ...fetchOptions,
          method
        },
        timeoutMs
      );

      if (!response.ok) {
        const apiError = await toApiError(response, path, method);
        if (shouldRetryRequest(method, attempt, retries, retryOnStatuses, undefined, response.status)) {
          attempt += 1;
          await delay(retryDelayMs);
          continue;
        }
        throw apiError;
      }

      if (response.status === 204) {
        return undefined as T;
      }
      const contentLength = response.headers.get('content-length');
      if (contentLength === '0') {
        return undefined as T;
      }
      return response.json() as Promise<T>;
    } catch (error) {
      const isTimeoutAbort = error instanceof DOMException && error.name === 'AbortError';
      if (isTimeoutAbort) {
        const timeoutError = new Error(`Request timed out after ${timeoutMs}ms`);
        if (shouldRetryRequest(method, attempt, retries, retryOnStatuses, timeoutError)) {
          attempt += 1;
          await delay(retryDelayMs);
          continue;
        }
        throw timeoutError;
      }
      if (shouldRetryRequest(method, attempt, retries, retryOnStatuses, error)) {
        attempt += 1;
        await delay(retryDelayMs);
        continue;
      }
      throw error;
    }
  }
}
