<script context="module" lang="ts">
  const queryFormStateStorageKey = "opendashly.queryForm.tabStates.v1";

  function loadPersistedTabStates(): Record<string, any> | null {
    if (typeof localStorage === "undefined") return null;
    try {
      const raw = localStorage.getItem(queryFormStateStorageKey);
      if (!raw) return null;
      const parsed = JSON.parse(raw);
      return parsed && typeof parsed === "object" ? parsed : null;
    } catch {
      return null;
    }
  }

  function storePersistedTabStates(states: Record<string, any>) {
    if (typeof localStorage === "undefined") return;
    try {
      localStorage.setItem(queryFormStateStorageKey, JSON.stringify(states));
    } catch {
      // Ignore storage errors: in-memory state still preserves filters during SPA navigation.
    }
  }

  let persistedStates: Record<string, any> | null = loadPersistedTabStates();
</script>

<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import {
    servicesState,
    selectService,
  } from "../lib/stores/query";
  import type { FilterItem, QueryRequest } from "../services/query";
  import FilterBuilder from "./FilterBuilder.svelte";
  import Modal from "./common/Modal.svelte";
  import ServiceDropdown from "./ServiceDropdown.svelte";
  import { getLocaleTag, locale, t } from "../lib/i18n";

  const dispatch = createEventDispatcher();

  type SearchMode = "manual";
  type TimeRangePreset = "5m" | "15m" | "30m" | "1h" | "6h" | "24h" | "7d" | "all" | "custom";

  export let activeTab: "logs" | "metriche" | "tracce";
  export let initialTraceId: string | null = null;
  export let forceMode: SearchMode | null = null;
  export let autoRun = false;
  export let autoSearch = true;

  interface TabState {
    searchMode: SearchMode;
    selectedRange: TimeRangePreset;
    fromInput: string;
    toInput: string;
    advancedFilters: FilterItem[];
    filterDurationOperator: string;
    filterDurationMs: string;
    traceErrorScope: "all" | "with_errors" | "without_errors";
    logTextSearch: string;
    traceSearch: string;
    traceSearchExact: boolean;
    selectedLogLevels: string[];
    selectedService: string;
  }

  const defaultState: TabState = {
    searchMode: "manual",
    selectedRange: "30m",
    fromInput: "",
    toInput: "",
    advancedFilters: [],
    filterDurationOperator: ">",
    filterDurationMs: "",
    traceErrorScope: "all",
    logTextSearch: "",
    traceSearch: "",
    traceSearchExact: false,
    selectedLogLevels: [],
    selectedService: "",
  };
  let tabStates: Record<string, TabState> = persistedStates || {
    logs: { ...defaultState },
    metriche: { ...defaultState },
    tracce: { ...defaultState },
  };
  persistedStates = tabStates;

  let searchMode: SearchMode = "manual";
  let selectedRange: TimeRangePreset = "30m";
  let fromInput = "";
  let toInput = "";
  let advancedFilters: FilterItem[] = [];
  let filterDurationOperator = ">";
  let filterDurationMs = "";
  let traceErrorScope: "all" | "with_errors" | "without_errors" = "all";
  let logTextSearch = "";
  let traceSearch = "";
  let traceSearchExact = false;
  let selectedLogLevels: string[] = [];
  let showLogLevelDropdown = false;
  let rangeError = "";
  let rangeSummary = "";

  let previousTab = "";
  let lastInitialTraceId = "";
  let suppressUrlSync = true;
  let autoSubmitReady = false;
  let suppressNextAutoSubmit = false;
  let autoSubmitTimer: ReturnType<typeof setTimeout> | null = null;
  let lastAutoSubmitKey = "";

  let showFilterModal = false;
  let showCustomRangeModal = false;
  let customFromInput = "";
  let customToInput = "";
  let activeAdvancedFiltersCount = 0;

  $: timeRangeOptions = [
    { value: "5m", label: t($locale, "range.last5m") },
    { value: "15m", label: t($locale, "range.last15m") },
    { value: "30m", label: t($locale, "range.last30m") },
    { value: "1h", label: t($locale, "range.last1h") },
    { value: "6h", label: t($locale, "range.last6h") },
    { value: "24h", label: t($locale, "range.last24h") },
    { value: "7d", label: t($locale, "range.last7d") },
    { value: "all", label: t($locale, "range.all") },
  ] as Array<{ value: TimeRangePreset; label: string }>;

  const availableLogLevels = ["TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"];

  const logLevelPalette: Record<string, { color: string; bg: string }> = {
    TRACE: { color: "var(--color-slate-500)", bg: "var(--color-slate-100)" },
    DEBUG: { color: "var(--color-primary-400)", bg: "var(--color-primary-25)" },
    INFO: { color: "var(--color-info-500)", bg: "var(--color-info-100)" },
    WARN: { color: "var(--color-warning-500)", bg: "var(--color-warning-100)" },
    ERROR: { color: "var(--color-danger-500)", bg: "var(--color-danger-100)" },
    FATAL: { color: "var(--color-danger-600)", bg: "var(--color-danger-50)" },
  };

  const presetMinutes: Record<Exclude<TimeRangePreset, "custom">, number | null> = {
    "5m": 5,
    "15m": 15,
    "30m": 30,
    "1h": 60,
    "6h": 360,
    "24h": 1440,
    "7d": 10080,
    all: null,
  };

  function normalizeAllSelection(value: unknown): string {
    const trimmed = String(value ?? "").trim();
    return trimmed === "Tutti" || trimmed.toLowerCase() === "all" ? "" : trimmed;
  }

  $: if (activeTab && activeTab !== previousTab) {
    const oldTab = previousTab;
    if (oldTab) saveState(oldTab);
    loadState(activeTab);
    if (oldTab) {
      suppressNextAutoSubmit = true;
    }
    previousTab = activeTab;
    persistedStates = tabStates;
  }

  $: if (initialTraceId && initialTraceId !== lastInitialTraceId) {
    lastInitialTraceId = initialTraceId;
    applyTraceIdOverride(initialTraceId);
  }

  $: syncUrlWithState();
  $: activeAdvancedFiltersCount = Array.isArray(advancedFilters) ? advancedFilters.length : 0;

  function saveState(tab: string) {
    tabStates[tab] = {
      searchMode,
      selectedRange,
      fromInput,
      toInput,
      advancedFilters,
      filterDurationOperator,
      filterDurationMs,
      traceErrorScope,
      logTextSearch,
      traceSearch,
      traceSearchExact,
      selectedLogLevels,
      selectedService: normalizeAllSelection($servicesState.selectedService),
    };
    persistedStates = tabStates;
    storePersistedTabStates(tabStates);
  }

  function loadState(tab: string) {
    const state = tabStates[tab] || { ...defaultState };
    searchMode = state.searchMode || "manual";
    selectedRange = state.selectedRange || "30m";
    fromInput = state.fromInput || "";
    toInput = state.toInput || "";
    advancedFilters = Array.isArray(state.advancedFilters) ? state.advancedFilters : [];
    filterDurationOperator = state.filterDurationOperator || ">";
    filterDurationMs = state.filterDurationMs || "";
    traceErrorScope = state.traceErrorScope || "all";
    logTextSearch = state.logTextSearch || "";
    traceSearch = state.traceSearch || "";
    traceSearchExact = !!state.traceSearchExact;
    selectedLogLevels = Array.isArray(state.selectedLogLevels)
      ? state.selectedLogLevels.filter((value) => availableLogLevels.includes(value))
      : [];
    selectService(normalizeAllSelection(state.selectedService));
    rangeError = "";
  }

  function getEffectiveAdvancedFilters(): FilterItem[] {
    return advancedFilters
      .map((f) => ({
        ...f,
        key: String(f.key ?? "").trim(),
        value: String(f.value ?? "").trim(),
      }))
      .filter((f) => f.key && f.value);
  }

  function openFilters() {
    showFilterModal = true;
  }

  function closeFilters() {
    showFilterModal = false;
  }

  function handleAdvancedFiltersChange(event: CustomEvent<FilterItem[]>) {
    advancedFilters = Array.isArray(event.detail) ? [...event.detail] : [...advancedFilters];
  }

  function applyAdvancedFilters() {
    advancedFilters = getEffectiveAdvancedFilters();
    closeFilters();
  }

  function toIso(value: string): string {
    if (!value) return "";
    const d = new Date(value);
    return Number.isNaN(d.getTime()) ? "" : d.toISOString();
  }

  function toLocalInputValue(date: Date): string {
    const pad = (n: number) => String(n).padStart(2, "0");
    const year = date.getFullYear();
    const month = pad(date.getMonth() + 1);
    const day = pad(date.getDate());
    const hours = pad(date.getHours());
    const minutes = pad(date.getMinutes());
    return `${year}-${month}-${day}T${hours}:${minutes}`;
  }

  function formatRangeDateTime(value: string): string {
    if (!value) return "";
    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
      return value.replace("T", " ");
    }
    return parsed.toLocaleString(getLocaleTag($locale), {
      day: "2-digit",
      month: "short",
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  function formatRangeDate(date: Date): string {
    return date.toLocaleString(getLocaleTag($locale), {
      day: "2-digit",
      month: "short",
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  function toggleLogLevel(level: string) {
    if (selectedLogLevels.includes(level)) {
      selectedLogLevels = selectedLogLevels.filter((value) => value !== level);
      return;
    }
    selectedLogLevels = [...selectedLogLevels, level];
  }

  function resetLogLevels() {
    selectedLogLevels = [];
  }

  function getLogLevelSummary() {
    if (selectedLogLevels.length === 0) {
      return t($locale, "query.allLevels");
    }
    return selectedLogLevels.join(", ");
  }

  function getLogLevelStyle(level: string) {
    const palette = logLevelPalette[level] || { color: "var(--color-slate-950)", bg: "var(--color-slate-50)" };
    return `background: ${palette.bg}; color: ${palette.color}`;
  }

  function handleWindowClick(event: MouseEvent) {
    const target = event.target as HTMLElement;
    if (!target.closest(".log-level-multi")) {
      showLogLevelDropdown = false;
    }
  }

  function openCustomRangeModal() {
    const now = new Date();
    const defaultFrom = new Date(now.getTime() - 30 * 60 * 1000);
    customFromInput = fromInput || toLocalInputValue(defaultFrom);
    customToInput = toInput || toLocalInputValue(now);
    showCustomRangeModal = true;
  }

  function closeCustomRangeModal() {
    showCustomRangeModal = false;
  }

  function applyCustomRange() {
    const fromIso = toIso(customFromInput);
    const toIsoValue = toIso(customToInput);
    if (!fromIso || !toIsoValue) {
      rangeError = t($locale, "range.errorInvalidDate");
      return;
    }
    if (new Date(toIsoValue).getTime() <= new Date(fromIso).getTime()) {
      rangeError = t($locale, "range.errorEndBeforeStart");
      return;
    }
    fromInput = customFromInput;
    toInput = customToInput;
    selectedRange = "custom";
    rangeError = "";
    showCustomRangeModal = false;
    if (autoSearch) {
      submit();
    }
  }

  function handleRangeChange(event: Event) {
    const value = (event.currentTarget as HTMLSelectElement).value as TimeRangePreset;
    if (value === "custom") {
      selectedRange = "custom";
      openCustomRangeModal();
      return;
    }
    selectedRange = value;
    rangeError = "";
    if (autoSearch) {
      submit();
    }
  }

  $: {
    if (selectedRange === "custom" || showCustomRangeModal) {
      if (fromInput && toInput) {
        rangeSummary = `${formatRangeDateTime(fromInput)} → ${formatRangeDateTime(toInput)}`;
      } else {
        rangeSummary = t($locale, "query.selectCustomRange");
      }
    } else {
      const minutes = presetMinutes[selectedRange as Exclude<TimeRangePreset, "custom">];
      const now = new Date();
      if (minutes === null) {
        rangeSummary = `${t($locale, "query.fromEver")} → ${formatRangeDate(now)}`;
      } else {
        const from = new Date(now.getTime() - minutes * 60 * 1000);
        rangeSummary = `${formatRangeDate(from)} → ${formatRangeDate(now)}`;
      }
    }
  }

  function buildTimeRange(): { from: string; to: string } | null {
    const zeroTime = "1970-01-01T00:00:00Z";
    const nowIso = new Date().toISOString();

    if (traceSearchExact) {
      return { from: zeroTime, to: nowIso };
    }

    if (selectedRange === "custom") {
      const fromIso = toIso(fromInput);
      const toIsoValue = toIso(toInput);
      if (!fromIso || !toIsoValue) {
        rangeError = t($locale, "range.errorInvalidDate");
        return null;
      }
      if (new Date(toIsoValue).getTime() <= new Date(fromIso).getTime()) {
        rangeError = t($locale, "range.errorEndBeforeStart");
        return null;
      }
      rangeError = "";
      return { from: fromIso, to: toIsoValue };
    }

    const minutes = presetMinutes[selectedRange as Exclude<TimeRangePreset, "custom">];
    if (minutes === null) {
      rangeError = "";
      return { from: zeroTime, to: nowIso };
    }

    const now = new Date();
    const from = new Date(now.getTime() - minutes * 60 * 1000);
    rangeError = "";
    return { from: from.toISOString(), to: now.toISOString() };
  }

  function handleManualEnter(event: KeyboardEvent) {
    if (showFilterModal || showCustomRangeModal) return;
    if (event.key !== "Enter") return;
    if (event.shiftKey || event.ctrlKey || event.metaKey || event.altKey) return;
    if (event.isComposing) return;
    event.preventDefault();
    submit();
  }

  function queueAutoSubmit(delayMs = 1000) {
    if (autoSubmitTimer) {
      clearTimeout(autoSubmitTimer);
    }
    autoSubmitTimer = setTimeout(() => {
      submit();
    }, delayMs);
  }

  function applyTraceIdOverride(traceId: string) {
    if (!traceId) return;
    selectService("");
    selectedLogLevels = [];
    advancedFilters = [];
    filterDurationOperator = ">";
    filterDurationMs = "";
    traceErrorScope = "all";
    traceSearch = traceId;
    traceSearchExact = true;
    selectedRange = "all";
    fromInput = "";
    toInput = "";
    if (autoRun) {
      submit();
    }
  }

  function clearTraceIdFromUrl() {
    const url = new URL($page.url);
    url.searchParams.delete("traceId");
    url.searchParams.delete("autorun");
    if (url.searchParams.get("tab") === "tracce") {
      url.searchParams.delete("tab");
    }
    goto(url.pathname + url.search, { replaceState: true });
  }

  function applyUrlParams() {
    const params = $page.url.searchParams;

    const rangeParam = params.get("range");
    const allowed: TimeRangePreset[] = ["5m", "15m", "30m", "1h", "6h", "24h", "7d", "all", "custom"];
    if (rangeParam && allowed.includes(rangeParam as TimeRangePreset)) {
      selectedRange = rangeParam as TimeRangePreset;
    }

    const serviceParam = params.get("service");
    if (serviceParam) {
      selectService(serviceParam);
    }

    const severityParam = params.get("severity");
    if (severityParam) {
      selectedLogLevels = severityParam
        .split(",")
        .map((value) => value.trim().toUpperCase())
        .filter((value) => availableLogLevels.includes(value));
    }

    const traceParam = params.get("traceId");
    if (traceParam) {
      applyTraceIdOverride(traceParam);
    }

    const fromParam = params.get("from");
    const toParam = params.get("to");
    if (fromParam) fromInput = fromParam;
    if (toParam) toInput = toParam;
  }

  function syncUrlWithState() {
    if (suppressUrlSync) return;
    const params = new URLSearchParams();
    params.set("tab", activeTab);

    const selectedService = normalizeAllSelection($servicesState.selectedService);
    if (selectedService) {
      params.set("service", selectedService);
    }
    if (selectedLogLevels.length > 0) {
      params.set("severity", selectedLogLevels.join(","));
    }

    params.set("range", selectedRange);
    if (selectedRange === "custom") {
      if (fromInput) params.set("from", fromInput);
      if (toInput) params.set("to", toInput);
    }

    if (traceSearch) {
      params.set("traceId", traceSearch);
    }

    const current = $page.url.searchParams.toString();
    const next = params.toString();
    if (current !== next) {
      goto(`${$page.url.pathname}?${next}`, { replaceState: true });
    }
  }

  function signalsForActiveTab(): string[] {
    if (activeTab === "logs") return ["logs"];
    if (activeTab === "tracce") return ["traces"];
    return ["logs", "traces"];
  }

  function submit() {
    saveState(activeTab);
    const selectedService = normalizeAllSelection($servicesState.selectedService);
    const serviceFilter = selectedService ? { "service.name": selectedService } : {};

    const manualFilters: Record<string, string> = { ...serviceFilter };
    if (activeTab === "logs" && selectedLogLevels.length > 0) {
      manualFilters["severity"] = selectedLogLevels.join(",");
    }

    const timeRange = buildTimeRange();
    if (!timeRange) {
      return;
    }

    let finalFilterList = [...getEffectiveAdvancedFilters()];
    for (const [k, v] of Object.entries(manualFilters)) {
      if (!finalFilterList.some((f) => f.key === k)) {
        finalFilterList.push({
          connector: "AND",
          key: k,
          operator: "=",
          value: v,
        });
      }
    }

    const durationRaw = String(filterDurationMs ?? "").trim();
    if (activeTab === "tracce" && durationRaw) {
      const durationNumber = Number(durationRaw);
      if (Number.isFinite(durationNumber) && durationNumber >= 0) {
        const durationValue = String(durationNumber);
        if (!finalFilterList.some((f) => f.key === "duration_ms")) {
          finalFilterList.push({
            connector: "AND",
            key: "duration_ms",
            operator: filterDurationOperator || ">",
            value: durationValue,
          });
        }
      }
    }

    if (activeTab === "tracce" && traceErrorScope !== "all") {
      if (!finalFilterList.some((f) => f.key === "trace_error_scope")) {
        finalFilterList.push({
          connector: "AND",
          key: "trace_error_scope",
          operator: "=",
          value: traceErrorScope,
        });
      }
    }

    const traceSearchTerm = activeTab === "tracce" ? String(traceSearch || "").trim() : "";
    if (activeTab === "tracce" && traceSearchTerm) {
      if (traceSearchExact) {
        if (!finalFilterList.some((f) => f.key === "trace_id" && f.value === traceSearchTerm)) {
          finalFilterList.push({
            connector: "AND",
            key: "trace_id",
            operator: "=",
            value: traceSearchTerm,
          });
        }
      } else if (!finalFilterList.some((f) => f.key === "trace_or_span" && f.value === traceSearchTerm)) {
        finalFilterList.push({
          connector: "AND",
          key: "trace_or_span",
          operator: "contains",
          value: traceSearchTerm,
        });
      }
    }

    const logTextTerm = activeTab === "logs" ? String(logTextSearch || "").trim() : "";
    if (logTextTerm) {
      if (!finalFilterList.some((f) => f.key === "body" && f.operator === "contains" && f.value === logTextTerm)) {
        finalFilterList.push({
          connector: "AND",
          key: "body",
          operator: "contains",
          value: logTextTerm,
        });
      }
    }

    finalFilterList = finalFilterList.filter((f) => f.key && f.value);

    dispatch("run", {
      request: {
        signals: signalsForActiveTab(),
        timeRange,
        filters: manualFilters,
        filterList: finalFilterList,
        page: 1,
        limit: 100,
      } as QueryRequest,
      autoRefreshSeconds: null,
      autoRefreshRangeMinutes: null,
    });
  }

  $: if (!traceSearch && $page.url.searchParams.get("traceId")) {
    clearTraceIdFromUrl();
  }

  $: if (autoSearch && autoSubmitReady) {
    const autoSubmitKey = JSON.stringify({
      tab: activeTab,
      selectedRange,
      fromInput,
      toInput,
      service: $servicesState.selectedService,
      level: $servicesState.selectedLogLevel,
      levels: selectedLogLevels,
      traceSearch,
      traceSearchExact,
      filterDurationOperator,
      filterDurationMs,
      traceErrorScope,
      logTextSearch,
      advancedFilters: getEffectiveAdvancedFilters(),
    });
    if (autoSubmitKey !== lastAutoSubmitKey) {
      lastAutoSubmitKey = autoSubmitKey;
      if (suppressNextAutoSubmit) {
        suppressNextAutoSubmit = false;
      } else {
        queueAutoSubmit();
      }
    }
  }

  function persistCurrentFilters() {
    if (activeTab) {
      saveState(activeTab);
    }
  }

  onMount(() => {
    loadState(activeTab);
    applyUrlParams();
    if (forceMode) {
      searchMode = forceMode;
    }
    if (initialTraceId) {
      traceSearch = initialTraceId;
      traceSearchExact = true;
      lastInitialTraceId = initialTraceId;
      selectedRange = "all";
      fromInput = "";
      toInput = "";
    }
    dispatch("modeChange", { mode: "manual" });
    const handleBeforeUnload = () => persistCurrentFilters();
    window.addEventListener("beforeunload", handleBeforeUnload);
    setTimeout(() => {
      suppressUrlSync = false;
      autoSubmitReady = autoSearch;
      if (autoRun) {
        submit();
      }
    }, 100);
    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  });

  onDestroy(() => {
    persistCurrentFilters();
  });

  export function resetFiltersToDefault() {
    searchMode = "manual";
    selectedRange = defaultState.selectedRange;
    fromInput = defaultState.fromInput;
    toInput = defaultState.toInput;
    logTextSearch = defaultState.logTextSearch;
    traceSearch = "";
    traceSearchExact = false;
    filterDurationOperator = defaultState.filterDurationOperator;
    filterDurationMs = defaultState.filterDurationMs;
    traceErrorScope = defaultState.traceErrorScope;
    advancedFilters = [];
    showFilterModal = false;
    showCustomRangeModal = false;
    rangeError = "";
    selectService("");
    selectedLogLevels = [];
    tabStates[activeTab] = { ...defaultState };
    persistedStates = tabStates;
    storePersistedTabStates(tabStates);
    dispatch("modeChange", { mode: "manual" });
    submit();
  }

  export function refreshCurrentQuery() {
    submit();
  }

  $: if (!showCustomRangeModal && autoSubmitTimer && !autoSubmitReady) {
    clearTimeout(autoSubmitTimer);
    autoSubmitTimer = null;
  }
</script>

<svelte:window on:click={handleWindowClick} />

<div class="query-form">
  <div class="query-service-filter">
    <ServiceDropdown />
  </div>

  {#if activeTab === "logs"}
    <div class="log-level-filter">
      <label for="log-level-trigger">{t($locale, "query.logLevel")}</label>
      <div class="log-level-multi">
        <button
          id="log-level-trigger"
          type="button"
          class="log-level-trigger"
          class:open={showLogLevelDropdown}
          on:click={() => (showLogLevelDropdown = !showLogLevelDropdown)}
        >
          <span class="trigger-values">
            {#if selectedLogLevels.length === 0}
              <span class="trigger-placeholder">{getLogLevelSummary()}</span>
            {:else}
              {#each selectedLogLevels as level}
                <span class="level-chip" style={getLogLevelStyle(level)}>{level}</span>
              {/each}
            {/if}
          </span>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="6 9 12 15 18 9"></polyline>
          </svg>
        </button>

        {#if showLogLevelDropdown}
          <div class="log-level-menu">
            {#each availableLogLevels as level}
              <label class="log-level-option" for={`log-level-${level}`}>
                <input
                  id={`log-level-${level}`}
                  type="checkbox"
                  checked={selectedLogLevels.includes(level)}
                  on:change={() => toggleLogLevel(level)}
                />
                <span class="level-chip" style={getLogLevelStyle(level)}>{level}</span>
              </label>
            {/each}
            <button type="button" class="btn-secondary clear-levels" on:click={resetLogLevels}>{t($locale, "query.resetLevels")}</button>
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <div class="time-range-block">
    <label for="time-range-select">{t($locale, "query.timeRange")}</label>
    <div class="time-range-controls">
      <select id="time-range-select" bind:value={selectedRange} on:change={handleRangeChange}>
        {#each timeRangeOptions as option}
          <option value={option.value}>{option.label}</option>
        {/each}
        <option value="custom">{t($locale, "range.custom")}</option>
      </select>
      {#if selectedRange === "custom"}
        <button type="button" class="btn-secondary edit-custom-range" on:click={openCustomRangeModal}>
          {t($locale, "query.editCustomRange")}
        </button>
      {/if}
    </div>
    <p class="time-range-summary">{rangeSummary}</p>
    {#if rangeError}
      <p class="error">{rangeError}</p>
    {/if}
  </div>

  <div class="manual-filters">
    {#if activeTab === "tracce"}
      <div class="filter-field">
        <label for="filter-trace-search">{t($locale, "query.textSearch")}</label>
        <input
          id="filter-trace-search"
          type="text"
          placeholder={t($locale, "query.traceSearchPlaceholder")}
          bind:value={traceSearch}
          on:input={() => {
            traceSearchExact = false;
          }}
          on:keydown={handleManualEnter}
        />
      </div>
      <div class="duration-row">
        <div class="filter-field compact operator">
          <label for="filter-duration-op">{t($locale, "query.duration")}</label>
          <select id="filter-duration-op" bind:value={filterDurationOperator} on:keydown={handleManualEnter}>
            <option value=">">&gt;</option>
            <option value="<">&lt;</option>
          </select>
        </div>
        <div class="filter-field compact">
          <label for="filter-duration-ms">{t($locale, "query.durationMs")}</label>
          <input
            id="filter-duration-ms"
            type="number"
            min="0"
            step="1"
            placeholder="300"
            bind:value={filterDurationMs}
            on:keydown={handleManualEnter}
          />
        </div>
      </div>
      <div class="filter-field">
        <p class="group-label">{t($locale, "query.traceErrors")}</p>
        <div class="trace-error-scope" role="group" aria-label={t($locale, "query.traceErrorsAria")}>
          <button type="button" class:active={traceErrorScope === "all"} on:click={() => (traceErrorScope = "all")}>{t($locale, "query.traceErrorsAll")}</button>
          <button type="button" class:active={traceErrorScope === "with_errors"} on:click={() => (traceErrorScope = "with_errors")}>{t($locale, "query.traceErrorsWith")}</button>
          <button type="button" class:active={traceErrorScope === "without_errors"} on:click={() => (traceErrorScope = "without_errors")}>{t($locale, "query.traceErrorsWithout")}</button>
        </div>
      </div>
    {/if}

    {#if activeTab === "logs"}
      <div class="filter-field">
        <label for="filter-log-text-search">{t($locale, "query.textSearch")}</label>
        <input
          id="filter-log-text-search"
          type="text"
          placeholder={t($locale, "query.logSearchPlaceholder")}
          bind:value={logTextSearch}
          on:keydown={handleManualEnter}
        />
      </div>

      <div class="advanced-filters-trigger">
        <button
          type="button"
          class="btn-secondary advanced-filters-btn"
          on:click={openFilters}
          aria-label={t($locale, "query.advancedFiltersAria", { count: activeAdvancedFiltersCount })}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"></polygon></svg
          >
          <span>{t($locale, "query.advancedFilters")}</span>
          {#if activeAdvancedFiltersCount > 0}
            <span class="advanced-filter-count-badge" aria-label={t($locale, "query.activeAdvancedFilters", { count: activeAdvancedFiltersCount })}>{activeAdvancedFiltersCount}</span>
          {/if}
        </button>
      </div>

      <Modal open={showFilterModal} title={t($locale, "query.advancedFiltersTitle")} on:close={closeFilters}>
        <FilterBuilder bind:filters={advancedFilters} on:change={handleAdvancedFiltersChange} />
        <div class="modal-actions">
          <button class="btn-primary" on:click={applyAdvancedFilters}>{t($locale, "query.applyFilters")}</button>
        </div>
      </Modal>
    {/if}
  </div>

  <Modal open={showCustomRangeModal} title={t($locale, "query.customTimeRangeTitle")} on:close={closeCustomRangeModal}>
    <div class="custom-range-grid">
      <div class="filter-field">
        <label for="custom-from">{t($locale, "query.from")}</label>
        <input id="custom-from" type="datetime-local" bind:value={customFromInput} />
      </div>
      <div class="filter-field">
        <label for="custom-to">{t($locale, "query.to")}</label>
        <input id="custom-to" type="datetime-local" bind:value={customToInput} />
      </div>
    </div>
    <div class="modal-actions">
      <button class="btn-secondary" on:click={closeCustomRangeModal}>{t($locale, "common.cancel")}</button>
      <button class="btn-primary" on:click={applyCustomRange}>{t($locale, "common.apply")}</button>
    </div>
  </Modal>
</div>

<style>
  .query-form {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .query-service-filter {
    margin-top: -8px;
    margin-bottom: 4px;
    padding-bottom: 14px;
    border-bottom: 1px solid rgba(var(--rgb-slate-950), 0.08);
  }

  .log-level-filter {
    margin-bottom: 6px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .log-level-multi {
    position: relative;
  }

  .log-level-trigger {
    width: 100%;
    min-height: 42px;
    padding: 0 12px;
    border: 1px solid var(--color-slate-200);
    border-radius: 10px;
    background: var(--color-slate-50);
    color: var(--color-slate-950);
    font-size: 13px;
    font-weight: 500;
    display: flex;
    align-items: center;
    justify-content: space-between;
    cursor: pointer;
  }

  .trigger-values {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    min-width: 0;
  }

  .trigger-placeholder {
    color: var(--color-slate-500);
    font-weight: 500;
    font-size: 13px;
  }

  .log-level-trigger.open,
  .log-level-trigger:focus {
    outline: none;
    border-color: var(--color-primary-600);
    box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.1);
    background: white;
  }

  .log-level-menu {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    right: 0;
    z-index: 20;
    background: white;
    border: 1px solid var(--color-slate-200);
    border-radius: 10px;
    box-shadow: 0 8px 20px rgba(var(--rgb-slate-950), 0.12);
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .log-level-option {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0;
    font-size: 12px;
    font-weight: 600;
    color: var(--color-slate-700);
    text-transform: none;
    letter-spacing: 0;
    cursor: pointer;
  }

  .level-chip {
    display: inline-flex;
    align-items: center;
    padding: 4px 8px;
    border-radius: 6px;
    font-size: 11px;
    font-weight: 700;
    line-height: 1;
  }

  .log-level-option input {
    width: 14px;
    height: 14px;
    min-height: 14px;
    margin: 0;
    padding: 0;
  }

  .clear-levels {
    margin-top: 4px;
    width: 100%;
  }

  .time-range-block {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
    border: 1px solid var(--color-slate-200);
    border-radius: 12px;
    background: var(--color-slate-50);
  }

  .time-range-controls {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 8px;
    align-items: center;
  }

  .time-range-controls:has(.edit-custom-range) {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .edit-custom-range {
    padding: 10px 14px;
    font-size: 13px;
    white-space: nowrap;
  }

  .time-range-summary {
    margin: 0;
    font-size: 12px;
    color: var(--color-slate-500);
    text-align: center;
  }

  .manual-filters {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .filter-field {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-bottom: 8px;
  }

  .group-label {
    margin: 0 0 4px 0;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-slate-500);
  }

  label {
    display: block;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-slate-500);
    margin-bottom: 4px;
  }

  input,
  select {
    width: 100%;
    padding: 12px 14px;
    font-size: 13px;
    border: 1px solid var(--color-slate-200);
    border-radius: 10px;
    background: var(--color-slate-50);
    color: var(--color-slate-950);
  }

  input:focus,
  select:focus {
    outline: none;
    border-color: var(--color-primary-600);
    box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.1);
    background: white;
  }

  .duration-row {
    display: grid;
    grid-template-columns: 110px minmax(0, 1fr);
    gap: 12px;
    align-items: end;
  }

  .trace-error-scope {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    background: var(--color-slate-100);
    border: 1px solid var(--color-slate-200);
    border-radius: 10px;
    padding: 4px;
    gap: 4px;
  }

  .trace-error-scope button {
    border: none;
    background: transparent;
    color: var(--color-slate-600);
    border-radius: 8px;
    font-size: 12px;
    font-weight: 600;
    padding: 8px 10px;
    cursor: pointer;
  }

  .trace-error-scope button.active {
    background: white;
    color: var(--color-slate-950);
    box-shadow: 0 2px 8px rgba(var(--rgb-slate-950), 0.12);
  }

  .advanced-filters-trigger {
    margin-top: 6px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .advanced-filters-trigger .btn-secondary {
    width: 100%;
  }

  .advanced-filter-count-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 20px;
    height: 20px;
    padding: 0 7px;
    border-radius: 999px;
    background: linear-gradient(180deg, #fde68a 0%, #facc15 100%);
    color: #713f12;
    border: 1px solid #f59e0b;
    font-size: 12px;
    font-weight: 800;
    line-height: 1;
    box-shadow:
      inset 0 1px 0 rgba(255, 255, 255, 0.55),
      0 1px 3px rgba(146, 64, 14, 0.22);
  }

  .btn-secondary {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 10px 14px;
    background: white;
    border: 1px solid var(--color-slate-300);
    border-radius: 8px;
    color: var(--color-slate-700);
    font-weight: 500;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-secondary:hover {
    background: var(--color-slate-50);
    border-color: var(--color-slate-400);
  }

  .btn-primary {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 12px 24px;
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
    color: white;
    border: none;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
    box-shadow: 0 4px 12px rgba(var(--rgb-primary-600), 0.3);
  }

  .btn-primary:hover {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(var(--rgb-primary-600), 0.35);
  }

  .badge {
    background: var(--color-primary-600);
    color: white;
    font-size: 11px;
    padding: 2px 6px;
    border-radius: 99px;
    font-weight: 700;
  }

  .error {
    margin: 0;
    color: var(--color-danger-700);
    font-size: 12px;
    font-weight: 600;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 24px;
    padding-top: 20px;
    border-top: 1px solid var(--color-slate-200);
  }

  .custom-range-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }

  @media (max-width: 760px) {
    .custom-range-grid {
      grid-template-columns: 1fr;
    }

    .duration-row {
      grid-template-columns: 1fr;
    }
  }
</style>
