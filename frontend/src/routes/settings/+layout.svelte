<script lang="ts">
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import Sidebar from "../../components/Sidebar.svelte";
  import { authState } from "$lib/stores/auth";
  import { locale, t } from "$lib/i18n";

  function handleSelect(tab: string) {
    void goto(`/?tab=${tab}`);
  }

  const adminTabPaths = [
    "/settings/dashboard",
    "/settings/users",
    "/settings/retention",
    "/settings/status",
  ];

  $: isAdmin = $authState.user?.role === "admin";
  $: pathName = $page.url.pathname;
  $: isAdminTab = adminTabPaths.includes(pathName);

  type TabItem = {
    href: string;
    label: string;
  };

  $: tabs = [
    { href: "/settings", label: t($locale, "settings.generalTitle") },
    ...(isAdmin
      ? [
          { href: "/settings/dashboard", label: t($locale, "common.dashboard") },
          { href: "/settings/users", label: t($locale, "sidebar.userManagement") },
          { href: "/settings/retention", label: t($locale, "sidebar.retentionCleanup") },
          { href: "/settings/status", label: t($locale, "sidebar.systemMonitor") },
        ]
      : []),
  ] as TabItem[];
</script>

<div class="settings-shell">
  <Sidebar activeTab={null} onSelect={handleSelect} />
  <main class="settings-content">
    <div class="settings-inner">
      <nav class="settings-tabs" aria-label={t($locale, "settings.tabsAria")}>
        {#each tabs as tab}
          <a class="settings-tab" class:active={pathName === tab.href} href={tab.href}>{tab.label}</a>
        {/each}
      </nav>

      {#if isAdminTab && !isAdmin}
        <section class="settings-denied" aria-live="polite">
          <h2>{t($locale, "settings.adminOnlyTitle")}</h2>
          <p>{t($locale, "settings.adminOnlyDescription")}</p>
          <a href="/settings">{t($locale, "settings.backToGeneral")}</a>
        </section>
      {:else}
        <slot />
      {/if}
    </div>
  </main>
</div>

<style>
  .settings-shell {
    display: grid;
    grid-template-columns: 1fr;
    min-height: 100vh;
    width: 100%;
  }

  .settings-content {
    background: linear-gradient(135deg, var(--color-slate-50) 0%, var(--color-slate-75) 100%);
    padding: 28px;
    overflow-y: auto;
    padding-left: 240px;
  }

  .settings-inner {
    max-width: 1200px;
    margin: 0 auto;
  }

  .settings-tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    margin: 0 0 20px;
  }

  .settings-tab {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 8px 12px;
    border-radius: 10px;
    border: 1px solid rgba(148, 163, 184, 0.3);
    background: rgba(255, 255, 255, 0.75);
    color: var(--color-slate-700);
    text-decoration: none;
    font-size: 13px;
    font-weight: 600;
    transition: all 0.2s ease;
  }

  .settings-tab:hover {
    border-color: rgba(var(--rgb-primary-600), 0.45);
    color: var(--color-slate-900);
  }

  .settings-tab.active {
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
    border-color: transparent;
    color: var(--color-white);
    box-shadow: 0 10px 20px rgba(var(--rgb-primary-600), 0.24);
  }

  .settings-denied {
    background: var(--color-white);
    border: 1px solid var(--color-slate-200);
    border-radius: 14px;
    padding: 20px;
  }

  .settings-denied h2 {
    margin: 0 0 8px;
    font-size: 18px;
    color: var(--color-slate-950);
  }

  .settings-denied p {
    margin: 0 0 10px;
    color: var(--color-slate-500);
    font-size: 14px;
  }

  .settings-denied a {
    color: var(--color-primary-700);
    text-decoration: none;
    font-size: 14px;
    font-weight: 600;
  }

  @media (max-width: 900px) {
    .settings-content {
      padding-left: 200px;
    }
  }

  @media (max-width: 720px) {
    .settings-shell {
      grid-template-columns: 1fr;
    }

    .settings-content {
      padding: 20px;
      padding-left: 20px;
    }
  }
</style>
