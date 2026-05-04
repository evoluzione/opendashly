<script lang="ts">
  import { onDestroy, onMount } from "svelte";
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
    getOrderedChartSettings
  } from "../lib/stores/dashboard_settings";
  import {
    buildDashboardRequest,
    countActiveDashboardFilters,
    createInitialRunRequest,
    createPageChangeRequest,
    createPageSizeRequest,
    getDashboardTimeRangeOptions,
    defaultCustomRangeInputs,
    formatLastRefresh,
    sleep,
    storeNextCursors,
    type DashboardRangePreset,
  } from "./home-page.logic";
  import { locale, t } from "$lib/i18n";

  // Dashboard components
  import ApdexGauge from "../components/dashboard/ApdexGauge.svelte";
  import ErrorRateGauge from "../components/dashboard/ErrorRateGauge.svelte";
  import ThroughputGauge from "../components/dashboard/ThroughputGauge.svelte";
  import LatencyDistributionChart from "../components/dashboard/LatencyDistributionChart.svelte";
  import ThroughputChart from "../components/dashboard/ThroughputChart.svelte";
  import LatencyPercentilesChart from "../components/dashboard/LatencyPercentilesChart.svelte";
  import ErrorRateChart from "../components/dashboard/ErrorRateChart.svelte";
  import SloComplianceCard from "../components/dashboard/SloComplianceCard.svelte";
  import ErrorBudgetBurnCard from "../components/dashboard/ErrorBudgetBurnCard.svelte";
  import ServiceLatencyRankCard from "../components/dashboard/ServiceLatencyRankCard.svelte";
  import ServiceThroughputCard from "../components/dashboard/ServiceThroughputCard.svelte";
  import AvailabilityTrendCard from "../components/dashboard/AvailabilityTrendCard.svelte";
  import TopEndpointsThroughputTable from "../components/dashboard/TopEndpointsThroughputTable.svelte";
  import SlowestEndpointsTable from "../components/dashboard/SlowestEndpointsTable.svelte";
  import ErrorHotspotsTable from "../components/dashboard/ErrorHotspotsTable.svelte";
  import type { DashboardHealth } from "../services/dashboard";

  const DASHBOARD_REFRESH_MIN_SPIN_MS = 700;
  const QUERY_REFRESH_MIN_SPIN_MS = 700;

  let activeTab: "logs" | "metriche" | "tracce" = "metriche";
  let dashboardLoaded = false;
  let lastQueryRefresh: Date | null = null;
  let initialTraceId: string | null = null;
  let forceMode: "manual" | null = null;
  let autoRun = false;
  let tabFromUrl = "";
  let tabFromUrlApplied = false;
  let dashboardRangePreset: DashboardRangePreset = "6h";
  let dashboardFromInput = "";
  let dashboardToInput = "";
  let dashboardRangeError = "";
  let dashboardRefreshSpinning = false;
  let dashboardRefreshToken = 0;
  let queryRefreshSpinning = false;
  let queryRefreshStartAt = 0;
  let queryRefreshWasLoading = false;
  let queryRefreshTimer: ReturnType<typeof setTimeout> | null = null;
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
    forceMode = mode === "manual" ? "manual" : null;
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

  $: if (dashboardRangePreset !== "custom") {
    dashboardRangeError = "";
  }

  $: if (
    dashboardRangePreset === "custom" &&
    (!dashboardFromInput || !dashboardToInput)
  ) {
    const defaults = defaultCustomRangeInputs();
    if (!dashboardFromInput) dashboardFromInput = defaults.fromInput;
    if (!dashboardToInput) dashboardToInput = defaults.toInput;
  }

  let lastRequest: import("../services/query").QueryRequest | null = null;
  const logsCursorByPage = new Map<number, string>();
  const tracesCursorByPage = new Map<number, string>();
  let pageSize = "100";
  const pageSizeOptions = ["25", "50", "100", "200"];
  $: dashboardTimeRangeOptions = getDashboardTimeRangeOptions($locale);

  async function handleRun(event: CustomEvent) {
    const { request } = event.detail;
    logsCursorByPage.clear();
    tracesCursorByPage.clear();
    lastRequest = createInitialRunRequest(request);
    await executeQuery(lastRequest);
    storeNextCursors({
      pagination: get(queryState).result?.pagination,
      page: 1,
      logsCursorByPage,
      tracesCursorByPage,
    });
    if (!get(queryState).error && get(queryState).result) {
      lastQueryRefresh = new Date();
    }
  }

  async function handlePageChange(
    signal: "logs" | "traces",
    nextPage: number,
  ) {
    if (!lastRequest) return;
    const updated = createPageChangeRequest({
      lastRequest,
      signal,
      nextPage,
      logsCursorByPage,
      tracesCursorByPage,
    });
    const page = updated.page ?? 1;
    lastRequest = updated;
    await executeQuery(updated, { retainResult: true });
    storeNextCursors({
      pagination: get(queryState).result?.pagination,
      page,
      logsCursorByPage,
      tracesCursorByPage,
    });
    if (!get(queryState).error && get(queryState).result) {
      lastQueryRefresh = new Date();
    }
  }

  async function handlePageSizeChange() {
    if (!lastRequest) return;
    logsCursorByPage.clear();
    tracesCursorByPage.clear();
    const updated = createPageSizeRequest(lastRequest, pageSize);
    lastRequest = updated;
    await executeQuery(updated, { retainResult: true });
    storeNextCursors({
      pagination: get(queryState).result?.pagination,
      page: 1,
      logsCursorByPage,
      tracesCursorByPage,
    });
    if (!get(queryState).error && get(queryState).result) {
      lastQueryRefresh = new Date();
    }
  }

  async function handleResultsPageSizeChange(event: CustomEvent) {
    const selected = String(event.detail?.size ?? pageSize);
    if (selected !== pageSize) {
      pageSize = selected;
    }
    await handlePageSizeChange();
  }

  function getDashboardWarningMessage(
    health: DashboardHealth | null | undefined,
    warnings: string[],
  ): string | null {
    if (health?.source === "stale_cache" && health.reason === "backend_pressure") {
      return t($locale, "dashboard.health.stalePressure");
    }
    if (health?.status === "degraded" && health.reason === "backend_pressure") {
      return t($locale, "dashboard.health.degradedPressure");
    }
    if (health?.source === "stale_cache") {
      return t($locale, "dashboard.health.stale");
    }
    if (health?.status === "degraded" && health.source === "empty") {
      return t($locale, "dashboard.health.rollupWarming");
    }
    if (health?.status === "partial") {
      return t($locale, "dashboard.health.partial");
    }
    if (warnings.length > 0) {
      return warnings.join(" · ");
    }
    return null;
  }

  function handleTabSelect(tab: "logs" | "metriche" | "tracce") {
    activeTab = tab;
    if (tab === "metriche" && !dashboardLoaded) {
      loadDashboardMetrics();
    }
  }

  async function loadDashboardMetrics() {
    const spinToken = ++dashboardRefreshToken;
    const spinStartedAt = Date.now();
    dashboardRefreshSpinning = true;

    const built = buildDashboardRequest({
      selectedService: $dashboardState.selectedService,
      rangePreset: dashboardRangePreset,
      fromInput: dashboardFromInput,
      toInput: dashboardToInput,
      locale: $locale,
    });

    if (built.error) {
      dashboardRangeError = built.error;
      return;
    }

    dashboardRangeError = "";

    try {
      await loadDashboard(built.request);
      dashboardLoaded = true;
      lastRefresh = new Date();
    } finally {
      const elapsed = Date.now() - spinStartedAt;
      const remaining = DASHBOARD_REFRESH_MIN_SPIN_MS - elapsed;
      if (remaining > 0) {
        await sleep(remaining);
      }
      if (spinToken === dashboardRefreshToken) {
        dashboardRefreshSpinning = false;
      }
    }
  }

  let lastRefresh: Date | null = null;

  $: if ($queryState.result && !lastQueryRefresh) {
    lastQueryRefresh = new Date();
  }

  async function handleRefresh() {
    await loadDashboardMetrics();
  }

  function handleDashboardRangePresetChange(event: Event) {
    const value = (event.target as HTMLSelectElement)
      .value as DashboardRangePreset;
    dashboardRangePreset = value;

    if (dashboardRangePreset !== "custom") {
      dashboardFromInput = "";
      dashboardToInput = "";
    }

    dashboardRangeError = "";
  }

  function handleDashboardServiceChange(event: Event) {
    const value = (event.target as HTMLSelectElement).value;
    selectDashboardService(value ? value : null, {
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
    dashboardRangePreset = "6h";
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

  $: {
    if ($queryState.loading && !queryRefreshWasLoading) {
      queryRefreshWasLoading = true;
      queryRefreshStartAt = Date.now();
      queryRefreshSpinning = true;
      if (queryRefreshTimer) {
        clearTimeout(queryRefreshTimer);
        queryRefreshTimer = null;
      }
    } else if (!$queryState.loading && queryRefreshWasLoading) {
      queryRefreshWasLoading = false;
      const elapsed = Date.now() - queryRefreshStartAt;
      const remaining = QUERY_REFRESH_MIN_SPIN_MS - elapsed;
      if (remaining <= 0) {
        queryRefreshSpinning = false;
      } else {
        queryRefreshTimer = setTimeout(() => {
          queryRefreshSpinning = false;
          queryRefreshTimer = null;
        }, remaining);
      }
    }
  }

  onDestroy(() => {
    if (queryRefreshTimer) {
      clearTimeout(queryRefreshTimer);
      queryRefreshTimer = null;
    }
  });

  $: activeDashboardFilters = countActiveDashboardFilters(
    dashboardRangePreset,
    $dashboardState.selectedService
  );

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
            ? t($locale, "home.logs")
            : activeTab === "metriche"
              ? t($locale, "home.metrics")
              : t($locale, "home.traces")}
        </h2>
        <p>
          {activeTab === "metriche"
            ? t($locale, "home.metricsSubtitle")
            : t($locale, "home.querySubtitle")}
        </p>
      </div>
      {#if activeTab === "metriche"}
        <div class="metrics-controls">
          <div class="refresh-controls">
            {#if lastRefresh}
              <span class="last-refresh">
                {t($locale, "home.lastRefresh", {
                  time: formatLastRefresh(lastRefresh, $locale),
                })}
              </span>
            {/if}
            <button
              class="refresh-btn"
              on:click={handleRefresh}
              disabled={$dashboardState.loading}
              title={t($locale, "home.refreshMetrics")}
              aria-label={t($locale, "home.refreshMetrics")}
            >
              <svg
                class="refresh-icon"
                class:spinning={dashboardRefreshSpinning}
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fill-rule="evenodd"
                  d="M15.312 11.424a5.5 5.5 0 01-9.201 2.466l-.312-.311h2.433a.75.75 0 000-1.5H3.989a.75.75 0 00-.75.75v4.242a.75.75 0 001.5 0v-2.43l.31.31a7 7 0 0011.712-3.138.75.75 0 00-1.449-.389zm1.23-7.424a.75.75 0 00-.75.75v2.43l-.31-.31A7 7 0 003.77 9.89a.75.75 0 101.45.388 5.5 5.5 0 019.201-2.466l.312.311h-2.433a.75.75 0 000 1.5h4.243a.75.75 0 00.75-.75V4.75a.75.75 0 00-.75-.75z"
                  clip-rule="evenodd"
                />
              </svg>
              {t($locale, "home.refresh")}
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
              {t($locale, "home.filters")}
              {#if activeDashboardFilters > 0}
                <span class="filters-badge">{activeDashboardFilters}</span>
              {/if}
            </button>
            {#if showDashboardFilters}
              <div class="filters-menu">
                <div class="filter-block">
                  <label class="filter-label" for="dashboard-service-dropdown"
                    >{t($locale, "home.service")}</label
                  >
                  <select
                    id="dashboard-service-dropdown"
                    class="filter-select"
                    on:change={handleDashboardServiceChange}
                    value={$dashboardState.selectedService || ""}
                  >
                    <option value="">{t($locale, "home.allServices")}</option>
                    {#each $servicesState.services as service}
                      <option value={service}>{service}</option>
                    {/each}
                  </select>
                </div>

                <div class="filter-block">
                  <span class="filter-label">{t($locale, "home.periodDateTime")}</span>
                  <select
                    class="filter-select"
                    on:change={handleDashboardRangePresetChange}
                    value={dashboardRangePreset}
                  >
                    {#each dashboardTimeRangeOptions as option}
                      <option value={option.value}>{option.label}</option>
                    {/each}
                  </select>
                </div>

                {#if dashboardRangePreset === "custom"}
                  <div class="filter-block">
                    <span class="filter-label">{t($locale, "home.customRange")}</span>
                  <div class="range-inputs">
                    <input
                      type="datetime-local"
                      bind:value={dashboardFromInput}
                      aria-label={t($locale, "home.fromDateTime")}
                    />
                    <input
                      type="datetime-local"
                      bind:value={dashboardToInput}
                      aria-label={t($locale, "home.toDateTime")}
                    />
                  </div>
                  </div>
                {/if}
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
                    {t($locale, "home.resetFilters")}
                  </button>
                  <button
                    class="apply-filters-btn"
                    type="button"
                    on:click={applyDashboardFilters}
                    disabled={$dashboardState.loading}
                  >
                    {t($locale, "common.apply")}
                  </button>
                </div>
              </div>
            {/if}
          </div>
        </div>
      {:else}
        <div class="metrics-controls">
          <div class="refresh-controls">
            <span class="last-refresh">
              {t($locale, "home.lastRefresh", {
                time: formatLastRefresh(lastQueryRefresh, $locale),
              })}
            </span>
            <button
              class="refresh-btn"
              on:click={handleRefreshQueryFilters}
              disabled={$queryState.loading}
              title={t($locale, "home.refreshResults")}
              aria-label={t($locale, "home.refreshResults")}
            >
              <svg
                class="refresh-icon"
                class:spinning={queryRefreshSpinning}
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
              {t($locale, "home.refresh")}
            </button>
          </div>
        </div>
      {/if}
    </div>

    <div class="results" class:dashboard-results={activeTab === "metriche"}>
      {#if activeTab === "metriche"}
        {#if $dashboardState.data}
          {@const dashboardWarning = getDashboardWarningMessage($dashboardState.data.health, $dashboardState.warnings)}
          {#if dashboardWarning}
            <div class="inline-warning">{dashboardWarning}</div>
          {/if}
          {#if $dashboardState.error}
            <div class="inline-warning">{$dashboardState.error}</div>
          {/if}
          <div class="dashboard-grid">
            {#each getOrderedChartSettings() as chartSetting}
              {#if chartSetting.enabled}
                {#if chartSetting.key === "apdex_gauge"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                  <ApdexGauge data={$dashboardState.data.satisfaction.apdex} />
                  </div>
                {:else if chartSetting.key === "error_rate_gauge"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                  <ErrorRateGauge
                    errorRate={$dashboardState.data.satisfaction.errorRate}
                    totalErrors={$dashboardState.data.satisfaction.throughput
                      .totalErrors}
                    totalRequests={$dashboardState.data.satisfaction.throughput
                      .totalRequests}
                  />
                  </div>
                {:else if chartSetting.key === "throughput_gauge"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                  <ThroughputGauge
                    data={$dashboardState.data.satisfaction.throughput}
                  />
                  </div>
                {:else if chartSetting.key === "latency_distribution"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                  <LatencyDistributionChart
                    data={$dashboardState.data.hotspots.latencyDistribution}
                  />
                  </div>
                {:else if chartSetting.key === "throughput_timeseries"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                  <ThroughputChart
                    data={$dashboardState.data.satisfaction.timeSeries}
                  />
                  </div>
                {:else if chartSetting.key === "latency_percentiles"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                  <LatencyPercentilesChart
                    data={$dashboardState.data.satisfaction.latencySeries}
                  />
                  </div>
                {:else if chartSetting.key === "error_rate_timeseries"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                  <ErrorRateChart
                    data={$dashboardState.data.satisfaction.errorRateSeries}
                  />
                  </div>
                {:else if chartSetting.key === "slowest_endpoints"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                  <SlowestEndpointsTable
                    data={$dashboardState.data.hotspots.slowestEndpoints}
                  />
                  </div>
                {:else if chartSetting.key === "top_endpoints_throughput"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                  <TopEndpointsThroughputTable
                    data={$dashboardState.data.hotspots.topEndpoints}
                  />
                  </div>
                {:else if chartSetting.key === "error_hotspots"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                  <ErrorHotspotsTable
                    data={$dashboardState.data.hotspots.errorHotspots}
                  />
                  </div>
                {:else if chartSetting.key === "slo_compliance"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                    <SloComplianceCard
                      data={$dashboardState.data.satisfaction.latencySeries}
                    />
                  </div>
                {:else if chartSetting.key === "error_budget_burn"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                    <ErrorBudgetBurnCard
                      data={$dashboardState.data.satisfaction.errorRateSeries}
                    />
                  </div>
                {:else if chartSetting.key === "service_latency_rank"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                    <ServiceLatencyRankCard
                      data={$dashboardState.data.hotspots.slowestEndpoints}
                    />
                  </div>
                {:else if chartSetting.key === "service_throughput"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                    <ServiceThroughputCard
                      data={$dashboardState.data.hotspots.topEndpoints}
                    />
                  </div>
                {:else if chartSetting.key === "availability_trend"}
                  <div
                    class="dashboard-item"
                    style={`grid-column:${(chartSetting.x ?? 0) + 1} / span ${chartSetting.w ?? 6};grid-row:${(chartSetting.y ?? 0) + 1} / span ${chartSetting.h ?? 3};`}
                  >
                    <AvailabilityTrendCard
                      data={$dashboardState.data.satisfaction.errorRateSeries}
                    />
                  </div>
                {/if}
              {/if}
            {/each}
          </div>
        {:else if $dashboardState.loading && !$dashboardState.data}
          <div class="status">{t($locale, "home.loadingDashboard")}</div>
        {:else if $dashboardState.error}
          <div class="status error">{$dashboardState.error}</div>
        {:else}
          <div class="status">{t($locale, "home.loadingMetrics")}</div>
        {/if}
      {:else if !$queryState.result}
        {#if $queryState.error}
          <div class="status error">{$queryState.error}</div>
        {:else}
          <div class="status">
            {$queryState.loading
              ? t($locale, "home.loadingResults")
              : t($locale, "home.startQuery")}
          </div>
        {/if}
      {:else}
        {#if $queryState.warnings.length > 0}
          <div class="inline-warning">{$queryState.warnings.join(" · ")}</div>
        {/if}
        {#if $queryState.error}
          <div class="inline-warning">{$queryState.error}</div>
        {/if}
        {#if $queryState.loading}
          <div class="status">{t($locale, "home.updating")}</div>
        {/if}
        {#if activeTab === "logs"}
          <LogResultsTable
            logs={$queryState.result.results.logs}
            pagination={$queryState.result.pagination?.logs ?? null}
            isLiveUpdate={$queryState.isLiveUpdate}
            lastUpdatedLabel={""}
            {pageSize}
            {pageSizeOptions}
            on:pageChange={(e) => handlePageChange("logs", e.detail.page)}
            on:pageSizeChange={handleResultsPageSizeChange}
          />
        {:else if activeTab === "tracce"}
          <TraceResultsList
            traces={$queryState.result.results.traces}
            pagination={$queryState.result.pagination?.traces ?? null}
            isLiveUpdate={$queryState.isLiveUpdate}
            lastUpdatedLabel={""}
            {pageSize}
            {pageSizeOptions}
            on:pageChange={(e) => handlePageChange("traces", e.detail.page)}
            on:pageSizeChange={handleResultsPageSizeChange}
          />
        {/if}
      {/if}
    </div>
  </section>

  {#if activeTab !== "metriche"}
    <aside class="filters">
      <div class="panel">
        <div class="filters-panel-header">
          <h3>{t($locale, "home.queryFilters")}</h3>
          <div class="query-header-actions">
            <button
              type="button"
              class="query-reset-btn"
              on:click={handleResetQueryFilters}
              title={t($locale, "home.resetAllFilters")}
            >
              {t($locale, "home.resetFilters")}
            </button>
          </div>
        </div>
        <QueryForm
          bind:this={queryFormRef}
          {activeTab}
          {initialTraceId}
          {forceMode}
          {autoRun}
          autoSearch={false}
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
    background: linear-gradient(135deg, var(--color-slate-50) 0%, var(--color-slate-100) 100%);
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
    border-bottom: 1px solid rgba(var(--rgb-slate-950), 0.06);
    flex-wrap: wrap;
  }

  .content-header h2 {
    margin: 0 0 6px 0;
    font-size: 24px;
    font-weight: 700;
    color: var(--color-slate-950);
  }

  .content-header p {
    margin: 0;
    color: var(--color-slate-500);
    font-size: 14px;
  }

  .results {
    flex: 1;
    background: white;
    border-radius: 16px;
    padding: 24px;
    box-shadow:
      0 1px 3px rgba(0, 0, 0, 0.05),
      0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(var(--rgb-slate-950), 0.06);
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
    grid-template-columns: repeat(6, minmax(0, 1fr));
    grid-auto-rows: minmax(120px, auto);
    gap: 20px;
    min-width: 0;
  }

  .dashboard-item {
    min-width: 0;
    height: 100%;
    display: flex;
    min-height: 0;
  }

  .dashboard-item :global(.gauge-card),
  .dashboard-item :global(.chart-card),
  .dashboard-item :global(.table-card) {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    box-sizing: border-box;
    min-height: 0;
  }

  .dashboard-item :global(.chart-container),
  .dashboard-item :global(.table-wrapper),
  .dashboard-item :global(.histogram),
  .dashboard-item :global(.breakdown) {
    flex: 1;
    min-height: 0;
  }

  .dashboard-item :global(.table-wrapper) {
    overflow: auto;
  }

  .dashboard-item :global(.empty) {
    min-height: 180px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .filters {
    background: white;
    border-left: 1px solid rgba(var(--rgb-slate-950), 0.06);
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
    color: var(--color-slate-500);
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
    align-items: flex-end;
    gap: 10px;
  }

  .query-reset-btn {
    padding: 7px 11px;
    border: 1px solid var(--color-slate-300);
    border-radius: 8px;
    background: var(--color-white);
    color: var(--color-slate-600);
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
    background: var(--color-slate-50);
    border-color: var(--color-slate-400);
    color: var(--color-slate-700);
  }

  .status {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 200px;
    color: var(--color-slate-400);
    font-size: 15px;
  }

  .status.error {
    color: var(--color-danger-500);
    background: rgba(239, 68, 68, 0.05);
    border-radius: 12px;
    padding: 16px;
  }

  .inline-warning {
    margin-bottom: 12px;
    padding: 10px 12px;
    border: 1px solid rgba(245, 158, 11, 0.35);
    background: rgba(245, 158, 11, 0.08);
    border-radius: 10px;
    color: #92400e;
    font-size: 12px;
    line-height: 1.4;
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
    border: 1px solid var(--color-slate-300);
    background: var(--color-white);
    color: var(--color-slate-700);
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition:
      background 0.15s ease,
      border-color 0.15s ease,
      color 0.15s ease,
      box-shadow 0.15s ease;
  }

  .filters-btn:hover {
    border-color: var(--color-slate-400);
    background: var(--color-slate-50);
  }

  .filters-btn.active {
    border-color: var(--color-primary-600);
    color: #4338ca;
    background: var(--color-primary-50);
  }

  .filters-btn:focus-visible {
    outline: none;
    box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.2);
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
    background: var(--color-primary-600);
    color: var(--color-white);
    font-size: 11px;
    font-weight: 700;
  }

  .filters-menu {
    position: absolute;
    top: calc(100% + 8px);
    right: 0;
    width: 340px;
    background: var(--color-white);
    border: 1px solid var(--color-slate-200);
    border-radius: 12px;
    box-shadow: 0 14px 35px rgba(var(--rgb-slate-950), 0.16);
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
    border-top: 1px solid var(--color-slate-100);
  }

  .filter-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--color-slate-500);
  }

  .filter-select {
    padding: 11px 12px;
    border-radius: 10px;
    border: 1px solid var(--color-slate-200);
    background: var(--color-white);
    font-size: 13px;
    color: var(--color-slate-950);
  }

  .range-inputs {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .range-inputs input {
    padding: 11px 12px;
    border-radius: 10px;
    border: 1px solid var(--color-slate-200);
    background: white;
    font-size: 13px;
    color: var(--color-slate-950);
  }

  .range-inputs input:focus {
    outline: none;
    border-color: var(--color-info-600);
    box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
  }

  .range-error {
    font-size: 12px;
    color: var(--color-danger-600);
    padding: 8px 10px;
    border-radius: 8px;
    background: var(--color-danger-50);
    border: 1px solid var(--color-danger-75);
  }

  .apply-filters-btn {
    padding: 9px 14px;
    border: none;
    border-radius: 8px;
    background: var(--color-primary-600);
    color: var(--color-white);
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition:
      background 0.15s ease,
      transform 0.15s ease,
      box-shadow 0.15s ease;
  }

  .apply-filters-btn:hover:not(:disabled) {
    background: var(--color-primary-700);
  }

  .apply-filters-btn:active:not(:disabled) {
    transform: scale(0.98);
  }

  .apply-filters-btn:focus-visible {
    outline: none;
    box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.2);
  }

  .apply-filters-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .filters-actions {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    margin-top: 4px;
    padding-top: 12px;
    border-top: 1px solid var(--color-slate-100);
  }

  .reset-filters-btn {
    padding: 9px 12px;
    border: 1px solid var(--color-slate-300);
    border-radius: 8px;
    background: var(--color-white);
    color: var(--color-slate-600);
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition:
      background 0.15s ease,
      border-color 0.15s ease,
      color 0.15s ease,
      box-shadow 0.15s ease;
  }

  .reset-filters-btn:hover:not(:disabled) {
    background: var(--color-slate-50);
    border-color: var(--color-slate-400);
    color: var(--color-slate-700);
  }

  .reset-filters-btn:focus-visible {
    outline: none;
    box-shadow: 0 0 0 3px rgba(148, 163, 184, 0.22);
  }

  .reset-filters-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .refresh-controls {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .last-refresh {
    font-size: 12px;
    color: var(--color-slate-400);
  }

  .refresh-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    height: 34px;
    padding: 8px 12px;
    box-sizing: border-box;
    background: var(--color-primary-600);
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition:
      background 0.15s ease,
      transform 0.15s ease,
      box-shadow 0.15s ease;
  }

  .refresh-btn:hover:not(:disabled) {
    background: var(--color-primary-700);
  }

  .refresh-btn:active:not(:disabled) {
    transform: scale(0.98);
  }

  .refresh-btn:focus-visible {
    outline: none;
    box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.2);
  }

  .refresh-btn:disabled {
    opacity: 0.6;
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

  @media (max-width: 1200px) {
    .dashboard {
      grid-template-columns: 1fr;
      height: auto;
      padding-left: 220px;
    }
    .filters {
      grid-column: 1;
      border-left: none;
      border-top: 1px solid rgba(var(--rgb-slate-950), 0.06);
    }
    .filters .panel {
      padding: 24px;
    }
  }


  @media (max-width: 900px) {
    .dashboard-grid {
      grid-template-columns: 1fr;
      grid-auto-rows: auto;
    }
    .dashboard-item {
      grid-column: 1 / -1 !important;
      grid-row: auto !important;
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
      grid-template-columns: repeat(6, minmax(0, 1fr));
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
