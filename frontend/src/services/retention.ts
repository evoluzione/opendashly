import { apiRequest } from './api';

export type SignalType = 'logs' | 'traces';

export type RetentionSetting = {
  signalType: SignalType;
  retentionDays: number;
};

export type RetentionSettings = {
  settings: RetentionSetting[];
};

export type CleanupRequest = {
  signalTypes: SignalType[];
  serviceName?: string;
};

export type CleanupResult = {
  jobId: string;
  signalType: SignalType;
  recordsDeleted: number;
};

export type CleanupResponse = {
  jobId: string;
  results: CleanupResult[];
};

export type CleanupJob = {
  jobId: string;
  jobType: string;
  signalType: string;
  serviceName: string;
  startedAt: string;
  completedAt: string | null;
  status: string;
  recordsDeleted: number;
  errorMessage: string;
};

export function getRetentionSettings(): Promise<RetentionSettings> {
  return apiRequest<RetentionSettings>('/api/admin/retention/settings');
}

export function updateRetentionSetting(signalType: SignalType, retentionDays: number): Promise<void> {
  return apiRequest<void>('/api/admin/retention/settings', {
    method: 'PUT',
    body: JSON.stringify({ signalType, retentionDays })
  });
}

export function executeCleanup(request: CleanupRequest): Promise<CleanupResponse> {
  return apiRequest<CleanupResponse>('/api/admin/retention/cleanup', {
    method: 'POST',
    body: JSON.stringify(request)
  });
}

export function listCleanupJobs(
  limit?: number,
  all?: boolean
): Promise<{ jobs: CleanupJob[]; total: number }> {
  const params = new URLSearchParams();
  if (all) {
    params.set('all', '1');
  }
  if (typeof limit === 'number') {
    params.set('limit', String(limit));
  }
  const suffix = params.toString() ? `?${params.toString()}` : '';
  return apiRequest<{ jobs: CleanupJob[]; total: number }>(
    '/api/admin/retention/jobs' + suffix
  );
}
