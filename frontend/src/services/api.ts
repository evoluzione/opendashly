const envBaseUrl = import.meta.env.VITE_API_BASE ?? '';
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

export async function apiRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const url = `${baseUrl}${path}`;
  const method = options.method ?? 'GET';
  const startedAt = performance.now();
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
    throw error;
  }
  const durationMs = Math.round(performance.now() - startedAt);

  if (!response.ok) {
    let bodyText = '';
    try {
      bodyText = await response.text();
    } catch {
      bodyText = '';
    }
    let errorMessage = `Request failed: ${response.status}`;
    if (bodyText) {
      try {
        const bodyJson = JSON.parse(bodyText);
        if (bodyJson && (bodyJson.error || bodyJson.message)) {
          errorMessage = bodyJson.error || bodyJson.message;
        }
      } catch {
        // Not a JSON body or parse failed, stick to default
      }
    }
    throw new Error(errorMessage);
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
