import { apiRequest } from './api';
import {
  buildChangePasswordPayload,
  buildFirstLoginPasswordPayload,
  buildLoginPayload
} from './auth.mapper';

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
    body: JSON.stringify(buildLoginPayload(username, password))
  });
}

export function logout(): Promise<void> {
  return apiRequest<void>('/api/auth/logout', { method: 'POST' });
}

export function changePassword(newPassword: string, currentPassword?: string): Promise<AuthSession> {
  const body = buildChangePasswordPayload(newPassword, currentPassword);
  return apiRequest<AuthSession>('/api/auth/change-password', {
    method: 'POST',
    body: JSON.stringify(body)
  });
}

export function firstLoginChangePassword(newPassword: string): Promise<AuthSession> {
  return apiRequest<AuthSession>('/api/auth/first-login-change-password', {
    method: 'POST',
    body: JSON.stringify(buildFirstLoginPasswordPayload(newPassword))
  });
}

export function fetchSession(): Promise<AuthSession> {
  return apiRequest<AuthSession>('/api/auth/session');
}
