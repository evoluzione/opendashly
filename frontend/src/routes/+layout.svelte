<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { get } from 'svelte/store';
  import { goto } from '$app/navigation';
  import { authState, loadSession } from '../lib/stores/auth';
  import AIAssistantWidget from '../components/AIAssistantWidget.svelte';

  export let params: Record<string, string> = {};

  const publicRoutes = ['/login'];

  onMount(() => {
    void loadSession();
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
</script>

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
      <span>Caricamento...</span>
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
  :global(*) {
    box-sizing: border-box;
  }
  
  :global(html, body) {
    margin: 0;
    padding: 0;
    width: 100%;
    overflow-x: hidden;
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  }
  
  .loading-screen {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
  }
  
  .loader {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 20px;
    color: #94a3b8;
    font-size: 14px;
  }
  
  .logo-pulse {
    width: 80px;
    height: 80px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    border-radius: 20px;
    color: white;
    animation: pulse-scale 1.5s ease-in-out infinite;
    box-shadow: 0 8px 32px rgba(99, 102, 241, 0.4);
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
    background: linear-gradient(135deg, #f0f4f8 0%, #e2e8f0 100%);
  }
</style>
