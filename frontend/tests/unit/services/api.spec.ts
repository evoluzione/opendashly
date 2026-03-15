import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError, apiRequest } from '../../../src/services/api';

describe('apiRequest', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('ritorna payload json su risposta ok', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ ok: true }), { status: 200 }))
    );

    const result = await apiRequest<{ ok: boolean }>('/api/test');

    expect(result).toEqual({ ok: true });
  });

  it('solleva ApiError con messaggio dal body json', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: 'boom' }), {
          status: 500,
          headers: { 'Content-Type': 'application/json' }
        })
      )
    );

    await expect(apiRequest('/api/fail')).rejects.toMatchObject<ApiError>({
      name: 'ApiError',
      message: 'boom',
      status: 500,
      path: '/api/fail',
      method: 'GET'
    });
  });

  it('ritenta richieste idempotenti su stato retriable', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response('temp', { status: 503 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ done: true }), { status: 200 }));

    vi.stubGlobal('fetch', fetchMock);

    const result = await apiRequest<{ done: boolean }>('/api/retry', {
      retries: 1,
      retryDelayMs: 1
    });

    expect(result).toEqual({ done: true });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('non ritenta richieste POST su stato retriable', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('temp', { status: 503 }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(
      apiRequest('/api/retry-post', {
        method: 'POST',
        retries: 2,
        retryDelayMs: 1
      })
    ).rejects.toMatchObject({ name: 'ApiError', status: 503 });

    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
