import { writable } from 'svelte/store';
import type { AuthSession, User } from '../../services/auth';
import { changePassword, fetchSession, login, logout } from '../../services/auth';

type AuthState = {
  user: User | null;
  mustChangePassword: boolean;
  loading: boolean;
  error: string | null;
};

const initial: AuthState = {
  user: null,
  mustChangePassword: false,
  loading: false,
  error: null
};

export const authState = writable<AuthState>(initial);

export async function loadSession() {
  authState.update((state) => ({ ...state, loading: true, error: null }));
  try {
    const session = await fetchSession();
    authState.set({ user: session.user, mustChangePassword: session.mustChangePassword, loading: false, error: null });
  } catch (err) {
    authState.set({ user: null, mustChangePassword: false, loading: false, error: null });
  }
}

export async function loginUser(username: string, password: string): Promise<AuthSession | null> {
  authState.update((state) => ({ ...state, loading: true, error: null }));
  try {
    const session = await login(username, password);
    authState.set({ user: session.user, mustChangePassword: session.mustChangePassword, loading: false, error: null });
    return session;
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Login failed';
    authState.set({ user: null, mustChangePassword: false, loading: false, error: message });
    return null;
  }
}

export async function changePasswordForUser(currentPassword: string, newPassword: string): Promise<AuthSession | null> {
  authState.update((state) => ({ ...state, loading: true, error: null }));
  try {
    const session = await changePassword(currentPassword, newPassword);
    authState.set({ user: session.user, mustChangePassword: session.mustChangePassword, loading: false, error: null });
    return session;
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Change password failed';
    authState.update((state) => ({ ...state, loading: false, error: message }));
    return null;
  }
}

export async function logoutUser() {
  await logout();
  authState.set(initial);
}
