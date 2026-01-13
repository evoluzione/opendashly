<script lang="ts">
  import QueryForm from '../../components/QueryForm.svelte';
  import LogResultsTable from '../../components/LogResultsTable.svelte';
  import TraceResultsList from '../../components/TraceResultsList.svelte';
  import MetricChart from '../../components/MetricChart.svelte';
  import { queryState, executeQuery } from '../../lib/stores/query';
  import { saveQuery } from '../../services/saved_queries';
  import type { QueryRequest } from '../../services/query';

  let queryName = '';
  let lastRequest: QueryRequest | null = null;

  async function handleRun(event) {
    lastRequest = event.detail;
    await executeQuery(event.detail);
  }

  async function handleSave() {
    if (!lastRequest) return;
    await saveQuery({
      name: queryName || 'Saved Query',
      description: '',
      request: lastRequest
    });
  }
</script>

<QueryForm on:run={handleRun} />

{#if $queryState.loading}
  <p>Loading...</p>
{:else if $queryState.error}
  <p class="error">{$queryState.error}</p>
{:else if !$queryState.result}
  <p class="empty">No results yet. Run a query to see telemetry.</p>
{:else}
  <section>
    <h2>Logs</h2>
    <LogResultsTable logs={$queryState.result.results.logs} />
  </section>
  <section>
    <h2>Traces</h2>
    <TraceResultsList traces={$queryState.result.results.traces} />
  </section>
  <section>
    <h2>Metrics</h2>
    <MetricChart series={$queryState.result.results.metrics} />
  </section>
  <div class="save">
    <input bind:value={queryName} placeholder="Save query as" />
    <button on:click={handleSave}>Save Query</button>
  </div>
{/if}

<style>
  section {
    margin-bottom: 24px;
  }
  .empty {
    color: #666;
  }
  .error {
    color: #b91c1c;
  }
  .save {
    display: flex;
    gap: 8px;
  }
</style>
