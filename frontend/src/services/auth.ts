import { apiRequest } from './api';

export type User = {
  id: string;
  username: string;
  role: 'admin' | 'user';
  isDisabled: boolean;
};

export type AuthSession = {
  user: User;
  mustChangePassword: boolean;
};

export function login(username: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password })
  });
}

export function logout(): Promise<void> {
  return apiRequest<void>('/api/auth/logout', { method: 'POST' });
}

export function changePassword(currentPassword: string, newPassword: string): Promise<AuthSession> {
  return apiRequest<AuthSession>('/api/auth/change-password', {
    method: 'POST',
    body: JSON.stringify({ currentPassword, newPassword })
  });
}

export function fetchSession(): Promise<AuthSession> {
  return apiRequest<AuthSession>('/api/auth/session');
}
