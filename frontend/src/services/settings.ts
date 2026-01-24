import { apiRequest } from './api';

export type AISettings = {
    tenantId: string;
    enabled: boolean;
    provider: string;
    model: string;
    apiKey?: string; // masked on get
    updatedAt: string;
};

export type UpdateAISettingsRequest = {
    enabled: boolean;
    provider: string;
    model: string;
    apiKey: string;
};

export function getAISettings(): Promise<AISettings> {
    return apiRequest<AISettings>('/api/admin/ai/settings');
}

export function updateAISettings(settings: UpdateAISettingsRequest): Promise<AISettings> {
    return apiRequest<AISettings>('/api/admin/ai/settings', {
        method: 'PUT',
        body: JSON.stringify(settings)
    });
}
