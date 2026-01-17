<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchRelated } from '../services/traces';
  import LogResultsTable from './LogResultsTable.svelte';
  import MetricChart from './MetricChart.svelte';

  export let traceId: string;
  let related: { logs: any[]; metrics: any[] } | null = null;
  let error: string | null = null;

  onMount(async () => {
    try {
      related = await fetchRelated(traceId);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Impossibile caricare la telemetria correlata';
    }
  });
</script>

<section class="panel">
  <header>
    <div>
      <h3>Telemetry correlata</h3>
      <p>Log e metriche legate alla traccia selezionata.</p>
    </div>
  </header>
  {#if error}
    <p class="error">{error}</p>
  {:else if !related}
    <p class="loading">Caricamento...</p>
  {:else}
    <div class="grid">
      <div class="block">
        <h4>Log</h4>
        <LogResultsTable logs={related.logs} />
      </div>
      <div class="block">
        <h4>Metriche</h4>
        <MetricChart series={related.metrics} />
      </div>
    </div>
  {/if}
</section>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    gap: 20px;
    background: white;
    border-radius: 16px;
    padding: 24px;
    border: 1px solid rgba(148, 163, 184, 0.3);
    box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  h3 {
    margin: 0 0 6px 0;
    font-size: 20px;
    color: #0f172a;
  }

  p {
    margin: 0;
    color: #64748b;
    font-size: 13px;
  }

  .grid {
    display: grid;
    gap: 20px;
  }

  .block {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .block h4 {
    margin: 0;
    font-size: 14px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #64748b;
  }

  .error {
    color: #b91c1c;
    background: #fee2e2;
    padding: 12px 16px;
    border-radius: 10px;
  }

  .loading {
    color: #94a3b8;
    font-size: 14px;
  }
</style>
