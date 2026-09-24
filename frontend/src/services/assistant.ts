import { apiRequest } from './api';

// AssistantContext is opaque to the UI: the backend returns it with each answer
// and gets it back with the next message, so follow-ups like "il primo" or
// "e ieri?" refer to the previous answer and pending questions get completed.
export type AssistantContext = Record<string, unknown>;

export type AssistantMessage = {
  role: 'user' | 'assistant';
  content: string;
  context?: AssistantContext;
  suggestions?: AssistantSuggestion[];
};

export type AssistantSuggestion = {
  label: string;
  prompt: string;
};

export type AssistantChatResponse = {
  answer: string;
  steps?: string[];
  suggestions?: AssistantSuggestion[];
  context?: AssistantContext;
  lang?: 'it' | 'en';
};

export function sendAssistantMessage(payload: {
  prompt: string;
  locale: string;
  context?: AssistantContext;
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
