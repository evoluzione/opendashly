import { apiRequest } from './api';
import type { QueryRequest } from './query';

export type SavedQuery = {
  id: string;
  name: string;
  description: string;
  request: QueryRequest;
  createdAt: string;
};

export function listSavedQueries(): Promise<{ items: SavedQuery[] }> {
  return apiRequest<{ items: SavedQuery[] }>('/api/queries');
}

export function saveQuery(payload: { name: string; description: string; request: QueryRequest }): Promise<SavedQuery> {
  return apiRequest<SavedQuery>('/api/queries', {
    method: 'POST',
    body: JSON.stringify(payload)
  });
}

export function deleteQuery(id: string): Promise<void> {
  return apiRequest<void>(`/api/queries/${id}`, { method: 'DELETE' });
}

const SAVED_QUERY_RUN_TIMEOUT_MS = 40000;

export function runSavedQuery(id: string) {
  return apiRequest(`/api/queries/${id}/run`, {
    method: 'POST',
    timeoutMs: SAVED_QUERY_RUN_TIMEOUT_MS
  });
}
