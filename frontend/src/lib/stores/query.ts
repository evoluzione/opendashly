import { writable } from 'svelte/store';
import type { QueryRunResult, QueryRequest } from '../../services/query';
import { runQuery } from '../../services/query';

type QueryState = {
  loading: boolean;
  error: string | null;
  result: QueryRunResult | null;
};

const initial: QueryState = { loading: false, error: null, result: null };

export const queryState = writable<QueryState>(initial);

export async function executeQuery(request: QueryRequest) {
  queryState.set({ loading: true, error: null, result: null });
  try {
    const result = await runQuery(request);
    queryState.set({ loading: false, error: null, result });
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    queryState.set({ loading: false, error: message, result: null });
  }
}
