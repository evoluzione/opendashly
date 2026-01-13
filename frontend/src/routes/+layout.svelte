<script lang="ts">
  export let data;
</script>

<div class="app-shell">
  <header>
    <h1>Telemetry Dashboard</h1>
    <nav>
      <a href="/query">Query</a>
      <a href="/queries">Saved Queries</a>
    </nav>
  </header>
  <main>
    <slot />
  </main>
</div>

<style>
  .app-shell {
    font-family: "Space Grotesk", system-ui, sans-serif;
    padding: 24px;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
  }
  nav a {
    margin-right: 16px;
    text-decoration: none;
    color: #1a1a1a;
  }
</style>
