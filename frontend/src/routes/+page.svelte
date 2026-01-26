<script lang="ts">
  import { onMount } from "svelte";
  import Sidebar from "../components/Sidebar.svelte";
  import ServiceDropdown from "../components/ServiceDropdown.svelte";
  import LogLevelDropdown from "../components/LogLevelDropdown.svelte";
  import QueryForm from "../components/QueryForm.svelte";
  import LogResultsTable from "../components/LogResultsTable.svelte";
  import TraceResultsList from "../components/TraceResultsList.svelte";
  import MetricChart from "../components/MetricChart.svelte";
  import {
    executeQuery,
    queryState,
    setAutoRefresh,
  } from "../lib/stores/query";
  import {
    dashboardState,
    loadDashboard,
    setDashboardAutoRefresh,
  } from "../lib/stores/dashboard";

  // Dashboard components
  import ApdexGauge from "../components/dashboard/ApdexGauge.svelte";
  import ErrorRateGauge from "../components/dashboard/ErrorRateGauge.svelte";
  import ThroughputGauge from "../components/dashboard/ThroughputGauge.svelte";
  import LatencyDistributionChart from "../components/dashboard/LatencyDistributionChart.svelte";
  import ThroughputChart from "../components/dashboard/ThroughputChart.svelte";
  import SlowestEndpointsTable from "../components/dashboard/SlowestEndpointsTable.svelte";
  import ErrorHotspotsTable from "../components/dashboard/ErrorHotspotsTable.svelte";
  import DashboardServiceFilter from "../components/DashboardServiceFilter.svelte";

  let activeTab: "logs" | "metriche" | "tracce" = "metriche";
  let metricsLoaded = false;
  let dashboardLoaded = false;

  onMount(() => {
    // Load dashboard metrics on startup
    loadDashboardMetrics();
  });

  function handleRun(event: CustomEvent) {
    const { request, autoRefreshSeconds } = event.detail;
    void executeQuery(request);
    setAutoRefresh(autoRefreshSeconds);
  }

  function handleTabSelect(tab: "logs" | "metriche" | "tracce") {
    activeTab = tab;
    if (tab === "metriche" && !dashboardLoaded) {
      loadDashboardMetrics();
    }
  }

  async function loadDashboardMetrics() {
    const now = new Date();
    const from = new Date(now.getTime() - 24 * 60 * 60 * 1000); // ultime 24 ore
    await loadDashboard({
      from: from.toISOString(),
      to: now.toISOString(),
    });
    dashboardLoaded = true;
    lastRefresh = new Date();
    // Also load the existing metrics chart
    await loadAllMetrics();
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

  async function loadAllMetrics() {
    const now = new Date();
    const from = new Date(now.getTime() - 24 * 60 * 60 * 1000); // ultime 24 ore
    await executeQuery({
      signals: ["metrics"],
      timeRange: {
        from: from.toISOString(),
        to: now.toISOString(),
      },
      filters: {},
      limit: 1000,
    });
    metricsLoaded = true;
  }
</script>

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
          <DashboardServiceFilter />
        </div>
      {:else}
        <div class="header-filters">
          <ServiceDropdown />
          {#if activeTab === "logs"}
            <LogLevelDropdown />
          {/if}
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

          <!-- Satisfaction Gauges -->
          <div class="gauges-row">
            <ApdexGauge data={$dashboardState.data.satisfaction.apdex} />
            <ErrorRateGauge
              errorRate={$dashboardState.data.satisfaction.errorRate}
              totalErrors={$dashboardState.data.satisfaction.throughput
                .totalErrors}
              totalRequests={$dashboardState.data.satisfaction.throughput
                .totalRequests}
            />
            <ThroughputGauge
              data={$dashboardState.data.satisfaction.throughput}
            />
          </div>

          <!-- Existing OTel Metrics Chart -->
          {#if $queryState.result?.results?.metrics}
            <div class="otel-metrics-section">
              <MetricChart series={$queryState.result.results.metrics} />
            </div>
          {/if}

          <!-- Charts Row -->
          <div class="charts-row">
            <LatencyDistributionChart
              data={$dashboardState.data.hotspots.latencyDistribution}
            />
            <ThroughputChart
              data={$dashboardState.data.satisfaction.timeSeries}
            />
          </div>

          <!-- Tables Row -->
          <div class="tables-row">
            <SlowestEndpointsTable
              data={$dashboardState.data.hotspots.slowestEndpoints}
            />
            <ErrorHotspotsTable
              data={$dashboardState.data.hotspots.errorHotspots}
            />
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
          <LogResultsTable logs={$queryState.result.results.logs} />
        {:else}
          <TraceResultsList traces={$queryState.result.results.traces} />
        {/if}
      {/if}
    </div>
  </section>

  {#if activeTab !== "metriche"}
    <aside class="filters">
      <div class="panel">
        <h3>Filtri query</h3>
        <QueryForm {activeTab} on:run={handleRun} />
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
    gap: 20px;
    flex-wrap: wrap;
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
    gap: 6px;
    padding: 8px 16px;
    background: #2563eb;
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition:
      background 0.15s ease,
      transform 0.15s ease;
  }

  .refresh-btn:hover:not(:disabled) {
    background: #1d4ed8;
  }

  .refresh-btn:active:not(:disabled) {
    transform: scale(0.98);
  }

  .refresh-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .refresh-icon {
    width: 16px;
    height: 16px;
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

  .otel-metrics-section {
    background: white;
    border-radius: 16px;
    padding: 20px;
    box-shadow:
      0 1px 3px rgba(0, 0, 0, 0.05),
      0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(15, 23, 42, 0.06);
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
    .gauges-row {
      grid-template-columns: 1fr;
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
</style>
