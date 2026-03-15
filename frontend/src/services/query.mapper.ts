export function toStringList(payload: unknown): string[] {
  if (!Array.isArray(payload)) {
    return [];
  }
  return payload.filter((item): item is string => typeof item === 'string');
}

export function buildAttributesQuery(search: string): string {
  const params = new URLSearchParams({ q: search });
  return params.toString();
}
