import { writable } from 'svelte/store';
import { getWorkspaceSettings, updateWorkspaceSettings } from '../../services/workspace';

export const DEFAULT_TITLE = 'OpenDashly';

type WorkspaceState = {
    title: string;
    loading: boolean;
    error: string | null;
};

const initial: WorkspaceState = {
    title: DEFAULT_TITLE,
    loading: false,
    error: null
};

export const workspaceState = writable<WorkspaceState>(initial);

export async function loadWorkspace(): Promise<void> {
    workspaceState.update((s) => ({ ...s, loading: true, error: null }));
    try {
        const settings = await getWorkspaceSettings();
        workspaceState.update((s) => ({ ...s, title: settings.title, loading: false }));
    } catch {
        workspaceState.update((s) => ({ ...s, loading: false }));
    }
}

export async function saveWorkspaceTitle(title: string): Promise<boolean> {
    workspaceState.update((s) => ({ ...s, loading: true, error: null }));
    try {
        const settings = await updateWorkspaceSettings(title);
        workspaceState.update((s) => ({ ...s, title: settings.title, loading: false }));
        return true;
    } catch (err) {
        const message = err instanceof Error ? err.message : 'Errore durante il salvataggio';
        workspaceState.update((s) => ({ ...s, loading: false, error: message }));
        return false;
    }
}
