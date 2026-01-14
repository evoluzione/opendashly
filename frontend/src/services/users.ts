import { apiRequest } from './api';
import type { User } from './auth';

export type NewUser = {
  username: string;
  password: string;
  role: 'admin' | 'user';
};

export function listUsers(): Promise<User[]> {
  return apiRequest<User[]>('/api/users');
}

export function createUser(payload: NewUser): Promise<User> {
  return apiRequest<User>('/api/users', {
    method: 'POST',
    body: JSON.stringify(payload)
  });
}
