<script lang="ts">
  import { authState, logoutUser } from "../lib/stores/auth";
  import { locale, t } from "$lib/i18n";

  export let activeTab: "logs" | "metriche" | "tracce" | null = null;
  export let onSelect: (tab: "logs" | "metriche" | "tracce") => void;

  $: roleLabel =
    $authState.user?.role === "admin"
      ? t($locale, "common.roleAdmin")
      : t($locale, "common.roleUser");
</script>

<aside class="sidebar">
  <div class="brand">
    <div class="logo">
      <img src="/sidebar-mark.svg" alt="OpenDashly" />
    </div>
    <div class="brand-text">
      <span class="name">Opendashly</span>
      <span class="tagline">{t($locale, "common.dashboard")}</span>
    </div>
  </div>

  <div class="nav-section">
    <span class="nav-label">{t($locale, "sidebar.telemetry")}</span>
    <nav>
      <button
        class:selected={activeTab === "metriche"}
        on:click={() => onSelect("metriche")}
      >
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <line x1="18" y1="20" x2="18" y2="10" />
          <line x1="12" y1="20" x2="12" y2="4" />
          <line x1="6" y1="20" x2="6" y2="14" />
        </svg>
        <span>{t($locale, "sidebar.metrics")}</span>
      </button>
      <button
        class:selected={activeTab === "logs"}
        on:click={() => onSelect("logs")}
      >
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"
          />
          <polyline points="14 2 14 8 20 8" />
          <line x1="16" y1="13" x2="8" y2="13" />
          <line x1="16" y1="17" x2="8" y2="17" />
        </svg>
        <span>{t($locale, "sidebar.logs")}</span>
      </button>
      <button
        class:selected={activeTab === "tracce"}
        on:click={() => onSelect("tracce")}
      >
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
        </svg>
        <span>{t($locale, "sidebar.traces")}</span>
      </button>
    </nav>
  </div>

  <div class="sidebar-footer">
    {#if $authState.user}
      <div class="user-section">
        <div class="user-info">
          <div class="avatar">
            {$authState.user.username.charAt(0).toUpperCase()}
          </div>
          <div class="user-details">
            <span class="username">{$authState.user.username}</span>
            <span class="role">{roleLabel}</span>
          </div>
        </div>
        <div class="user-actions">
          <button class="user-icon-btn logout-btn" on:click={logoutUser} title={t($locale, "sidebar.logout")}>
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
              <polyline points="16 17 21 12 16 7" />
              <line x1="21" y1="12" x2="9" y2="12" />
            </svg>
          </button>
          <a class="user-icon-btn settings-icon-btn" href="/settings" title={t($locale, "sidebar.settings")}>
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              aria-hidden="true"
            >
              <circle cx="12" cy="12" r="3"></circle>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.01a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51h.01a1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.01a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
            </svg>
          </a>
        </div>
      </div>
    {/if}
  </div>

</aside>

<style>
  .sidebar {
    display: flex;
    flex-direction: column;
    gap: 32px;
    padding: 24px 16px;
    background: linear-gradient(180deg, var(--color-slate-950) 0%, var(--color-slate-900) 100%);
    width: 240px;
    box-sizing: border-box;
    height: 100vh;
    position: fixed;
    top: 0;
    left: 0;
    flex-shrink: 0;
    overflow-y: auto;
    overflow-x: hidden;
    z-index: 50;
  }

  .sidebar::before {
    content: "";
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%239C92AC' fill-opacity='0.03'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E");
    pointer-events: none;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 8px;
    position: relative;
    z-index: 1;
  }

  .logo {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 56px;
    height: 56px;
    border-radius: 16px;
    box-shadow: 0 8px 24px rgba(var(--rgb-primary-600), 0.4);
    overflow: hidden;
    background: transparent;
  }

  .logo img {
    width: 100%;
    height: 100%;
    display: block;
    object-fit: cover;
  }

  .brand-text {
    display: flex;
    flex-direction: column;
  }

  .name {
    font-weight: 700;
    font-size: 15px;
    color: var(--color-slate-50);
    letter-spacing: -0.02em;
  }

  .tagline {
    font-size: 13px;
    color: var(--color-slate-500);
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }

  .nav-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
    position: relative;
    z-index: 1;
  }

  .nav-label {
    font-size: 11px;
    font-weight: 600;
    color: var(--color-slate-500);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    padding: 0 12px;
    margin-bottom: 4px;
  }

  nav {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .nav-section button {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 14px;
    border-radius: 10px;
    border: none;
    background: transparent;
    font-size: 14px;
    font-weight: 500;
    color: var(--color-slate-400);
    cursor: pointer;
    transition: all 0.2s ease;
    text-align: left;
  }

  .nav-section button:hover {
    background: rgba(255, 255, 255, 0.05);
    color: var(--color-slate-200);
  }

  .nav-section button.selected {
    background: linear-gradient(
      135deg,
      rgba(var(--rgb-primary-600), 0.2) 0%,
      rgba(139, 92, 246, 0.2) 100%
    );
    color: #a5b4fc;
    box-shadow: inset 0 0 0 1px rgba(var(--rgb-primary-600), 0.3);
  }

  .nav-section button.selected svg {
    color: #818cf8;
  }

  .sidebar-footer {
    margin-top: auto;
    padding: 0 8px;
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .user-section {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    border: 1px solid rgba(255, 255, 255, 0.08);
  }

  .user-info {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
    flex: 1;
  }

  .avatar {
    width: 36px;
    height: 36px;
    border-radius: 10px;
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    font-size: 14px;
  }

  .user-details {
    display: flex;
    flex-direction: column;
    min-width: 0;
    flex: 1;
  }

  .username {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-slate-100);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .role {
    font-size: 11px;
    color: var(--color-slate-500);
    text-transform: capitalize;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .user-icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border-radius: 8px;
    color: var(--color-slate-500);
    background: rgba(148, 163, 184, 0.12);
    border: 1px solid rgba(148, 163, 184, 0.2);
    transition: all 0.2s ease;
    flex: 0 0 30px;
  }

  .user-icon-btn svg {
    width: 15px;
    height: 15px;
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  .logout-btn {
    border: none;
    padding: 0;
    background: rgba(148, 163, 184, 0.12);
    cursor: pointer;
  }

  .user-actions {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    flex: 0 0 auto;
  }

  .settings-icon-btn {
    color: var(--color-slate-400);
    text-decoration: none;
  }

  .logout-btn:hover {
    background: rgba(239, 68, 68, 0.22);
    border-color: rgba(248, 113, 113, 0.35);
    color: #fda4af;
  }

  .settings-icon-btn:hover {
    background: rgba(var(--rgb-primary-600), 0.24);
    border-color: rgba(129, 140, 248, 0.45);
    color: var(--color-primary-200);
  }

</style>
