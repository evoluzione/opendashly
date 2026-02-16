<script lang="ts">
  import { onMount } from "svelte";
  import { get } from "svelte/store";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import Sidebar from "../components/Sidebar.svelte";
  import QueryForm from "../components/QueryForm.svelte";
  import LogResultsTable from "../components/LogResultsTable.svelte";
  import TraceResultsList from "../components/TraceResultsList.svelte";
  import {
    executeQuery,
    queryState,
    setAutoRefresh,
    servicesState,
    loadServices,
  } from "../lib/stores/query";
  import {
    dashboardState,
    loadDashboard,
    selectDashboardService,
  } from "../lib/stores/dashboard";
  import {
    loadDashboardSettings,
    isChartEnabled,
    getOrderedChartKeys,
  } from "../lib/stores/dashboard_settings";

  // Dashboard components
  import ApdexGauge from "../components/dashboard/ApdexGauge.svelte";
  import ErrorRateGauge from "../components/dashboard/ErrorRateGauge.svelte";
  import ThroughputGauge from "../components/dashboard/ThroughputGauge.svelte";
  import LatencyDistributionChart from "../components/dashboard/LatencyDistributionChart.svelte";
  import ThroughputChart from "../components/dashboard/ThroughputChart.svelte";
  import LatencyPercentilesChart from "../components/dashboard/LatencyPercentilesChart.svelte";
  import ErrorRateChart from "../components/dashboard/ErrorRateChart.svelte";
  import StatusCodeBreakdownChart from "../components/dashboard/StatusCodeBreakdownChart.svelte";
  import TopEndpointsThroughputTable from "../components/dashboard/TopEndpointsThroughputTable.svelte";
  import LogVolumeChart from "../components/dashboard/LogVolumeChart.svelte";
  import LogLevelDistributionChart from "../components/dashboard/LogLevelDistributionChart.svelte";
  import SlowestEndpointsTable from "../components/dashboard/SlowestEndpointsTable.svelte";
  import ErrorHotspotsTable from "../components/dashboard/ErrorHotspotsTable.svelte";

  let activeTab: "logs" | "metriche" | "tracce" = "metriche";
  let dashboardLoaded = false;
  let initialTraceId: string | null = null;
  let forceMode: "auto" | "manual" | "smart" | null = null;
  let autoRun = false;
  let tabFromUrl = "";
  let tabFromUrlApplied = false;
  let dashboardAllTime = true;
  let dashboardFromInput = "";
  let dashboardToInput = "";
  let dashboardRangeError = "";
  let showDashboardFilters = false;
  let activeDashboardFilters = 0;
  let dashboardFiltersRef: HTMLDivElement | null = null;
  let queryFormRef:
    | {
        resetFiltersToDefault: () => void;
        refreshCurrentQuery: () => void;
      }
    | null = null;

  $: {
    const params = $page.url.searchParams;
    initialTraceId = params.get("traceId");
    autoRun = params.get("autorun") === "1";
    const mode = params.get("mode");
    forceMode =
      mode === "auto" || mode === "manual" || mode === "smart" ? mode : null;
    const tabParam = params.get("tab") ?? "";
    if (tabParam && tabParam !== tabFromUrl) {
      tabFromUrl = tabParam;
      tabFromUrlApplied = false;
    }
    if (initialTraceId) {
      activeTab = "tracce";
      tabFromUrlApplied = true;
    } else if (!tabFromUrlApplied) {
      if (
        tabParam === "logs" ||
        tabParam === "metriche" ||
        tabParam === "tracce"
      ) {
        activeTab = tabParam;
        tabFromUrlApplied = true;
      }
    }
  }

  $: {
    const params = new URLSearchParams($page.url.searchParams);
    if (params.get("tab") !== activeTab) {
      params.set("tab", activeTab);
      void goto(`${$page.url.pathname}?${params.toString()}`, {
        replaceState: true,
      });
    }
  }

  onMount(() => {
    // Load dashboard metrics on startup
    loadDashboardMetrics();
    void loadDashboardSettings();
    void loadServices();
  });

  function formatDateTimeLocal(date: Date): string {
    const pad = (n: number) => String(n).padStart(2, "0");
    const y = date.getFullYear();
    const m = pad(date.getMonth() + 1);
    const d = pad(date.getDate());
    const h = pad(date.getHours());
    const min = pad(date.getMinutes());
    return `${y}-${m}-${d}T${h}:${min}`;
  }

  function toIsoFromLocal(value: string): string | null {
    if (!value) return null;
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return null;
    return date.toISOString();
  }

  $: if (dashboardAllTime) {
    dashboardRangeError = "";
  }

  $: if (!dashboardAllTime && (!dashboardFromInput || !dashboardToInput)) {
    const now = new Date();
    const from = new Date(now.getTime() - 60 * 60 * 1000);
    if (!dashboardFromInput) dashboardFromInput = formatDateTimeLocal(from);
    if (!dashboardToInput) dashboardToInput = formatDateTimeLocal(now);
  }

  let lastRequest: any = null;
  const logsCursorByPage = new Map<number, string>();
  const tracesCursorByPage = new Map<number, string>();
  let pageSize = "100";
  const pageSizeOptions = ["25", "50", "100", "200"];

  function handleRun(event: CustomEvent) {
    const { request, autoRefreshSeconds } = event.detail;
    logsCursorByPage.clear();
    tracesCursorByPage.clear();
    lastRequest = {
      ...request,
      page: 1,
      logsCursor: undefined,
      tracesCursor: undefined,
    };
    void executeQuery(lastRequest).then(() => storeNextCursors(1));
    setAutoRefresh(autoRefreshSeconds);
  }

  async function handlePageChange(
    signal: "logs" | "traces" | "metrics",
    nextPage: number,
  ) {
    if (!lastRequest) return;
    const page = nextPage < 1 ? 1 : nextPage;
    const updated = {
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
    lastRequest = updated;
    await executeQuery(updated, { retainResult: true });
    storeNextCursors(page);
  }

  async function handlePageSizeChange() {
    if (!lastRequest) return;
    const limit = Number(pageSize) || 100;
    logsCursorByPage.clear();
    tracesCursorByPage.clear();
    const updated = {
      ...lastRequest,
      signals: ["logs", "traces", "metrics"],
      limit,
      page: 1,
      logsCursor: undefined,
      tracesCursor: undefined,
    };
    lastRequest = updated;
    await executeQuery(updated, { retainResult: true });
    storeNextCursors(1);
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

  function handleTabSelect(tab: "logs" | "metriche" | "tracce") {
    activeTab = tab;
    if (tab === "metriche" && !dashboardLoaded) {
      loadDashboardMetrics();
    }
  }

  async function loadDashboardMetrics() {
    const request: { from?: string; to?: string; serviceName?: string } = {};
    if ($dashboardState.selectedService) {
      request.serviceName = $dashboardState.selectedService;
    }
    if (!dashboardAllTime) {
      const fromIso = toIsoFromLocal(dashboardFromInput);
      const toIso = toIsoFromLocal(dashboardToInput);
      if (!fromIso || !toIso) {
        dashboardRangeError = "Inserisci una data/ora valida per inizio e fine.";
        return;
      }
      if (new Date(toIso).getTime() <= new Date(fromIso).getTime()) {
        dashboardRangeError = "La data/ora di fine deve essere successiva all'inizio.";
        return;
      }
      dashboardRangeError = "";
      request.from = fromIso;
      request.to = toIso;
    }
    await loadDashboard(request);
    dashboardLoaded = true;
    lastRefresh = new Date();
  }

  let lastRefresh: Date | null = null;

  function formatLastRefresh(date: Date | null): string {
    if (!date) return "";
    return date.toLocaleTimeString("it-IT", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }

  async function handleRefresh() {
    await loadDashboardMetrics();
  }

  function enableDashboardPeriod() {
    if (!dashboardAllTime) return;
    dashboardAllTime = false;
    dashboardRangeError = "";
  }

  function handleDashboardServiceChange(event: Event) {
    const value = (event.target as HTMLSelectElement).value;
    selectDashboardService(value === "Tutti" ? null : value, {
      reload: false,
    });
  }

  async function applyDashboardFilters() {
    await loadDashboardMetrics();
    if (!dashboardRangeError) {
      showDashboardFilters = false;
    }
  }

  async function resetDashboardFilters() {
    dashboardAllTime = true;
    dashboardRangeError = "";
    dashboardFromInput = "";
    dashboardToInput = "";
    selectDashboardService(null, { reload: false });
    await loadDashboardMetrics();
    showDashboardFilters = false;
  }

  function handleResetQueryFilters() {
    queryFormRef?.resetFiltersToDefault();
  }

  function handleRefreshQueryFilters() {
    queryFormRef?.refreshCurrentQuery();
  }

  $: activeDashboardFilters =
    (dashboardAllTime ? 0 : 1) + ($dashboardState.selectedService ? 1 : 0);

  function handleGlobalClick(event: MouseEvent) {
    if (!showDashboardFilters || !dashboardFiltersRef) return;
    const target = event.target as Node | null;
    if (target && !dashboardFiltersRef.contains(target)) {
      showDashboardFilters = false;
    }
  }

</script>

<svelte:window on:click={handleGlobalClick} />

<div class="dashboard" class:metrics-view={activeTab === "metriche"}>
  <Sidebar {activeTab} onSelect={handleTabSelect} />

  <section class="content">
    <div class="content-header">
      <div>
        <h2>
          {activeTab === "logs"
            ? "Log"
            : activeTab === "metriche"
              ? "Metriche"
              : "Tracce"}
        </h2>
        <p>
          {activeTab === "metriche"
            ? "Dashboard performance e metriche avanzate."
            : "Esplora i dati con filtri espliciti e servizi selezionabili."}
        </p>
      </div>
      {#if activeTab === "metriche"}
        <div class="metrics-controls">
          <div class="refresh-controls">
            {#if lastRefresh}
              <span class="last-refresh"
                >Ultimo aggiornamento: {formatLastRefresh(lastRefresh)}</span
              >
            {/if}
            <button
              class="refresh-btn"
              on:click={handleRefresh}
              disabled={$dashboardState.loading}
              title="Aggiorna metriche"
              aria-label="Aggiorna metriche"
            >
              <svg
                class="refresh-icon"
                class:spinning={$dashboardState.loading}
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fill-rule="evenodd"
                  d="M15.312 11.424a5.5 5.5 0 01-9.201 2.466l-.312-.311h2.433a.75.75 0 000-1.5H3.989a.75.75 0 00-.75.75v4.242a.75.75 0 001.5 0v-2.43l.31.31a7 7 0 0011.712-3.138.75.75 0 00-1.449-.389zm1.23-7.424a.75.75 0 00-.75.75v2.43l-.31-.31A7 7 0 003.77 9.89a.75.75 0 101.45.388 5.5 5.5 0 019.201-2.466l.312.311h-2.433a.75.75 0 000 1.5h4.243a.75.75 0 00.75-.75V4.75a.75.75 0 00-.75-.75z"
                  clip-rule="evenodd"
                />
              </svg>
              Aggiorna
            </button>
          </div>
          <div class="dashboard-filters-dropdown" bind:this={dashboardFiltersRef}>
            <button
              class="filters-btn"
              class:active={activeDashboardFilters > 0}
              on:click={() => (showDashboardFilters = !showDashboardFilters)}
              aria-expanded={showDashboardFilters}
              aria-haspopup="true"
              type="button"
            >
              <svg
                class="filters-icon"
                viewBox="0 0 24 24"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
                aria-hidden="true"
              >
                <path
                  d="M4 6H20M7 12H17M10 18H14"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                />
              </svg>
              Filtri
              {#if activeDashboardFilters > 0}
                <span class="filters-badge">{activeDashboardFilters}</span>
              {/if}
            </button>
            {#if showDashboardFilters}
              <div class="filters-menu">
                <div class="filter-block">
                  <label class="filter-label" for="dashboard-service-dropdown"
                    >Servizio</label
                  >
                  <select
                    id="dashboard-service-dropdown"
                    class="filter-select"
                    on:change={handleDashboardServiceChange}
                    value={$dashboardState.selectedService || "Tutti"}
                  >
                    <option value="Tutti">Tutti i servizi</option>
                    {#each $servicesState.services as service}
                      <option value={service}>{service}</option>
                    {/each}
                  </select>
                </div>

                <div class="filter-block">
                  <span class="filter-label">Periodo (data + ora)</span>
                  <div class="range-inputs">
                    <input
                      type="datetime-local"
                      bind:value={dashboardFromInput}
                      aria-label="Data ora inizio"
                      readonly={dashboardAllTime}
                      on:focus={enableDashboardPeriod}
                      on:click={enableDashboardPeriod}
                    />
                    <input
                      type="datetime-local"
                      bind:value={dashboardToInput}
                      aria-label="Data ora fine"
                      readonly={dashboardAllTime}
                      on:focus={enableDashboardPeriod}
                      on:click={enableDashboardPeriod}
                    />
                  </div>
                </div>
                {#if dashboardRangeError}
                  <span class="range-error">{dashboardRangeError}</span>
                {/if}
                <div class="filters-actions">
                  <button
                    class="reset-filters-btn"
                    type="button"
                    on:click={resetDashboardFilters}
                    disabled={$dashboardState.loading}
                  >
                    Reset filtri
                  </button>
                  <button
                    class="apply-filters-btn"
                    type="button"
                    on:click={applyDashboardFilters}
                    disabled={$dashboardState.loading}
                  >
                    Applica
                  </button>
                </div>
              </div>
            {/if}
          </div>
        </div>
      {:else}
        <div class="header-filters">
          <div class="page-size-selector">
            <label for="page-size">Risultati</label>
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
        </div>
      {/if}
    </div>

    <div class="results" class:dashboard-results={activeTab === "metriche"}>
      {#if activeTab === "metriche"}
        {#if $dashboardState.error}
          <div class="status error">{$dashboardState.error}</div>
        {:else if $dashboardState.loading && !$dashboardState.data}
          <div class="status">Caricamento dashboard...</div>
        {:else if $dashboardState.data}
          {#if $dashboardState.loading}
            <div class="refresh-indicator">Aggiornamento in corso...</div>
          {/if}

          <div class="dashboard-grid">
            {#each getOrderedChartKeys() as chartKey}
              {#if isChartEnabled(chartKey)}
                {#if chartKey === "apdex_gauge"}
                  <ApdexGauge data={$dashboardState.data.satisfaction.apdex} />
                {:else if chartKey === "error_rate_gauge"}
                  <ErrorRateGauge
                    errorRate={$dashboardState.data.satisfaction.errorRate}
                    totalErrors={$dashboardState.data.satisfaction.throughput
                      .totalErrors}
                    totalRequests={$dashboardState.data.satisfaction.throughput
                      .totalRequests}
                  />
                {:else if chartKey === "throughput_gauge"}
                  <ThroughputGauge
                    data={$dashboardState.data.satisfaction.throughput}
                  />
                {:else if chartKey === "latency_distribution"}
                  <LatencyDistributionChart
                    data={$dashboardState.data.hotspots.latencyDistribution}
                  />
                {:else if chartKey === "throughput_timeseries"}
                  <ThroughputChart
                    data={$dashboardState.data.satisfaction.timeSeries}
                  />
                {:else if chartKey === "latency_percentiles"}
                  <LatencyPercentilesChart
                    data={$dashboardState.data.satisfaction.latencySeries}
                  />
                {:else if chartKey === "error_rate_timeseries"}
                  <ErrorRateChart
                    data={$dashboardState.data.satisfaction.errorRateSeries}
                  />
                {:else if chartKey === "log_volume"}
                  <LogVolumeChart
                    data={$dashboardState.data.logs.volumeSeries}
                  />
                {:else if chartKey === "log_levels"}
                  <LogLevelDistributionChart
                    data={$dashboardState.data.logs.levels}
                  />
                {:else if chartKey === "slowest_endpoints"}
                  <SlowestEndpointsTable
                    data={$dashboardState.data.hotspots.slowestEndpoints}
                  />
                {:else if chartKey === "top_endpoints_throughput"}
                  <TopEndpointsThroughputTable
                    data={$dashboardState.data.hotspots.topEndpoints}
                  />
                {:else if chartKey === "error_hotspots"}
                  <ErrorHotspotsTable
                    data={$dashboardState.data.hotspots.errorHotspots}
                  />
                {:else if chartKey === "status_codes"}
                  <StatusCodeBreakdownChart
                    data={$dashboardState.data.hotspots.statusCodes}
                  />
                {/if}
              {/if}
            {/each}
          </div>
        {:else}
          <div class="status">Caricamento metriche...</div>
        {/if}
      {:else if $queryState.error}
        <div class="status error">{$queryState.error}</div>
      {:else if !$queryState.result}
        <div class="status">
          {$queryState.loading
            ? "Caricamento risultati..."
            : "Avvia una query per vedere i risultati."}
        </div>
      {:else}
        {#if $queryState.loading}
          <div class="status">Aggiornamento in corso...</div>
        {/if}
        {#if activeTab === "logs"}
          <LogResultsTable
            logs={$queryState.result.results.logs}
            pagination={$queryState.result.pagination?.logs ?? null}
            isLiveUpdate={$queryState.isLiveUpdate}
            on:pageChange={(e) => handlePageChange("logs", e.detail.page)}
          />
        {:else if activeTab === "tracce"}
          <TraceResultsList
            traces={$queryState.result.results.traces}
            pagination={$queryState.result.pagination?.traces ?? null}
            isLiveUpdate={$queryState.isLiveUpdate}
            on:pageChange={(e) => handlePageChange("traces", e.detail.page)}
          />
        {/if}
      {/if}
    </div>
  </section>

  {#if activeTab !== "metriche"}
    <aside class="filters">
      <div class="panel">
        <div class="filters-panel-header">
          <h3>Filtri query</h3>
          <div class="query-header-actions">
            <button
              type="button"
              class="query-refresh-btn"
              on:click={handleRefreshQueryFilters}
              disabled={$queryState.loading}
              title="Aggiorna risultati con i filtri correnti"
              aria-label="Aggiorna risultati"
            >
              <svg
                class="refresh-icon"
                class:spinning={$queryState.loading}
                viewBox="0 0 20 20"
                fill="currentColor"
                aria-hidden="true"
              >
                <path
                  fill-rule="evenodd"
                  d="M15.312 11.424a5.5 5.5 0 01-9.201 2.466l-.312-.311h2.433a.75.75 0 000-1.5H3.989a.75.75 0 00-.75.75v4.242a.75.75 0 001.5 0v-2.43l.31.31a7 7 0 0011.712-3.138.75.75 0 00-1.449-.389zm1.23-7.424a.75.75 0 00-.75.75v2.43l-.31-.31A7 7 0 003.77 9.89a.75.75 0 101.45.388 5.5 5.5 0 019.201-2.466l.312.311h-2.433a.75.75 0 000 1.5h4.243a.75.75 0 00.75-.75V4.75a.75.75 0 00-.75-.75z"
                  clip-rule="evenodd"
                />
              </svg>
            </button>
            <button
              type="button"
              class="query-reset-btn"
              on:click={handleResetQueryFilters}
              title="Resetta tutti i filtri ai valori di default"
            >
              Reset filtri
            </button>
          </div>
        </div>
        <QueryForm
          bind:this={queryFormRef}
          {activeTab}
          {initialTraceId}
          {forceMode}
          {autoRun}
          on:run={handleRun}
        />
      </div>
    </aside>
  {/if}
</div>

<style>
  .dashboard {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 320px;
    height: 100vh;
    width: 100%;
    overflow: hidden;
    padding-left: 240px;
  }

  .dashboard.metrics-view {
    grid-template-columns: 1fr;
    overflow-x: hidden;
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

  .header-filters {
    display: flex;
    gap: 16px;
  }

  .header-filters :global(> div) {
    min-width: 160px;
  }

  .results {
    flex: 1;
    background: white;
    border-radius: 16px;
    padding: 24px;
    box-shadow:
      0 1px 3px rgba(0, 0, 0, 0.05),
      0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(15, 23, 42, 0.06);
    overflow-x: auto;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .results.dashboard-results {
    background: transparent;
    padding: 0;
    box-shadow: none;
    border: none;
    display: flex;
    flex-direction: column;
    gap: 24px;
    min-width: 0;
  }

  .dashboard-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(320px, 1fr));
    gap: 20px;
    min-width: 0;
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
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
  }

  .filters-panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 20px;
  }

  .query-header-actions {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .query-refresh-btn {
    width: 28px;
    height: 28px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: none;
    border-radius: 8px;
    background: #6366f1;
    color: #ffffff;
    cursor: pointer;
    transition:
      background 0.15s ease,
      transform 0.15s ease;
  }

  .query-refresh-btn .refresh-icon {
    width: 12px;
    height: 12px;
  }

  .query-refresh-btn:hover:not(:disabled) {
    background: #4f46e5;
  }

  .query-refresh-btn:active:not(:disabled) {
    transform: scale(0.98);
  }

  .query-refresh-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .query-reset-btn {
    padding: 7px 11px;
    border: 1px solid #cbd5e1;
    border-radius: 8px;
    background: #ffffff;
    color: #475569;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    white-space: nowrap;
    transition:
      background 0.15s ease,
      border-color 0.15s ease,
      color 0.15s ease;
  }

  .query-reset-btn:hover {
    background: #f8fafc;
    border-color: #94a3b8;
    color: #334155;
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

  .refresh-indicator {
    text-align: center;
    color: #64748b;
    font-size: 13px;
    padding: 8px 16px;
    background: rgba(37, 99, 235, 0.05);
    border-radius: 8px;
  }

  .metrics-controls {
    display: flex;
    align-items: flex-end;
    gap: 10px;
    flex-wrap: wrap;
  }

  .dashboard-filters-dropdown {
    position: relative;
  }

  .filters-btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    height: 34px;
    padding: 8px 12px;
    box-sizing: border-box;
    border-radius: 8px;
    border: 1px solid #cbd5e1;
    background: #ffffff;
    color: #1e293b;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
  }

  .filters-btn:hover {
    border-color: #94a3b8;
    background: #f8fafc;
  }

  .filters-btn.active {
    border-color: #2563eb;
    color: #1d4ed8;
    background: #eff6ff;
  }

  .filters-icon {
    width: 14px;
    height: 14px;
    flex: 0 0 14px;
  }

  .filters-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 18px;
    height: 18px;
    padding: 0 5px;
    border-radius: 999px;
    background: #2563eb;
    color: #ffffff;
    font-size: 11px;
    font-weight: 700;
  }

  .filters-menu {
    position: absolute;
    top: calc(100% + 8px);
    right: 0;
    width: 340px;
    background: #ffffff;
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    box-shadow: 0 14px 35px rgba(15, 23, 42, 0.16);
    padding: 16px;
    z-index: 40;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .filter-block {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .filter-block + .filter-block {
    padding-top: 12px;
    border-top: 1px solid #f1f5f9;
  }

  .filter-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #64748b;
  }

  .filter-select {
    padding: 11px 12px;
    border-radius: 10px;
    border: 1px solid #e2e8f0;
    background: #ffffff;
    font-size: 13px;
    color: #0f172a;
  }

  .range-inputs {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .range-inputs input {
    padding: 11px 12px;
    border-radius: 10px;
    border: 1px solid #e2e8f0;
    background: white;
    font-size: 13px;
    color: #0f172a;
  }

  .range-inputs input[readonly] {
    background: #f8fafc;
    color: #64748b;
    cursor: pointer;
  }

  .range-inputs input:focus {
    outline: none;
    border-color: #2563eb;
    box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
  }

  .range-error {
    font-size: 12px;
    color: #dc2626;
    padding: 8px 10px;
    border-radius: 8px;
    background: #fef2f2;
    border: 1px solid #fecaca;
  }

  .apply-filters-btn {
    padding: 9px 14px;
    border: none;
    border-radius: 8px;
    background: #2563eb;
    color: #ffffff;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }

  .apply-filters-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .filters-actions {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    margin-top: 4px;
    padding-top: 12px;
    border-top: 1px solid #f1f5f9;
  }

  .reset-filters-btn {
    padding: 9px 12px;
    border: 1px solid #cbd5e1;
    border-radius: 8px;
    background: #ffffff;
    color: #475569;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }

  .reset-filters-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .refresh-controls {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .last-refresh {
    font-size: 12px;
    color: #94a3b8;
  }

  .refresh-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    height: 34px;
    padding: 8px 12px;
    box-sizing: border-box;
    background: #6366f1;
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition:
      background 0.15s ease,
      transform 0.15s ease;
  }

  .refresh-btn:hover:not(:disabled) {
    background: #4f46e5;
  }

  .refresh-btn:active:not(:disabled) {
    transform: scale(0.98);
  }

  .refresh-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .refresh-btn .refresh-icon {
    width: 12px;
    height: 12px;
  }

  .refresh-icon.spinning {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  .gauges-row {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 20px;
  }


  .charts-row {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 20px;
    min-width: 0;
  }

  .charts-row > :global(*) {
    min-width: 0;
  }

  .tables-row {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 20px;
    min-width: 0;
  }

  .tables-row > :global(*) {
    min-width: 0;
  }

  @media (max-width: 1200px) {
    .dashboard {
      grid-template-columns: 1fr;
      height: auto;
      padding-left: 220px;
    }
    .filters {
      grid-column: 1;
      border-left: none;
      border-top: 1px solid rgba(15, 23, 42, 0.06);
    }
    .filters .panel {
      padding: 24px;
    }
    .gauges-row {
      grid-template-columns: repeat(3, 1fr);
    }
    .charts-row,
    .tables-row {
      grid-template-columns: 1fr;
    }
  }


  @media (max-width: 900px) {
    .dashboard-grid {
      grid-template-columns: 1fr;
    }
    .gauges-row {
      grid-template-columns: 1fr;
    }
    .range-inputs {
      grid-template-columns: 1fr;
    }
    .filters-menu {
      left: 0;
      right: auto;
      width: min(92vw, 340px);
    }
  }

  @media (min-width: 1600px) {
    .dashboard-grid {
      grid-template-columns: repeat(3, minmax(320px, 1fr));
    }
  }

  @media (max-width: 820px) {
    .dashboard {
      grid-template-columns: 1fr;
      padding-left: 0;
    }
    .filters {
      grid-column: 1;
    }
  }

  .page-size-selector {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .page-size-selector label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
  }

  .page-size-selector select {
    padding: 12px 16px;
    border-radius: 10px;
    border: 1px solid #e2e8f0;
    background: white;
    font-size: 14px;
    font-weight: 500;
    color: #0f172a;
    cursor: pointer;
    transition: all 0.2s ease;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%2364748b' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 12px center;
    padding-right: 40px;
  }

  .page-size-selector select:hover {
    border-color: #cbd5e1;
  }

  .page-size-selector select:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
  }
</style>
