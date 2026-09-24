import { apiRequest } from './api';
import type { AssistantLink } from './assistantLinks';

export type { AssistantLink };

// AssistantContext is opaque to the UI: the backend returns it with each answer
// and gets it back with the next message, so follow-ups like "il primo" or
// "e ieri?" refer to the previous answer and pending questions get completed.
export type AssistantContext = Record<string, unknown>;

export type AssistantMessage = {
  role: 'user' | 'assistant';
  content: string;
  context?: AssistantContext;
  suggestions?: AssistantSuggestion[];
  links?: AssistantLink[];
  rows?: AssistantRow[];
};

// kind places the button: "inline" acts on the answer and sits inside it,
// "rerun" repeats the analysis elsewhere (shown after "Ripeti per:"), the
// others are plain next questions.
export type AssistantSuggestion = {
  label: string;
  prompt: string;
  kind?: 'inline' | 'rerun';
};

// AssistantRow holds the actions of one numbered item of an answer.
export type AssistantRow = {
  prompt: string;
  link?: AssistantLink;
};

export type AssistantChatResponse = {
  answer: string;
  steps?: string[];
  suggestions?: AssistantSuggestion[];
  context?: AssistantContext;
  lang?: 'it' | 'en';
  links?: AssistantLink[];
  rows?: AssistantRow[];
};

export function sendAssistantMessage(payload: {
  prompt: string;
  locale: string;
  context?: AssistantContext;
}): Promise<AssistantChatResponse> {
  // "oggi" / "ieri" start at the viewer's midnight.
  const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  return apiRequest<AssistantChatResponse>('/api/ai/assistant/chat', {
    method: 'POST',
    body: JSON.stringify({ ...payload, timeZone }),
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
