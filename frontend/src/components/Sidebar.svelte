<script context="module" lang="ts">
  let persistedAdminMenuOpen = false;
</script>

<script lang="ts">
  import { authState, logoutUser } from "../lib/stores/auth";

  export let activeTab: "logs" | "metriche" | "tracce" | null = null;
  export let onSelect: (tab: "logs" | "metriche" | "tracce") => void;

  let showAdminMenu = persistedAdminMenuOpen;

  $: persistedAdminMenuOpen = showAdminMenu;
</script>

<aside class="sidebar">
  <div class="brand">
    <div class="logo">
      <svg
        width="20"
        height="20"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <path d="M12 2L2 7l10 5 10-5-10-5z" />
        <path d="M2 17l10 5 10-5" />
        <path d="M2 12l10 5 10-5" />
      </svg>
    </div>
    <div class="brand-text">
      <span class="name">Opendashly</span>
      <span class="tagline">Dashboard</span>
    </div>
  </div>

  <div class="nav-section">
    <span class="nav-label">Telemetria</span>
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
        <span>Metriche</span>
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
        <span>Log</span>
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
        <span>Tracce</span>
      </button>
    </nav>
  </div>

  {#if $authState.user?.role === "admin"}
    <div class="nav-section">
      <button
        type="button"
        class="admin-toggle"
        class:open={showAdminMenu}
        on:click={() => (showAdminMenu = !showAdminMenu)}
        aria-expanded={showAdminMenu}
        aria-controls="admin-menu"
      >
        <span>Amministrazione</span>
        <svg
          class="admin-toggle-chevron"
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          aria-hidden="true"
        >
          <polyline points="6 8 10 12 14 8"></polyline>
        </svg>
      </button>
      {#if showAdminMenu}
        <nav id="admin-menu" class="admin-nav">
        <a href="/admin/users" class="nav-link">
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
            <circle cx="9" cy="7" r="4" />
            <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
            <path d="M16 3.13a4 4 0 0 1 0 7.75" />
          </svg>
          <span>Gestione utenti</span>
        </a>
        <a href="/admin/dashboard" class="nav-link">
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M3 3h18v4H3z" />
            <path d="M3 11h10v10H3z" />
            <path d="M17 11h4v10h-4z" />
          </svg>
          <span>Dashboard</span>
        </a>
        <a href="/admin/retention" class="nav-link">
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="12" cy="12" r="10" />
            <polyline points="12 6 12 12 16 14" />
          </svg>
          <span>Conservazione e pulizia</span>
        </a>
        <a href="/settings/ai" class="nav-link">
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              d="M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"
            />
            <path d="M19 13l1 3 3 1-3 1-1 3-1-3-3-1 3-1 1-3z" />
            <path d="M5 17l1 2 2 1-2 1-1 2-1-2-2-1 2-1 1-2z" />
          </svg>
          <span>Impostazioni AI</span>
        </a>
        <a href="/admin/status" class="nav-link">
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="12" cy="12" r="9" />
            <path d="M12 8v4" />
            <circle cx="12" cy="16" r="1" fill="currentColor" stroke="none" />
          </svg>
          <span>Monitor sistema</span>
        </a>
        </nav>
      {/if}
    </div>
  {/if}

  <div class="sidebar-footer">
    {#if $authState.user}
      <div class="user-section">
        <div class="user-info">
          <div class="avatar">
            {$authState.user.username.charAt(0).toUpperCase()}
          </div>
          <div class="user-details">
            <span class="username">{$authState.user.username}</span>
            <span class="role">{$authState.user.role}</span>
          </div>
        </div>
        <button class="logout-btn" on:click={logoutUser} title="Esci">
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
    background: linear-gradient(180deg, #0f172a 0%, #1e293b 100%);
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
    width: 42px;
    height: 42px;
    border-radius: 12px;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: white;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
  }

  .brand-text {
    display: flex;
    flex-direction: column;
  }

  .name {
    font-weight: 700;
    font-size: 15px;
    color: #f8fafc;
    letter-spacing: -0.02em;
  }

  .tagline {
    font-size: 11px;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.08em;
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
    color: #64748b;
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

  button {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 14px;
    border-radius: 10px;
    border: none;
    background: transparent;
    font-size: 14px;
    font-weight: 500;
    color: #94a3b8;
    cursor: pointer;
    transition: all 0.2s ease;
    text-align: left;
  }

  button:hover {
    background: rgba(255, 255, 255, 0.05);
    color: #e2e8f0;
  }

  button.selected {
    background: linear-gradient(
      135deg,
      rgba(99, 102, 241, 0.2) 0%,
      rgba(139, 92, 246, 0.2) 100%
    );
    color: #a5b4fc;
    box-shadow: inset 0 0 0 1px rgba(99, 102, 241, 0.3);
  }

  button.selected svg {
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

  .admin-toggle {
    width: 100%;
    justify-content: space-between;
    background: transparent;
    border: none;
    color: #64748b;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    padding: 0 12px;
    border-radius: 0;
  }

  .admin-toggle:hover {
    background: transparent;
    color: #94a3b8;
  }

  .admin-toggle.open {
    color: #c7d2fe;
  }

  .admin-toggle-chevron {
    width: 14px;
    height: 14px;
    transition: transform 0.2s ease;
  }

  .admin-toggle.open .admin-toggle-chevron {
    transform: rotate(180deg);
  }

  .admin-nav {
    margin-top: 6px;
    padding: 6px;
    border-radius: 10px;
    background: rgba(15, 23, 42, 0.45);
    border: 1px solid rgba(148, 163, 184, 0.14);
  }

  .user-section {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    border: 1px solid rgba(255, 255, 255, 0.08);
  }

  .user-info {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .avatar {
    width: 36px;
    height: 36px;
    border-radius: 10px;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
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
  }

  .username {
    font-size: 13px;
    font-weight: 600;
    color: #f1f5f9;
  }

  .role {
    font-size: 11px;
    color: #64748b;
    text-transform: capitalize;
  }

  .logout-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: 8px;
    border: none;
    background: transparent;
    color: #64748b;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .logout-btn:hover {
    background: rgba(239, 68, 68, 0.15);
    color: #f87171;
  }

  .nav-link {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 14px;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 500;
    color: #94a3b8;
    text-decoration: none;
    transition: all 0.2s ease;
  }

  .nav-link:hover {
    background: rgba(255, 255, 255, 0.05);
    color: #e2e8f0;
  }

  .nav-link.selected {
    background: linear-gradient(
      135deg,
      rgba(99, 102, 241, 0.2) 0%,
      rgba(139, 92, 246, 0.2) 100%
    );
    color: #a5b4fc;
    box-shadow: inset 0 0 0 1px rgba(99, 102, 241, 0.3);
  }

</style>
