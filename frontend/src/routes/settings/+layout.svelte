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
    "/settings/ai",
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
          { href: "/settings/ai", label: t($locale, "sidebar.aiSettings") },
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
    background: linear-gradient(135deg, #f8fafc 0%, #eef2f7 100%);
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
    color: #334155;
    text-decoration: none;
    font-size: 13px;
    font-weight: 600;
    transition: all 0.2s ease;
  }

  .settings-tab:hover {
    border-color: rgba(99, 102, 241, 0.45);
    color: #1e293b;
  }

  .settings-tab.active {
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    border-color: transparent;
    color: #ffffff;
    box-shadow: 0 10px 20px rgba(99, 102, 241, 0.24);
  }

  .settings-denied {
    background: #fff;
    border: 1px solid #e2e8f0;
    border-radius: 14px;
    padding: 20px;
  }

  .settings-denied h2 {
    margin: 0 0 8px;
    font-size: 18px;
    color: #0f172a;
  }

  .settings-denied p {
    margin: 0 0 10px;
    color: #64748b;
    font-size: 14px;
  }

  .settings-denied a {
    color: #4f46e5;
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
