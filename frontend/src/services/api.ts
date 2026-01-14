const baseUrl = import.meta.env.VITE_API_BASE ?? '';

export async function apiRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const url = `${baseUrl}${path}`;
  const method = options.method ?? 'GET';
  const startedAt = performance.now();
  console.debug('api.request', { url, method });
  let response: Response;
  try {
    response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...(options.headers || {})
      },
      credentials: 'include',
      ...options
    });
  } catch (error) {
    const durationMs = Math.round(performance.now() - startedAt);
    console.debug('api.request.failed', { url, method, durationMs, error });
    throw error;
  }
  const durationMs = Math.round(performance.now() - startedAt);
  console.debug('api.response', { url, method, status: response.status, ok: response.ok, durationMs });

  if (!response.ok) {
    let bodyText = '';
    try {
      bodyText = await response.text();
    } catch {
      bodyText = '';
    }
    if (bodyText) {
      console.debug('api.response.body', { url, status: response.status, bodyText });
    }
    throw new Error(`Request failed: ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }
  const contentLength = response.headers.get('content-length');
  if (contentLength === '0') {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}
