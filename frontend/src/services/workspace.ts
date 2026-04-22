import { apiRequest } from './api';

export type WorkspaceSettings = {
    title: string;
};

export function getWorkspaceSettings(): Promise<WorkspaceSettings> {
    return apiRequest<WorkspaceSettings>('/api/workspace/settings');
}

export function updateWorkspaceSettings(title: string): Promise<WorkspaceSettings> {
    return apiRequest<WorkspaceSettings>('/api/admin/workspace/settings', {
        method: 'PUT',
        body: JSON.stringify({ title })
    });
}
