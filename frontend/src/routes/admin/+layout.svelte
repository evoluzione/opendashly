<script lang="ts">
  import { goto } from '$app/navigation';
  import Sidebar from '../../components/Sidebar.svelte';

  export let params: Record<string, string> = {};

  function handleSelect() {
    void goto('/');
  }
</script>

<div class="admin-shell">
  <Sidebar activeTab={null} onSelect={handleSelect} />
  <main class="admin-content">
    <div class="admin-inner">
      <slot />
    </div>
  </main>
</div>

<style>
  .admin-shell {
    display: grid;
    grid-template-columns: 1fr;
    min-height: 100vh;
    width: 100%;
  }

  .admin-content {
    background: linear-gradient(135deg, #f8fafc 0%, #eef2f7 100%);
    padding: 28px;
    overflow-y: auto;
    padding-left: 240px;
  }

  .admin-inner {
    max-width: 1200px;
    margin: 0 auto;
  }

  @media (max-width: 900px) {
    .admin-content {
      padding-left: 200px;
    }
  }

  @media (max-width: 720px) {
    .admin-shell {
      grid-template-columns: 1fr;
    }

    .admin-content {
      padding: 20px;
      padding-left: 20px;
    }
  }
</style>
