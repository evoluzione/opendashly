<script lang="ts">
  import { onMount } from 'svelte';
  import Sidebar from '../components/Sidebar.svelte';
  import ServiceDropdown from '../components/ServiceDropdown.svelte';
  import QueryForm from '../components/QueryForm.svelte';
  import LogResultsTable from '../components/LogResultsTable.svelte';
  import TraceResultsList from '../components/TraceResultsList.svelte';
  import MetricChart from '../components/MetricChart.svelte';
  import { executeQuery, queryState, setAutoRefresh } from '../lib/stores/query';

  let activeTab: 'logs' | 'metriche' | 'tracce' = 'logs';

  function handleRun(event: CustomEvent) {
    const { request, autoRefreshSeconds } = event.detail;
    void executeQuery(request);
    setAutoRefresh(autoRefreshSeconds);
  }
</script>

<div class="dashboard">
  <Sidebar {activeTab} onSelect={(tab) => (activeTab = tab)} />

  <section class="content">
    <div class="content-header">
      <div>
        <h2>{activeTab === 'logs' ? 'Logs' : activeTab === 'metriche' ? 'Metriche' : 'Tracce'}</h2>
        <p>Esplora i dati con filtri espliciti e servizi selezionabili.</p>
      </div>
      <div class="service">
        <ServiceDropdown />
      </div>
    </div>

    <div class="results">
      {#if $queryState.loading}
        <div class="status">Caricamento risultati...</div>
      {:else if $queryState.error}
        <div class="status error">{$queryState.error}</div>
      {:else if !$queryState.result}
        <div class="status">Avvia una query per vedere i risultati.</div>
      {:else}
        {#if activeTab === 'logs'}
          <LogResultsTable logs={$queryState.result.results.logs} />
        {:else if activeTab === 'metriche'}
          <MetricChart series={$queryState.result.results.metrics} />
        {:else}
          <TraceResultsList traces={$queryState.result.results.traces} />
        {/if}
      {/if}
    </div>
  </section>

  <aside class="filters">
    <div class="panel">
      <h3>Filtri query</h3>
      <QueryForm on:run={handleRun} />
    </div>
  </aside>
</div>

<style>
  .dashboard {
    display: grid;
    grid-template-columns: 240px minmax(0, 1fr) 320px;
    height: 100vh;
    width: 100%;
    overflow: hidden;
  }
  
  .content {
    background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
    padding: 28px;
    display: flex;
    flex-direction: column;
    gap: 24px;
    overflow-x: hidden;
    overflow-y: auto;
  }
  
  .content-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 20px;
    padding-bottom: 20px;
    border-bottom: 1px solid rgba(15, 23, 42, 0.06);
    flex-wrap: wrap;
  }
  
  .content-header h2 {
    margin: 0 0 6px 0;
    font-size: 24px;
    font-weight: 700;
    color: #0f172a;
  }
  
  .content-header p {
    margin: 0;
    color: #64748b;
    font-size: 14px;
  }
  
  .service {
    min-width: 200px;
  }
  
  .results {
    flex: 1;
    background: white;
    border-radius: 16px;
    padding: 24px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(15, 23, 42, 0.06);
    overflow-x: auto;
  }
  
  .filters {
    background: white;
    border-left: 1px solid rgba(15, 23, 42, 0.06);
    overflow-y: auto;
  }
  
  .filters .panel {
    padding: 24px;
  }
  
  .filters h3 {
    margin: 0 0 20px 0;
    font-size: 14px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
  }
  
  .status {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 200px;
    color: #94a3b8;
    font-size: 15px;
  }
  
  .status.error {
    color: #ef4444;
    background: rgba(239, 68, 68, 0.05);
    border-radius: 12px;
    padding: 16px;
  }
  
  @media (max-width: 1200px) {
    .dashboard {
      grid-template-columns: 220px 1fr;
      height: auto;
    }
    .filters {
      grid-column: span 2;
      border-left: none;
      border-top: 1px solid rgba(15, 23, 42, 0.06);
    }
    .filters .panel {
      padding: 24px;
    }
  }
  
  @media (max-width: 820px) {
    .dashboard {
      grid-template-columns: 1fr;
    }
    .filters {
      grid-column: 1;
    }
  }
</style>
