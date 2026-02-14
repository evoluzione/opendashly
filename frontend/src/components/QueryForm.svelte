<script context="module" lang="ts">
  let persistedStates: any = null;
</script>

<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import {
    servicesState,
    selectService,
    selectLogLevel,
  } from "../lib/stores/query";
  import { generateSmartQuery } from "../services/query";
  import type { QueryRequest } from "../services/query";
  import { getAISettings } from "../services/settings";
  import FilterBuilder from "./FilterBuilder.svelte";
  import Modal from "./common/Modal.svelte";
  import type { FilterItem } from "../services/query";
  import LogLevelSelector from "./LogLevelSelector.svelte";
  import ServiceDropdown from "./ServiceDropdown.svelte";

  const dispatch = createEventDispatcher();

  type SearchMode = "auto" | "manual" | "smart";

  export let activeTab: "logs" | "metriche" | "tracce";
  export let initialTraceId: string | null = null;
  export let forceMode: SearchMode | null = null;
  export let autoRun = false;

  interface TabState {
    searchMode: SearchMode;
    fromInput: string;
    toInput: string;
    autoRangeMinutes: number | null;
    autoRefreshSeconds: number | null;
    smartPrompt: string;
    smartError: string;
    smartRequest: QueryRequest | null;
    advancedFilters: FilterItem[];
    filterDurationOperator: string;
    filterDurationMs: string;
    traceErrorScope: "all" | "with_errors" | "without_errors";
  }

  const defaultState: TabState = {
    searchMode: "auto",
    fromInput: "",
    toInput: "",
    autoRangeMinutes: null,
    autoRefreshSeconds: 10,
    smartPrompt: "",
    smartError: "",
    smartRequest: null,
    advancedFilters: [],
    filterDurationOperator: ">",
    filterDurationMs: "",
    traceErrorScope: "all",
  };

  let tabStates: Record<string, TabState> = persistedStates || {
    logs: { ...defaultState },
    metriche: { ...defaultState },
    tracce: { ...defaultState },
  };
  // Sincronizza il riferimento persistente
  persistedStates = tabStates;

  let searchMode: SearchMode = "auto";
  let fromInput = "";
  let toInput = "";
  let autoRangeMinutes: number | null = null;
  let autoRefreshSeconds: number | null = 10;
  let smartPrompt = "";
  let smartError = "";
  let smartRequest: QueryRequest | null = null;
  let advancedFilters: FilterItem[] = [];
  let filterDurationOperator = ">";
  let filterDurationMs = "";
  let traceErrorScope: "all" | "with_errors" | "without_errors" = "all";

  // Track previous tab to save state before switching
  let previousTab = "";

  // React to activeTab changes
  $: if (activeTab && activeTab !== previousTab) {
    const oldTab = previousTab;
    if (oldTab) saveState(oldTab);
    loadState(activeTab);
    previousTab = activeTab;

    // Sincronizza il riferimento persistente
    persistedStates = tabStates;

    // Esegui submit automatico solo se non è il caricamento iniziale (gestito da onMount)
    if (oldTab && searchMode === "auto") {
      submit();
    }
  }

  function saveState(tab: string) {
    tabStates[tab] = {
      searchMode,
      fromInput,
      toInput,
      autoRangeMinutes,
      autoRefreshSeconds,
      smartPrompt,
      smartError,
      smartRequest,
      advancedFilters,
      filterDurationOperator,
      filterDurationMs,
      traceErrorScope,
    };
  }

  function loadState(tab: string) {
    const state = tabStates[tab] || { ...defaultState };
    searchMode = state.searchMode;
    fromInput = state.fromInput;
    toInput = state.toInput;
    autoRangeMinutes = state.autoRangeMinutes;
    autoRefreshSeconds = state.autoRefreshSeconds;
    smartPrompt = state.smartPrompt;
    smartError = state.smartError;
    smartRequest = state.smartRequest;
    advancedFilters = state.advancedFilters || [];
    filterDurationOperator = state.filterDurationOperator || ">";
    filterDurationMs = state.filterDurationMs || "";
    traceErrorScope = state.traceErrorScope || "all";
  }

  let smartLoading = false;
  let smartEnabled = false;
  let lastInitialTraceId = "";
  let suppressUrlSync = true;

  // Manual filter fields
  let traceSearch = "";
  let traceSearchExact = false;

  // Separate SQL field
  let generatedSql = "";

  let showFilterModal = false;
  const INPUT_IDLE_AUTOSUBMIT_MS = 700;
  let autoSubmitTimer: ReturnType<typeof setTimeout> | null = null;
  let autoSubmitReady = false;
  let lastManualAutoSubmitKey = "";
  let lastSmartAutoSubmitKey = "";

  function openFilters() {
    showFilterModal = true;
  }

  function closeFilters() {
    showFilterModal = false;
  }

  // Dirty tracking for conditional button enabling
  let promptDirty = true;

  function handlePromptChange() {
    promptDirty = true;
  }

  function handlePromptKeydown(event: KeyboardEvent) {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      if (!smartLoading && promptDirty) {
        generateSql();
      }
    }
  }

  function handleManualEnter(event: KeyboardEvent) {
    if (searchMode !== "manual") return;
    if (showFilterModal) return;
    if (event.key !== "Enter") return;
    if (event.shiftKey || event.ctrlKey || event.metaKey || event.altKey) return;
    if (event.isComposing) return;
    event.preventDefault();
    submit();
  }

  const quickRanges = [
    { label: "Ultimi 5 minuti", minutes: 5, defaultRefresh: 1 },
    { label: "Ultimi 10 minuti", minutes: 10 },
    { label: "Ultimi 30 minuti", minutes: 30 },
    { label: "Ultima ora", minutes: 60 },
    { label: "Tutto", minutes: null },
  ];

  function formatDateTimeLocal(date: Date) {
    const pad = (value: number) => String(value).padStart(2, "0");
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }

  function applyQuickRange(minutes: number | null, defaultRefresh?: number) {
    const now = new Date();
    if (minutes === null) {
      fromInput = "";
      toInput = "";
      return;
    }
    const fromDate = new Date(now.getTime() - minutes * 60 * 1000);
    fromInput = formatDateTimeLocal(fromDate);
    toInput = formatDateTimeLocal(now);

    if (defaultRefresh) {
      autoRefreshSeconds = defaultRefresh;
    }
  }

  function togglePlayPause() {
    if (autoRefreshSeconds) {
      autoRefreshSeconds = null;
    } else {
      autoRefreshSeconds = 1;
    }
    submit();
  }

  function toIso(value: string) {
    if (!value) return "";
    try {
      const d = new Date(value);
      return isNaN(d.getTime()) ? "" : d.toISOString();
    } catch {
      return "";
    }
  }

  function applyTraceIdOverride(traceId: string) {
    if (!traceId) return;
    searchMode = "manual";
    // In trace deep-link flow, clear restrictive filters to guarantee the trace lookup.
    selectService("Tutti");
    selectLogLevel("Tutti");
    advancedFilters = [];
    filterDurationOperator = ">";
    filterDurationMs = "";
    traceErrorScope = "all";
    traceSearch = traceId;
    traceSearchExact = true;
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
    url.searchParams.delete("mode");
    if (url.searchParams.get("tab") === "tracce") {
      url.searchParams.delete("tab");
    }
    goto(url.pathname + url.search, { replaceState: true });
  }

  function applyUrlParams() {
    const params = $page.url.searchParams;
    const modeParam = params.get("mode");
    if (
      modeParam === "auto" ||
      modeParam === "manual" ||
      modeParam === "smart"
    ) {
      searchMode = modeParam;
    }

    const rangeParam = params.get("range");
    if (rangeParam === "all") {
      autoRangeMinutes = null;
    } else if (rangeParam) {
      const parsed = Number(rangeParam);
      autoRangeMinutes = Number.isFinite(parsed) ? parsed : autoRangeMinutes;
    }

    const refreshParam = params.get("refresh");
    if (refreshParam) {
      const parsed = Number(refreshParam);
      autoRefreshSeconds = Number.isFinite(parsed)
        ? parsed
        : autoRefreshSeconds;
    }

    const serviceParam = params.get("service");
    if (serviceParam) {
      selectService(serviceParam);
    }

    const severityParam = params.get("severity");
    if (severityParam) {
      selectLogLevel(severityParam);
    }

    const traceParam = params.get("traceId");
    if (traceParam) {
      applyTraceIdOverride(traceParam);
    }

    if (searchMode === "manual" && !traceParam) {
      const fromParam = params.get("from");
      const toParam = params.get("to");
      if (fromParam) fromInput = fromParam;
      if (toParam) toInput = toParam;
    }

    if (searchMode === "smart") {
      const promptParam = params.get("prompt");
      const sqlParam = params.get("sql");
      if (promptParam) smartPrompt = promptParam;
      if (sqlParam) {
        generatedSql = sqlParam;
      }
    }
  }

  function syncUrlWithState() {
    if (suppressUrlSync) return;
    const params = new URLSearchParams();
    params.set("tab", activeTab);
    params.set("mode", searchMode);

    const selectedService = $servicesState.selectedService;
    const selectedLogLevel = $servicesState.selectedLogLevel;
    if (selectedService && selectedService !== "Tutti") {
      params.set("service", selectedService);
    }
    if (selectedLogLevel && selectedLogLevel !== "Tutti") {
      params.set("severity", selectedLogLevel);
    }

    if (searchMode === "auto") {
      params.set(
        "range",
        autoRangeMinutes === null ? "all" : String(autoRangeMinutes),
      );
      if (autoRefreshSeconds) {
        params.set("refresh", String(autoRefreshSeconds));
      }
    }

    if (searchMode === "manual") {
      if (traceSearch) {
        params.set("traceId", traceSearch);
      } else {
        if (fromInput) params.set("from", fromInput);
        if (toInput) params.set("to", toInput);
      }
    }

    if (searchMode === "smart") {
      if (smartPrompt) params.set("prompt", smartPrompt);
      if (generatedSql) params.set("sql", generatedSql);
    }

    const current = $page.url.searchParams.toString();
    const next = params.toString();
    if (current !== next) {
      goto(`${$page.url.pathname}?${next}`, { replaceState: true });
    }
  }

  onMount(async () => {
    loadState(activeTab);
    applyUrlParams();
    if (forceMode) {
      searchMode = forceMode;
    }

    if (initialTraceId) {
      traceSearch = initialTraceId;
      traceSearchExact = true;
      lastInitialTraceId = initialTraceId;
      fromInput = "";
      toInput = "";
    }

    // Caricamento differito per stabilità degli store e della navigazione
    setTimeout(() => {
      suppressUrlSync = false;
      autoSubmitReady = true;
      if (searchMode === "auto" || autoRun) {
        submit();
      }
    }, 100);

    try {
      const settings = await getAISettings();
      smartEnabled = settings.enabled;
    } catch (e) {
    }
  });

  onDestroy(() => {
    if (autoSubmitTimer) {
      clearTimeout(autoSubmitTimer);
      autoSubmitTimer = null;
    }
  });

  function queueAutoSubmit(delayMs = INPUT_IDLE_AUTOSUBMIT_MS) {
    if (autoSubmitTimer) {
      clearTimeout(autoSubmitTimer);
    }
    autoSubmitTimer = setTimeout(() => {
      submit();
    }, delayMs);
  }

  $: if (initialTraceId && initialTraceId !== lastInitialTraceId) {
    lastInitialTraceId = initialTraceId;
    applyTraceIdOverride(initialTraceId);
  }

  $: syncUrlWithState();

  // Auto-submit when global filters change (if in auto/manual mode)
  // We track the previous values to avoid initial double-fetch if needed
  let lastService = $servicesState.selectedService;
  let lastLogLevel = $servicesState.selectedLogLevel;

  $: {
    if (
      ($servicesState.selectedService !== lastService ||
        $servicesState.selectedLogLevel !== lastLogLevel) &&
      searchMode === "auto"
    ) {
      lastService = $servicesState.selectedService;
      lastLogLevel = $servicesState.selectedLogLevel;
      submit();
    }
  }

  function handleModeChange(nextMode: SearchMode) {
    if (searchMode === nextMode) return;
    searchMode = nextMode;
    smartError = "";
    if (nextMode !== "manual" && traceSearch) {
      traceSearch = "";
      traceSearchExact = false;
      clearTraceIdFromUrl();
    }
    if (searchMode === "auto") {
      applyQuickRange(autoRangeMinutes);
      submit();
    }
    dispatch("modeChange", { mode: searchMode });
  }

  async function generateSql() {
    smartError = "";
    smartRequest = null;
    const prompt = smartPrompt.trim();
    if (!prompt) {
      smartError = "Inserisci una richiesta in linguaggio naturale.";
      return;
    }
    smartLoading = true;

    let contextType = "auto";
    if (activeTab === "logs") contextType = "logs";
    if (activeTab === "metriche") contextType = "metrics";
    if (activeTab === "tracce") contextType = "traces";

    try {
      const response = await generateSmartQuery({
        prompt,
        contextType: contextType as "logs" | "metrics" | "traces" | "auto",
      });
      generatedSql = response.sql;
      smartRequest = response.request;
      promptDirty = false;
      queueAutoSubmit(0);
    } catch (err) {
      smartError =
        err instanceof Error ? err.message : "Impossibile generare la query.";
    } finally {
      smartLoading = false;
    }
  }

  function submit() {
    const limit = 100;
    const selectedService = $servicesState.selectedService;
    const selectedLogLevel = $servicesState.selectedLogLevel;
    const serviceFilter =
      selectedService && selectedService !== "Tutti"
        ? { "service.name": selectedService }
        : {};

    // Add manual filters
    const manualFilters: Record<string, string> = { ...serviceFilter };
    if (
      activeTab === "logs" &&
      selectedLogLevel &&
      selectedLogLevel !== "Tutti"
    ) {
      manualFilters["severity"] = selectedLogLevel;
    }
    const traceSearchTerm =
      activeTab === "tracce" ? String(traceSearch || "").trim() : "";

    if (searchMode === "smart") {
      if (!generatedSql) {
        return;
      }
      // Reset global filters to avoid conflict/confusion
      selectService("Tutti");
      selectLogLevel("Tutti");

      dispatch("run", {
        request: {
          ...smartRequest,
          sql: generatedSql,
          // Since we reset globals, we don't pass manualFilters derived from them
          // Assuming smartRequest.filters contains what AI thinks is needed
          filters: { ...(smartRequest?.filters ?? {}) },
          page: 1,
          limit,
        },
        autoRefreshSeconds: null,
        autoRefreshRangeMinutes: null,
      });
      return;
    }

    const rangeMinutes = searchMode === "auto" ? autoRangeMinutes : null;
    const refreshSeconds =
      searchMode === "auto" ? (autoRefreshSeconds ?? 10) : 0;
    if (searchMode === "auto") {
      applyQuickRange(autoRangeMinutes);
    }
    const zeroTime = "1970-01-01T00:00:00Z";
    let from = zeroTime;
    let to = new Date().toISOString();

    if (searchMode === "auto") {
      if (rangeMinutes !== null) {
        from = toIso(fromInput) || from;
        to = toIso(toInput) || to;
      }
    } else {
      if (!traceSearchExact && fromInput && toInput) {
        from = toIso(fromInput) || from;
        to = toIso(toInput) || to;
      }
    }

    if (!from || !to) {
      // cleanup if conversion failed
      from = zeroTime;
      to = zeroTime;
    }
    // Merge manual filters into filterList (always, to assume control over operators)
    let finalFilterList = [...advancedFilters];
    for (const [k, v] of Object.entries(manualFilters)) {
      // Only add if not already present to avoid duplicates
      // Note: This simple check prevents overriding advanced filters with same key
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

    if (activeTab === "tracce" && traceSearchTerm) {
      if (traceSearchExact) {
        if (
          !finalFilterList.some(
            (f) => f.key === "trace_id" && f.value === traceSearchTerm,
          )
        ) {
          finalFilterList.push({
            connector: "AND",
            key: "trace_id",
            operator: "=",
            value: traceSearchTerm,
          });
        }
      } else if (
        !finalFilterList.some(
          (f) => f.key === "trace_or_span" && f.value === traceSearchTerm,
        )
      ) {
        finalFilterList.push({
          connector: "AND",
          key: "trace_or_span",
          operator: "contains",
          value: traceSearchTerm,
        });
      }
    }

    // Filter out incomplete filters
    finalFilterList = finalFilterList.filter((f) => f.key && f.value);

    dispatch("run", {
      request: {
        signals: ["logs", "traces", "metrics"],
        timeRange: { from, to },
        filters: manualFilters,
        filterList: finalFilterList,
        page: 1,
        limit,
      },
      autoRefreshSeconds: refreshSeconds || null,
      autoRefreshRangeMinutes: rangeMinutes,
    });
  }

  $: if (!traceSearch && $page.url.searchParams.get("traceId")) {
    clearTraceIdFromUrl();
  }

  $: if (autoSubmitReady && searchMode === "manual") {
    const manualAutoSubmitKey = JSON.stringify({
      tab: activeTab,
      service: $servicesState.selectedService,
      level: $servicesState.selectedLogLevel,
      fromInput,
      toInput,
      traceSearch,
      traceSearchExact,
      filterDurationOperator,
      filterDurationMs,
      traceErrorScope,
      advancedFilters,
    });
    if (manualAutoSubmitKey !== lastManualAutoSubmitKey) {
      lastManualAutoSubmitKey = manualAutoSubmitKey;
      queueAutoSubmit();
    }
  }

  $: if (autoSubmitReady && searchMode === "smart") {
    const smartAutoSubmitKey = JSON.stringify({
      tab: activeTab,
      generatedSql,
      smartRequest,
    });
    if (smartAutoSubmitKey !== lastSmartAutoSubmitKey) {
      lastSmartAutoSubmitKey = smartAutoSubmitKey;
      if (generatedSql) {
        queueAutoSubmit();
      }
    }
  }

  function clearManualFilters() {
    fromInput = "";
    toInput = "";
    traceSearch = "";
    traceSearchExact = false;
    filterDurationOperator = ">";
    filterDurationMs = "";
    traceErrorScope = "all";
    advancedFilters = [];
  }
</script>

<div class="query-form">
  <fieldset class="mode-picker">
    <legend>Modalita di ricerca</legend>
    <div class="mode-buttons">
      <button
        type="button"
        class:active={searchMode === "auto"}
        on:click={() => handleModeChange("auto")}
      >
        Automatica
      </button>
      <button
        type="button"
        class:active={searchMode === "manual"}
        on:click={() => handleModeChange("manual")}
      >
        Manuale
      </button>
      {#if smartEnabled}
        <button
          type="button"
          class:active={searchMode === "smart"}
          on:click={() => handleModeChange("smart")}
        >
          Smart
        </button>
      {/if}
    </div>
  </fieldset>

  <div class="query-service-filter">
    <ServiceDropdown />
  </div>

  {#if activeTab === "logs" && searchMode !== "smart"}
    <div class="log-level-filter">
      <LogLevelSelector />
    </div>
  {/if}

  {#if searchMode === "auto"}
    <fieldset class="quick-range">
      <legend>Intervallo automatico</legend>
      <div class="quick-range-buttons">
        {#each quickRanges as range}
          <button
            type="button"
            class:active={autoRangeMinutes === range.minutes}
            on:click={() => {
              autoRangeMinutes = range.minutes;
              applyQuickRange(range.minutes, range.defaultRefresh);
              submit();
            }}
          >
            {range.label}
          </button>
        {/each}
      </div>
      <div class="auto-refresh">
        <label for="play-pause">Aggiornamento Live</label>
        <button
          id="play-pause"
          type="button"
          class="btn-icon"
          class:active={!!autoRefreshSeconds}
          on:click={togglePlayPause}
          title={autoRefreshSeconds
            ? "Pausa aggiornamento automatico"
            : "Attiva aggiornamento Live"}
        >
          {#if autoRefreshSeconds}
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
              ><rect x="6" y="4" width="4" height="16"></rect><rect
                x="14"
                y="4"
                width="4"
                height="16"
              ></rect></svg
            >
            <span>Live</span>
          {:else}
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
              ><polygon points="5 3 19 12 5 21 5 3"></polygon></svg
            >
            <span>Play</span>
          {/if}
        </button>
      </div>
    </fieldset>
  {:else if searchMode === "manual"}
    <div class="manual-filters">
      <div class="date-row">
        <div class="filter-field compact date-range-field">
          <label for="query-from">Da</label>
          <input
            id="query-from"
            type="datetime-local"
            bind:value={fromInput}
            on:keydown={handleManualEnter}
          />
        </div>
        <div class="filter-field compact date-range-field">
          <label for="query-to">A</label>
          <input
            id="query-to"
            type="datetime-local"
            bind:value={toInput}
            on:keydown={handleManualEnter}
          />
        </div>
      </div>

      {#if activeTab === "tracce"}
        <div class="filter-field">
          <label for="filter-trace-search">Ricerca testuale</label>
          <input
            id="filter-trace-search"
            type="text"
            placeholder="Cerca tramite trace id o nome span..."
            bind:value={traceSearch}
            on:input={() => {
              traceSearchExact = false;
            }}
            on:keydown={handleManualEnter}
          />
        </div>
        <div class="duration-row">
          <div class="filter-field compact operator">
            <label for="filter-duration-op">Durata</label>
            <select
              id="filter-duration-op"
              bind:value={filterDurationOperator}
              on:keydown={handleManualEnter}
            >
              <option value=">">&gt;</option>
              <option value="<">&lt;</option>
            </select>
          </div>
          <div class="filter-field compact">
            <label for="filter-duration-ms">Durata (ms)</label>
            <input
              id="filter-duration-ms"
              type="number"
              min="0"
              step="1"
              placeholder="es. 300"
              bind:value={filterDurationMs}
              on:keydown={handleManualEnter}
            />
          </div>
        </div>
        <div class="filter-field">
          <label>Errori Traccia</label>
          <div class="trace-error-scope" role="group" aria-label="Filtro errori traccia">
            <button
              type="button"
              class:active={traceErrorScope === "all"}
              on:click={() => (traceErrorScope = "all")}
            >
              Tutte
            </button>
            <button
              type="button"
              class:active={traceErrorScope === "with_errors"}
              on:click={() => (traceErrorScope = "with_errors")}
            >
              Con errori
            </button>
            <button
              type="button"
              class:active={traceErrorScope === "without_errors"}
              on:click={() => (traceErrorScope = "without_errors")}
            >
              Senza errori
            </button>
          </div>
        </div>
      {/if}

      {#if activeTab === "logs"}
        <div class="advanced-filters-trigger">
          <button type="button" class="btn-secondary" on:click={openFilters}>
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
              ><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"
              ></polygon></svg
            >
            Filtri Avanzati
            {#if advancedFilters.length > 0}
              <span class="badge">{advancedFilters.length}</span>
            {/if}
          </button>
        </div>

        <Modal
          open={showFilterModal}
          title="Filtri Avanzati"
          on:close={closeFilters}
        >
          <FilterBuilder bind:filters={advancedFilters} />
          <div class="modal-actions">
            <button class="btn-primary" on:click={closeFilters}
              >Applica Filtri</button
            >
          </div>
        </Modal>
      {/if}

      <div class="actions-row bottom">
        <button
          type="button"
          class="btn-text"
          on:click={clearManualFilters}
          title="Svuota tutti i campi"
        >
          Pulisci filtri
        </button>
      </div>
    </div>
  {:else}
    <div class="smart-box">
      <div class="smart-field">
        <label for="smart-prompt">Prompt in linguaggio naturale</label>
        <textarea
          id="smart-prompt"
          rows="3"
          bind:value={smartPrompt}
          on:input={handlePromptChange}
          on:keydown={handlePromptKeydown}
          placeholder="Es: Mostrami gli errori del servizio checkout negli ultimi 10 minuti"
        ></textarea>
      </div>

      <div class="smart-actions">
        <button
          type="button"
          class="btn-generate"
          on:click={generateSql}
          disabled={smartLoading || !promptDirty}
        >
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              d="M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z"
            />
            <path d="M19 13l1 3 3 1-3 1-1 3-1-3-3-1 3-1 1-3z" />
          </svg>
          {smartLoading ? "Generazione..." : "Genera Query"}
        </button>
        {#if smartError}
          <span class="error">{smartError}</span>
        {/if}
      </div>

      {#if generatedSql}
        <div class="smart-field">
          <label for="smart-sql">Query SQL Generata</label>
          <textarea
            id="smart-sql"
            rows="5"
            bind:value={generatedSql}
            class="sql-editor"
          ></textarea>
          <p class="helper">Le modifiche vengono applicate automaticamente.</p>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .query-form {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .mode-picker,
  .quick-range {
    border: none;
    padding: 0;
    margin: 0;
  }

  .mode-picker legend,
  .quick-range legend {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
    margin-bottom: 10px;
  }

  .mode-buttons {
    display: inline-flex;
    gap: 8px;
    padding: 6px;
    border-radius: 999px;
    background: #f1f5f9;
  }

  .mode-buttons button {
    padding: 8px 14px;
    font-size: 12px;
    font-weight: 600;
    border: none;
    border-radius: 999px;
    background: transparent;
    color: #475569;
    cursor: pointer;
  }

  .mode-buttons button.active {
    background: white;
    color: #0f172a;
    box-shadow: 0 4px 12px rgba(15, 23, 42, 0.08);
  }

  .quick-range-buttons {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .auto-refresh {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 24px;
  }

  .auto-refresh label {
    margin-bottom: 0;
  }

  .manual-filters {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .advanced-filters-trigger {
    display: flex;
    flex-direction: column;
    gap: 16px;
    margin-top: 10px;
  }

  .filter-field {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-bottom: 10px;
  }

  .quick-range-buttons button {
    padding: 8px 12px;
    font-size: 12px;
    font-weight: 500;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    background: #f8fafc;
    color: #475569;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .trace-error-scope {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    background: #f1f5f9;
    border: 1px solid #e2e8f0;
    border-radius: 10px;
    padding: 4px;
    gap: 4px;
  }

  .trace-error-scope button {
    border: none;
    background: transparent;
    color: #475569;
    border-radius: 8px;
    font-size: 12px;
    font-weight: 600;
    padding: 8px 10px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .trace-error-scope button:hover {
    background: #e2e8f0;
    color: #334155;
  }

  .trace-error-scope button.active {
    background: white;
    color: #0f172a;
    box-shadow: 0 2px 8px rgba(15, 23, 42, 0.12);
  }

  .quick-range-buttons button:hover {
    border-color: #6366f1;
    color: #6366f1;
    background: rgba(99, 102, 241, 0.05);
  }

  .quick-range-buttons button.active {
    border-color: #6366f1;
    color: #4338ca;
    background: rgba(99, 102, 241, 0.1);
  }

  label {
    display: block;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
    margin-bottom: 6px;
  }

  .btn-icon {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 8px 12px;
    font-size: 13px;
    font-weight: 600;
    color: #475569;
    background: white;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-icon:hover {
    border-color: #cbd5e1;
    background: #f8fafc;
  }

  .btn-icon.active {
    color: #ef4444;
    border-color: #fecaca;
    background: #fef2f2;
  }

  .btn-icon.active:hover {
    background: #fee2e2;
  }

  input,
  textarea,
  select {
    width: 100%;
    padding: 12px 14px;
    font-size: 13px;
    border: 1px solid #e2e8f0;
    border-radius: 10px;
    background: #f8fafc;
    color: #0f172a;
    resize: vertical;
  }

  textarea:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
    background: white;
  }

  .smart-box {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .smart-field {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .sql-editor {
    font-family: "Fira Code", "Consolas", "Monaco", monospace;
    font-size: 12px;
    background: #1e293b;
    color: #e2e8f0;
    border: 1px solid #334155;
    border-radius: 10px;
  }

  .sql-editor:focus {
    border-color: #6366f1;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.2);
  }

  .smart-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .btn-generate {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 600;
    border: none;
    border-radius: 8px;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: white;
    cursor: pointer;
    transition: all 0.2s ease;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
  }

  .btn-generate:hover:not([disabled]) {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(99, 102, 241, 0.4);
  }

  .btn-generate[disabled] {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .smart-actions .error {
    color: #b91c1c;
    font-size: 12px;
    font-weight: 600;
  }

  .helper {
    margin-top: 8px;
    font-size: 12px;
    color: #64748b;
  }

  .query-form > button {
    margin-top: 8px;
    padding: 14px 20px;
    font-size: 14px;
    font-weight: 600;
    border: none;
    border-radius: 10px;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: white;
    cursor: pointer;
    transition: all 0.2s ease;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
  }

  .query-form > button:hover:not([disabled]) {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(99, 102, 241, 0.4);
  }

  .query-form > button[disabled] {
    opacity: 0.5;
    cursor: not-allowed;
    transform: none;
  }

  .date-row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
    align-items: end;
    width: 100%;
  }

  .date-row .date-range-field {
    min-width: 0;
  }

  .date-row .date-range-field label {
    margin-bottom: 2px;
    font-size: 12px;
  }

  .date-row .date-range-field input[type="datetime-local"] {
    height: 38px;
    width: 100%;
    padding: 6px 6px;
    font-size: 10.5px;
    font-variant-numeric: tabular-nums;
    min-width: 0;
    box-sizing: border-box;
  }

  .date-row .date-range-field input[type="datetime-local"]::-webkit-datetime-edit {
    padding: 0;
  }

  @media (max-width: 760px) {
    .date-row {
      grid-template-columns: 1fr;
    }
  }

  .filter-field.compact {
    flex: 1 1 180px;
    min-width: 0;
  }

  .filter-field.compact input {
    padding: 8px 10px;
    font-size: 13px;
  }

  .duration-row {
    display: grid;
    grid-template-columns: 110px minmax(0, 1fr);
    gap: 12px;
    align-items: end;
  }

  .filter-field.compact.operator select {
    width: 100%;
    padding: 8px 10px;
    font-size: 13px;
    border: 1px solid #e2e8f0;
    border-radius: 10px;
    background: #f8fafc;
    color: #0f172a;
  }

  .btn-secondary {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: white;
    border: 1px solid #cbd5e1;
    border-radius: 8px;
    color: #334155;
    font-weight: 500;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-secondary:hover {
    background: #f8fafc;
    border-color: #94a3b8;
  }

  .badge {
    background: #6366f1;
    color: white;
    font-size: 11px;
    padding: 2px 6px;
    border-radius: 99px;
    font-weight: 700;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 24px;
    padding-top: 20px;
    border-top: 1px solid #e2e8f0;
  }

  .btn-primary {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 12px 24px;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: white;
    border: none;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
  }

  .btn-primary:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(99, 102, 241, 0.4);
  }

  .btn-primary:active {
    transform: translateY(0);
  }

  .actions-row {
    display: flex;
    justify-content: flex-end;
  }

  .actions-row.bottom {
    margin-top: 8px;
    padding-top: 8px;
    border-top: 1px dashed rgba(148, 163, 184, 0.35);
  }

  .btn-text {
    background: none;
    border: none;
    padding: 4px 8px;
    font-size: 12px;
    color: #64748b;
    cursor: pointer;
    text-decoration: underline;
  }

  .btn-text:hover {
    color: #334155;
  }

  .log-level-filter {
    margin-bottom: 16px;
  }

  .query-service-filter {
    margin-top: -8px;
    margin-bottom: 4px;
    padding-bottom: 14px;
    border-bottom: 1px solid rgba(15, 23, 42, 0.08);
  }
</style>
