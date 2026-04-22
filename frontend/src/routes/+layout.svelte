<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { get } from 'svelte/store';
  import { goto } from '$app/navigation';
  import { authState, loadSession } from '../lib/stores/auth';
  import { workspaceState, loadWorkspace } from '../lib/stores/workspace';
  import { initializeLocale, locale, t } from '$lib/i18n';
  import AIAssistantWidget from '../components/AIAssistantWidget.svelte';

  const publicRoutes = ['/login'];

  onMount(() => {
    initializeLocale();
    void loadSession();
    void loadWorkspace();
  });

  onMount(() => {
    const unsubscribe = authState.subscribe((state) => {
      if (state.loading) {
        return;
      }
      const path = get(page).url.pathname;
      if (!state.user && !publicRoutes.includes(path) && path !== '/first-login') {
        void goto('/login');
      } else if (state.user && state.mustChangePassword && path !== '/first-login') {
        void goto('/first-login');
      }
    });
    return () => unsubscribe();
  });

  $: currentPath = $page.url.pathname;
  $: isPublicRoute = publicRoutes.includes(currentPath) || currentPath === '/first-login';
  $: showContent = !$authState.loading && ($authState.user || isPublicRoute);

  function sectionLabel(path: string, loc: typeof $locale): string {
    if (path.startsWith('/settings')) return t(loc, 'settings.title');
    if (path.startsWith('/admin')) return t(loc, 'sidebar.administration');
    if (path.startsWith('/query')) return t(loc, 'sidebar.logs');
    if (path.startsWith('/traces')) return t(loc, 'sidebar.traces');
    if (path.startsWith('/assistant-ai')) return t(loc, 'assistant.title');
    if (path === '/') return t(loc, 'common.dashboard');
    return '';
  }

  $: section = sectionLabel(currentPath, $locale);
  $: pageTitle = section ? `${$workspaceState.title} – ${section}` : $workspaceState.title;
</script>

<svelte:head>
  <title>{pageTitle}</title>
</svelte:head>

{#if $authState.loading && !isPublicRoute}
  <div class="loading-screen">
    <div class="loader">
      <div class="logo-pulse">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 2L2 7l10 5 10-5-10-5z"/>
          <path d="M2 17l10 5 10-5"/>
          <path d="M2 12l10 5 10-5"/>
        </svg>
      </div>
      <span>{t($locale, 'layout.loading')}</span>
    </div>
  </div>
{:else if showContent || isPublicRoute}
  <div class="app-shell">
    <main class:unauth={!$authState.user}>
      <slot />
    </main>
    {#if $authState.user && !isPublicRoute}
      <AIAssistantWidget />
    {/if}
  </div>
{/if}

<style>
  .loading-screen {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--gradient-app-bg);
  }
  
  .loader {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 20px;
    color: var(--color-slate-400);
    font-size: 14px;
  }
  
  .logo-pulse {
    width: 80px;
    height: 80px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--gradient-primary);
    border-radius: 20px;
    color: var(--color-white);
    animation: pulse-scale 1.5s ease-in-out infinite;
    box-shadow: var(--shadow-elevated-primary);
  }
  
  @keyframes pulse-scale {
    0%, 100% { transform: scale(1); opacity: 1; }
    50% { transform: scale(0.95); opacity: 0.8; }
  }
  
  .app-shell {
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    min-height: 100vh;
    width: 100%;
  }
  
  main {
    min-height: 100vh;
    width: 100%;
  }
  
  main.unauth {
    background: var(--gradient-surface-soft);
  }
</style>
