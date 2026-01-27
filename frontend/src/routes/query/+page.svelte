<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import QueryForm from "../../components/QueryForm.svelte";
  import LogResultsTable from "../../components/LogResultsTable.svelte";
  import TraceResultsList from "../../components/TraceResultsList.svelte";
  import MetricChart from "../../components/MetricChart.svelte";
  import {
    queryState,
    executeQuery,
    setAutoRefresh,
    stopAutoRefresh,
  } from "../../lib/stores/query";
  import { saveQuery } from "../../services/saved_queries";
  import type { QueryRequest } from "../../services/query";

  export let params: Record<string, string> = {};

  let queryName = "";
  let lastRequest: QueryRequest | null = null;
  let pageSize = "100";
  const pageSizeOptions = ["25", "50", "100", "200"];
  const activeTab = "tracce";
  let traceIdParam: string | null = null;
  $: traceIdParam = $page.url.searchParams.get("traceId");

  onMount(() => {
    const params = $page.url.searchParams.toString();
    const target = params ? `/?${params}` : "/";
    void goto(target, { replaceState: true });
  });

  async function handleRun(event) {
    const limit = Number(pageSize) || 100;
    lastRequest = { ...event.detail.request, limit, page: 1 };
    setAutoRefresh(
      event.detail.autoRefreshSeconds ?? null,
      event.detail.autoRefreshRangeMinutes ?? null,
    );
    await executeQuery(lastRequest);
  }

  function handleModeChange(event) {
    if (event.detail.mode !== "auto") {
      stopAutoRefresh();
    }
  }

  async function handlePageChange(
    _signal: "logs" | "traces" | "metrics",
    nextPage: number,
  ) {
    if (!lastRequest) return;
    const page = nextPage < 1 ? 1 : nextPage;
    lastRequest = { ...lastRequest, page };
    await executeQuery(lastRequest, { retainResult: true });
  }

  async function handlePageSizeChange() {
    if (!lastRequest) return;
    const limit = Number(pageSize) || 100;
    lastRequest = { ...lastRequest, limit, page: 1 };
    await executeQuery(lastRequest, { retainResult: true });
  }

  async function handleSave() {
    if (!lastRequest) return;
    await saveQuery({
      name: queryName || "Query salvata",
      description: "",
      request: lastRequest,
    });
  }

  onDestroy(() => {
    stopAutoRefresh();
  });
</script>

<QueryForm
  {activeTab}
  initialTraceId={traceIdParam}
  forceMode={traceIdParam ? "manual" : null}
  autoRun={!!traceIdParam}
  on:run={handleRun}
  on:modeChange={handleModeChange}
/>

{#if $queryState.error}
  <p class="error">{$queryState.error}</p>
{/if}

{#if !$queryState.result}
  {#if $queryState.loading}
    <p>Caricamento...</p>
  {:else}
    <p class="empty">
      Ancora nessun risultato. Esegui una query per vedere la telemetria.
    </p>
  {/if}
{:else}
  {#if $queryState.loading}
    <p class="loading">Aggiornamento in corso...</p>
  {/if}
  <div class="page-size">
    <label for="page-size">Risultati per pagina (log e tracce)</label>
    <select
      id="page-size"
      bind:value={pageSize}
      on:change={handlePageSizeChange}
    >
      {#each pageSizeOptions as size}
        <option value={size}>{size}</option>
      {/each}
    </select>
  </div>
  <section>
    <h2>Log</h2>
    <LogResultsTable
      logs={$queryState.result.results.logs}
      pagination={$queryState.result.pagination?.logs ?? null}
      on:pageChange={(event) => handlePageChange("logs", event.detail.page)}
    />
  </section>
  <section>
    <h2>Tracce</h2>
    <TraceResultsList
      traces={$queryState.result.results.traces}
      pagination={$queryState.result.pagination?.traces ?? null}
      on:pageChange={(event) => handlePageChange("traces", event.detail.page)}
    />
  </section>
  <section>
    <h2>Metriche</h2>
    <MetricChart
      series={$queryState.result.results.metrics}
      pagination={$queryState.result.pagination?.metrics ?? null}
      on:pageChange={(event) => handlePageChange("metrics", event.detail.page)}
    />
  </section>
  <div class="save">
    <input bind:value={queryName} placeholder="Salva query come" />
    <button on:click={handleSave}>Salva query</button>
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
  .loading {
    color: #0f172a;
    font-weight: 600;
  }
  .save {
    display: flex;
    gap: 8px;
  }
  .page-size {
    display: flex;
    align-items: center;
    gap: 12px;
    margin: 12px 0 20px;
  }
  .page-size label {
    font-size: 12px;
    font-weight: 600;
    color: #475569;
  }
  .page-size select {
    padding: 8px 12px;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    color: #0f172a;
    font-size: 13px;
    font-weight: 600;
  }
</style>
