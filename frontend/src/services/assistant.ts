import { apiRequest } from './api';

export type AssistantMessage = {
  role: 'user' | 'assistant';
  content: string;
};

export type AssistantChatResponse = {
  answer: string;
  steps?: string[];
};

export function getAIAssistantAvailability(): Promise<{ enabled: boolean }> {
  return apiRequest<{ enabled: boolean }>('/api/ai/availability');
}

export function sendAssistantMessage(payload: {
  prompt: string;
  messages: AssistantMessage[];
}): Promise<AssistantChatResponse> {
  return apiRequest<AssistantChatResponse>('/api/ai/assistant/chat', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export function getAssistantSession(): Promise<{ messages: AssistantMessage[] }> {
  return apiRequest<{ messages: AssistantMessage[] }>('/api/ai/assistant/session');
}

export function saveAssistantSession(payload: { messages: AssistantMessage[] }): Promise<void> {
  return apiRequest<void>('/api/ai/assistant/session', {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export function resetAssistantSession(): Promise<void> {
  return apiRequest<void>('/api/ai/assistant/session', {
    method: 'DELETE',
  });
}
