<script lang="ts">
  import { onMount } from "svelte";
  import { get } from "svelte/store";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import QueryForm from "../../components/QueryForm.svelte";
  import LogResultsTable from "../../components/LogResultsTable.svelte";
  import TraceResultsList from "../../components/TraceResultsList.svelte";
  import MetricChart from "../../components/MetricChart.svelte";
  import {
    queryState,
    executeQuery,
  } from "../../lib/stores/query";
  import { saveQuery } from "../../services/saved_queries";
  import type { QueryRequest } from "../../services/query";

  export let params: Record<string, string> = {};

  let queryName = "";
  let lastRequest: QueryRequest | null = null;
  const logsCursorByPage = new Map<number, string>();
  const tracesCursorByPage = new Map<number, string>();
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
    logsCursorByPage.clear();
    tracesCursorByPage.clear();
    lastRequest = {
      ...event.detail.request,
      limit,
      page: 1,
      logsCursor: undefined,
      tracesCursor: undefined,
    };
    await executeQuery(lastRequest);
    storeNextCursors(1);
  }

  async function handlePageChange(
    signal: "logs" | "traces" | "metrics",
    nextPage: number,
  ) {
    if (!lastRequest) return;
    const page = nextPage < 1 ? 1 : nextPage;
    lastRequest = {
      ...lastRequest,
      signals: [signal],
      page,
      logsCursor:
        signal === "logs" && page > 1 ? logsCursorByPage.get(page) : undefined,
      tracesCursor:
        signal === "traces" && page > 1
          ? tracesCursorByPage.get(page)
          : undefined,
    };
    await executeQuery(lastRequest, { retainResult: true });
    storeNextCursors(page);
  }

  async function handlePageSizeChange() {
    if (!lastRequest) return;
    const limit = Number(pageSize) || 100;
    logsCursorByPage.clear();
    tracesCursorByPage.clear();
    lastRequest = {
      ...lastRequest,
      signals: ["logs", "traces", "metrics"],
      limit,
      page: 1,
      logsCursor: undefined,
      tracesCursor: undefined,
    };
    await executeQuery(lastRequest, { retainResult: true });
    storeNextCursors(1);
  }

  async function handleSave() {
    if (!lastRequest) return;
    await saveQuery({
      name: queryName || "Query salvata",
      description: "",
      request: lastRequest,
    });
  }


  function storeNextCursors(page: number) {
    const pagination = get(queryState).result?.pagination;
    if (!pagination) return;
    if (pagination.logs?.hasNext && pagination.logs.nextCursor) {
      logsCursorByPage.set(page + 1, pagination.logs.nextCursor);
    } else {
      logsCursorByPage.delete(page + 1);
    }
    if (pagination.traces?.hasNext && pagination.traces.nextCursor) {
      tracesCursorByPage.set(page + 1, pagination.traces.nextCursor);
    } else {
      tracesCursorByPage.delete(page + 1);
    }
  }
</script>

<QueryForm
  {activeTab}
  initialTraceId={traceIdParam}
  forceMode={traceIdParam ? "manual" : null}
  autoRun={!!traceIdParam}
  on:run={handleRun}
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
