<script lang="ts">
  import { goto } from "$app/navigation";
  import Sidebar from "../../components/Sidebar.svelte";

  export let params: Record<string, string> = {};

  function handleSelect(tab: string) {
    void goto(`/?tab=${tab}&autorun=1`);
  }
</script>

<div class="settings-shell">
  <Sidebar activeTab={null} onSelect={handleSelect} />
  <main class="settings-content">
    <div class="settings-inner">
      <slot />
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
    max-width: 800px;
    margin: 0 auto;
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
