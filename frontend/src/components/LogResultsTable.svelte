<script lang="ts">
  import { createEventDispatcher, onDestroy } from "svelte";
  import { goto } from "$app/navigation";
  import { getLocaleTag, locale, t } from "../lib/i18n";

  export let logs: any[] = [];
  export let pagination: { page: number; hasNext: boolean } | null = null;
  export let isLiveUpdate = false;
  export let lastUpdatedLabel = "";
  export let pageSize = "100";
  export let pageSizeOptions: string[] = ["25", "50", "100", "200"];
  export let inlineDetails = false;

  const dispatch = createEventDispatcher();
  let selectedLog: any | null = null;
  let isMessageExpanded = false;
  let lastSelectedKey = "";
  let knownKeys = new Set<string>();
  let highlightKeys = new Set<string>();
  let highlightTimers = new Map<string, ReturnType<typeof setTimeout>>();
  let initialized = false;
  const LOG_MESSAGE_PREVIEW = 320;

  const severityStyles: Record<
    string,
    { label: string; color: string; bg?: string }
  > = {
    fatal: { label: "FATAL", color: "var(--color-danger-500)", bg: "var(--color-danger-100)" },
    error: { label: "ERROR", color: "var(--color-danger-500)", bg: "var(--color-danger-100)" },
    warn: { label: "WARN", color: "var(--color-warning-500)", bg: "var(--color-warning-100)" },
    warning: { label: "WARN", color: "var(--color-warning-500)", bg: "var(--color-warning-100)" },
    info: { label: "INFO", color: "var(--color-info-500)", bg: "var(--color-info-100)" },
    debug: { label: "DEBUG", color: "var(--color-primary-400)", bg: "var(--color-primary-25)" },
    trace: { label: "TRACE", color: "var(--color-slate-500)", bg: "var(--color-slate-100)" },
  };

  function changePage(nextPage: number) {
    dispatch("pageChange", { page: nextPage });
  }

  function changePageSize(value: string) {
    dispatch("pageSizeChange", { size: value });
  }

  function handlePageSizeChange(event: Event) {
    const target = event.currentTarget as HTMLSelectElement | null;
    if (!target) return;
    changePageSize(target.value);
  }

  function formatTimestamp(value: string | number | Date) {
    if (!value) return "-";
    const date = value instanceof Date ? value : new Date(value);
    return new Intl.DateTimeFormat(getLocaleTag($locale), {
      dateStyle: "short",
      timeStyle: "medium",
    }).format(date);
  }

  function shortId(id?: string) {
    if (!id) return "";
    return id.length > 12 ? `${id.slice(0, 8)}...${id.slice(-4)}` : id;
  }

  function parseStructured(value: any) {
    if (!value) return null;
    if (typeof value === "object") return value;
    if (typeof value !== "string") return null;
    const trimmed = value.trim();
    if (!trimmed.startsWith("{") && !trimmed.startsWith("[")) return null;
    try {
      return JSON.parse(trimmed);
    } catch {
      return null;
    }
  }

  function logKey(entry: any) {
    const timestamp = entry?.timestamp ?? "";
    const traceId = entry?.traceId ?? "";
    const spanId = entry?.spanId ?? "";
    const body =
      typeof entry?.body === "string"
        ? entry.body
        : JSON.stringify(entry?.body ?? "");
    return `${timestamp}|${traceId}|${spanId}|${body}`;
  }

  function markHighlight(key: string) {
    const existing = highlightTimers.get(key);
    if (existing) {
      clearTimeout(existing);
    }
    highlightKeys = new Set([...highlightKeys, key]);
    const timeout = setTimeout(() => {
      highlightTimers.delete(key);
      if (!highlightKeys.has(key)) return;
      const next = new Set(highlightKeys);
      next.delete(key);
      highlightKeys = next;
    }, 3000);
    highlightTimers.set(key, timeout);
  }

  function extractMessage(body: any, structured: any) {
    if (structured && typeof structured === "object") {
      const candidate =
        structured.message ||
        structured.msg ||
        structured.event ||
        structured.body;
      if (candidate && typeof candidate === "string") return candidate;
    }
    if (typeof body === "string") return body;
    if (body && typeof body === "object") return JSON.stringify(body);
    return "-";
  }

  function escapeRegex(value: string) {
    return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  }

  function escapeHtml(value: string) {
    return value
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function mergeAttributes(log: any) {
    return {
      ...(log?.resourceAttributes ?? {}),
      ...(log?.logAttributes ?? {}),
    };
  }

  function interpolateMessageHtml(
    message: string,
    attributes: Record<string, any>,
  ) {
    let output = escapeHtml(message);
    for (const [key, rawValue] of Object.entries(attributes)) {
      const value = formatValue(rawValue);
      if (!value) continue;
      const escapedKey = escapeRegex(key);
      const patterns = [
        new RegExp(`\\{\\s*${escapedKey}\\s*\\}`, "g"),
        new RegExp(`\\$\\{\\s*${escapedKey}\\s*\\}`, "g"),
        new RegExp(`%\\{\\s*${escapedKey}\\s*\\}`, "g"),
      ];
      const replacement = `<strong>${escapeHtml(value)}</strong>`;
      for (const pattern of patterns) {
        output = output.replace(pattern, replacement);
      }
    }
    return output;
  }

  function formatLogMessageHtml(log: any, structured: any) {
    const base = extractMessage(log?.body, structured);
    if (!base || base === "-") return base;
    const attributes = mergeAttributes(log);
    return interpolateMessageHtml(base, attributes);
  }

  function formatLogMessageHtmlFromText(log: any, message: string) {
    if (!message || message === "-") return message;
    const attributes = mergeAttributes(log);
    return interpolateMessageHtml(message, attributes);
  }

  function formatValue(value: any) {
    if (value === null || value === undefined) return "";
    if (typeof value === "string") return value;
    if (typeof value === "number" || typeof value === "boolean")
      return String(value);
    return JSON.stringify(value);
  }

  function buildTags(log: any) {
    const merged = {
      ...(log.resourceAttributes ?? {}),
      ...(log.logAttributes ?? {}),
    };
    const preferredKeys = [
      "service.name",
      "service.version",
      "host.name",
      "deployment.environment",
    ];
    const orderedKeys = preferredKeys.filter((key) => key in merged);
    const extraKeys = Object.keys(merged).filter(
      (key) => !orderedKeys.includes(key),
    );
    const keys = [...orderedKeys, ...extraKeys].slice(0, 6);
    return keys
      .map((key) => ({ key, value: formatValue(merged[key]) }))
      .filter((tag) => tag.value);
  }

  function severityFor(log: any) {
    const key = log?.severity ? String(log.severity).toLowerCase() : "info";
    return severityStyles[key] ?? severityStyles.info;
  }

  function downloadJson(data: any, filename: string) {
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  function closeLogModal() {
    selectedLog = null;
  }

  function goToTraceSearch(traceId: string) {
    if (!traceId) return;
    closeLogModal();
    goto(
      `/?traceId=${encodeURIComponent(traceId)}&tab=tracce&mode=manual&autorun=1`,
    );
  }

  function handleBackdropKeydown(event: KeyboardEvent, onClose: () => void) {
    if (event.key === "Enter" || event.key === " " || event.key === "Escape") {
      event.preventDefault();
      onClose();
    }
  }

  $: if (logs) {
    const nextKeys = new Set(logs.map(logKey));
    if (!initialized) {
      knownKeys = nextKeys;
      initialized = true;
    } else if (isLiveUpdate) {
      for (const key of nextKeys) {
        if (!knownKeys.has(key)) {
          markHighlight(key);
        }
      }
      knownKeys = nextKeys;
    } else {
      // Page change or manual query - reset without highlighting
      knownKeys = nextKeys;
      highlightKeys = new Set();
    }
  }
  $: if (selectedLog) {
    const nextKey = logKey(selectedLog);
    if (nextKey !== lastSelectedKey) {
      lastSelectedKey = nextKey;
      isMessageExpanded = false;
    }
  }
  onDestroy(() => {
    highlightTimers.forEach((timer) => clearTimeout(timer));
    highlightTimers.clear();
  });
</script>

<div class="results-container" class:split-view={inlineDetails && selectedLog}>
  <div class="list-pane">
  <div class="results-content">
    {#if logs.length === 0}
      <div class="empty">{t($locale, "logs.empty")}</div>
    {:else}
      <ul class="log-list">
        {#each logs as log}
          {@const structuredBody = parseStructured(log.body)}
          {@const severity = severityFor(log)}
          {@const key = logKey(log)}
          <li>
            <button
              type="button"
              class="log-row"
              class:new-item={highlightKeys.has(key)}
              on:click={() => (selectedLog = log)}
            >
              <span class="timestamp">{formatTimestamp(log.timestamp)}</span>
              <span class="severity" style={`color:${severity.color}`}
                >{severity.label}</span
              >
              <span class="message"
                >{@html formatLogMessageHtml(log, structuredBody)}</span
              >
              {#if log.traceId}
                <span class="trace">{t($locale, "logs.trace", { id: shortId(log.traceId) })}</span>
              {/if}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>

  {#if pagination}
    <div class="pager">
      <div class="pager-meta">
        {#if lastUpdatedLabel}
          <span class="last-refresh">{lastUpdatedLabel}</span>
        {/if}
        <div class="page-size-selector">
          <label for="logs-page-size">{t($locale, "pagination.results")}</label>
          <select
            id="logs-page-size"
            value={pageSize}
            on:change={handlePageSizeChange}
          >
            {#each pageSizeOptions as size}
              <option value={size}>{size}</option>
            {/each}
          </select>
        </div>
      </div>
      <div class="pager-controls">
        <button
          type="button"
          class="pager-btn"
          on:click={() => changePage(1)}
          disabled={pagination.page <= 1}
          title={t($locale, "pagination.firstPage")}
        >
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <polyline points="11 17 6 12 11 7"></polyline>
            <polyline points="18 17 13 12 18 7"></polyline>
          </svg>
        </button>
        <button
          type="button"
          class="pager-btn"
          on:click={() => changePage(pagination.page - 1)}
          disabled={pagination.page <= 1}
          title={t($locale, "pagination.previousPage")}
        >
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <polyline points="15 18 9 12 15 6"></polyline>
          </svg>
        </button>
        <span>{t($locale, "pagination.page", { page: pagination.page })}</span>
        <button
          type="button"
          class="pager-btn"
          on:click={() => changePage(pagination.page + 1)}
          disabled={!pagination.hasNext}
          title={t($locale, "pagination.nextPage")}
        >
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <polyline points="9 18 15 12 9 6"></polyline>
          </svg>
        </button>
      </div>
    </div>
  {/if}
  </div>

  {#if selectedLog}
    {@const structuredBody = parseStructured(selectedLog.body)}
    {@const tags = buildTags(selectedLog)}
    <!-- svelte-ignore a11y-no-noninteractive-tabindex -->
    <div
      class="details-wrapper"
      class:as-modal={!inlineDetails}
      role={!inlineDetails ? 'button' : undefined}
      tabindex={!inlineDetails ? 0 : undefined}
      aria-label={!inlineDetails ? t($locale, "logs.closeDetails") : undefined}
      on:click|self={() => { if (!inlineDetails) closeLogModal(); }}
      on:keydown={(event) => { if (!inlineDetails) handleBackdropKeydown(event, closeLogModal); }}
    >
      <div class="modal log-modal" class:inline-panel={inlineDetails} role="dialog" aria-modal="true">
        <header class="modal-header">
        <div class="header-title">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"
            ></path>
            <polyline points="14 2 14 8 20 8"></polyline>
            <line x1="16" y1="13" x2="8" y2="13"></line>
            <line x1="16" y1="17" x2="8" y2="17"></line>
          </svg>
          <span>{t($locale, "logs.detailsTitle")}</span>
        </div>
        <div class="header-actions">
          <button
            type="button"
            class="close-btn"
            on:click={() => downloadJson(selectedLog, `log-${String(selectedLog.timestamp).replace(/[:.]/g, '-')}.json`)}
            aria-label={t($locale, 'logs.downloadJson')}
            title={t($locale, 'logs.downloadJson')}
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
              <polyline points="7 10 12 15 17 10"></polyline>
              <line x1="12" y1="15" x2="12" y2="3"></line>
            </svg>
          </button>
          <button
            type="button"
            class="close-btn"
            on:click={closeLogModal}
            aria-label={t($locale, "common.close")}
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>
      </header>

      <div class="modal-body">
        <!-- Info principali: Severity e Data -->
        <div class="log-header-info">
          <div
            class="severity-badge severity-{(
              selectedLog.severity || 'info'
            ).toLowerCase()}"
          >
            {selectedLog.severity || "INFO"}
          </div>
          <span class="log-timestamp"
            >{formatTimestamp(selectedLog.timestamp)}</span
          >
        </div>

        <!-- Messaggio -->
        <section class="log-section message-section">
          <h4 class="section-title">{t($locale, "logs.message")}</h4>
          {#if true}
            {@const baseMessage = extractMessage(selectedLog.body, structuredBody)}
            {@const isLongMessage = baseMessage && baseMessage.length > LOG_MESSAGE_PREVIEW}
            {@const displayMessage = isLongMessage && !isMessageExpanded
              ? `${baseMessage.slice(0, LOG_MESSAGE_PREVIEW).trimEnd()}…`
              : baseMessage}
            <p class="log-message">
              {@html formatLogMessageHtmlFromText(selectedLog, displayMessage)}
            </p>
            {#if isLongMessage}
              <button
                type="button"
                class="message-toggle"
                on:click={() => (isMessageExpanded = !isMessageExpanded)}
              >
                  {isMessageExpanded ? t($locale, "logs.showLess") : t($locale, "logs.showMore")}
              </button>
            {/if}
          {/if}
        </section>

        <!-- Attributi Log (prioritari) -->
        {#if selectedLog.logAttributes && Object.keys(selectedLog.logAttributes).length > 0}
          <section class="log-section">
            <h4 class="section-title">{t($locale, "logs.logAttributes")}</h4>
            <div class="attributes-list">
              {#each Object.entries(selectedLog.logAttributes) as [key, value]}
                <div class="attr-row">
                  <span class="attr-key">{key}</span>
                  <span class="attr-value">{formatValue(value)}</span>
                </div>
              {/each}
            </div>
          </section>
        {/if}

        <!-- Sezione secondaria -->
        <div class="secondary-info">
          <!-- Trace/Span Info -->
          {#if selectedLog.traceId || selectedLog.spanId || tags.length > 0}
            <section class="log-section compact">
              <h4 class="section-title">{t($locale, "logs.context")}</h4>
              <div class="context-grid">
                {#if selectedLog.traceId}
                  <div class="context-item">
                    <span class="context-label">{t($locale, "logs.traceId")}</span>
                    <button
                      type="button"
                      class="context-value link"
                      on:click={() => goToTraceSearch(selectedLog.traceId)}
                    >
                      {selectedLog.traceId}
                    </button>
                  </div>
                {/if}
                {#if selectedLog.spanId}
                  <div class="context-item">
                    <span class="context-label">{t($locale, "logs.spanId")}</span>
                    <span class="context-value mono">{selectedLog.spanId}</span>
                  </div>
                {/if}
                {#each tags as tag}
                  <div class="context-item">
                    <span class="context-label">{tag.key}</span>
                    <span class="context-value">{tag.value}</span>
                  </div>
                {/each}
              </div>
            </section>
          {/if}

          <!-- Corpo strutturato -->
          {#if structuredBody}
            <section class="log-section compact">
              <h4 class="section-title">{t($locale, "logs.structuredBody")}</h4>
              <pre class="code-block">{JSON.stringify(
                  structuredBody,
                  null,
                  2,
                )}</pre>
            </section>
          {/if}

          <!-- Attributi Risorsa -->
          {#if selectedLog.resourceAttributes && Object.keys(selectedLog.resourceAttributes).length > 0}
            <section class="log-section compact">
              <h4 class="section-title">{t($locale, "logs.resourceAttributes")}</h4>
              <div class="attributes-list compact">
                {#each Object.entries(selectedLog.resourceAttributes) as [key, value]}
                  <div class="attr-row">
                    <span class="attr-key">{key}</span>
                    <span class="attr-value">{formatValue(value)}</span>
                  </div>
                {/each}
              </div>
            </section>
          {/if}
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .empty {
    color: var(--color-slate-400);
    font-size: 14px;
    text-align: center;
    padding: 40px 0;
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
  }

  .results-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    flex: 1;
  }

  .results-container.split-view {
    display: grid;
    grid-template-columns: 2fr 1fr;
    grid-template-rows: minmax(0, 1fr);
    gap: 12px;
    min-height: 0;
    overflow: hidden;
  }

  .list-pane {
    display: flex;
    flex-direction: column;
    min-height: 0;
    min-width: 0;
    flex: 1;
    overflow: hidden;
  }

  .results-content {
    flex: 1;
    overflow-y: auto;
    min-height: 0;
  }

  .log-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .log-row {
    width: 100%;
    display: grid;
    grid-template-columns: 140px 70px 1fr auto;
    gap: 16px;
    align-items: center;
    padding: 12px 16px;
    border-radius: 10px;
    border: 1px solid rgba(148, 163, 184, 0.2);
    background: white;
    cursor: pointer;
    text-align: left;
    transition:
      border 0.15s ease,
      box-shadow 0.15s ease;
  }

  .log-row:hover {
    border-color: var(--color-info-600);
    box-shadow: 0 2px 8px rgba(37, 99, 235, 0.12);
  }

  .log-row.new-item {
    background: #fff7ed;
    border-color: var(--color-warning-500);
    box-shadow: 0 0 0 1px rgba(245, 158, 11, 0.2);
    animation: highlight-fade 3s ease-out forwards;
  }

  @keyframes highlight-fade {
    0% {
      background: #fff7ed;
      border-color: var(--color-warning-500);
      box-shadow: 0 0 0 1px rgba(245, 158, 11, 0.2);
    }
    100% {
      background: white;
      border-color: rgba(148, 163, 184, 0.2);
      box-shadow: none;
    }
  }

  .timestamp {
    font-size: 12px;
    color: var(--color-slate-500);
    font-weight: 600;
  }

  .severity {
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 4px 8px;
    border-radius: 6px;
    display: inline-block;
    text-align: center;
    width: fit-content;
  }

  .message {
    font-size: 13px;
    color: var(--color-slate-950);
    font-weight: 500;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  .trace {
    font-size: 11px;
    padding: 4px 10px;
    border-radius: 999px;
    background: #f5f3ff;
    color: var(--color-primary-600);
    font-weight: 600;
  }

  .pager {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 20px 16px 6px 16px;
    border-top: 1px solid var(--color-slate-200);
    background: white;
    flex-shrink: 0;
  }

  .pager-meta {
    min-width: 0;
    flex: 1;
  }

  .last-refresh {
    font-size: 12px;
    color: var(--color-slate-400);
    white-space: nowrap;
  }

  .page-size-selector {
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-width: 150px;
  }

  .page-size-selector label {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-slate-500);
  }

  .page-size-selector select {
    padding: 7px 30px 7px 10px;
    border-radius: 8px;
    border: 1px solid var(--color-slate-200);
    background: white;
    font-size: 12px;
    font-weight: 500;
    color: var(--color-slate-950);
    cursor: pointer;
    transition: all 0.2s ease;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%2364748b' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 9px center;
  }

  .page-size-selector select:hover {
    border-color: var(--color-slate-300);
  }

  .page-size-selector select:focus {
    outline: none;
    border-color: var(--color-primary-600);
    box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.1);
  }

  .pager-controls {
    display: inline-flex;
    align-items: center;
    gap: 12px;
  }

  .pager button {
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 500;
    border: 1px solid var(--color-slate-200);
    border-radius: 8px;
    background: white;
    color: var(--color-slate-600);
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .pager button:hover:not(:disabled) {
    border-color: var(--color-primary-600);
    color: var(--color-primary-600);
  }

  .pager button:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .pager span {
    font-size: 13px;
    color: var(--color-slate-500);
    padding: 0 8px;
  }

  @media (max-width: 760px) {
    .pager {
      flex-direction: column;
      align-items: flex-start;
    }

    .pager-controls {
      width: 100%;
      justify-content: center;
    }
  }

  .pager-btn {
    min-width: 36px;
    padding: 8px 12px !important;
  }

  .details-wrapper.as-modal {
    position: fixed;
    inset: 0;
    background: rgba(var(--rgb-slate-950), 0.7);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    z-index: 100;
  }

  .details-wrapper:not(.as-modal) {
    display: flex;
    min-height: 0;
    min-width: 0;
    overflow: hidden;
  }

  .log-modal {
    width: min(1200px, 96vw);
    max-height: 90vh;
    overflow: hidden;
    background: var(--color-slate-50);
    border-radius: 16px;
    box-shadow:
      0 25px 50px -12px rgba(0, 0, 0, 0.25),
      0 0 0 1px rgba(255, 255, 255, 0.1);
    display: flex;
    flex-direction: column;
  }

  .log-modal.inline-panel {
    width: 100%;
    max-height: 100%;
    height: 100%;
    min-height: 0;
    border-radius: 12px;
    background: var(--color-white);
    border: 1px solid rgba(148, 163, 184, 0.2);
    box-shadow: none;
    overflow-y: auto;
  }

  .log-modal.inline-panel .modal-header {
    background: var(--color-white);
    color: var(--color-slate-950);
    border-bottom: 1px solid var(--color-slate-200);
    padding: 12px;
  }

  .log-modal.inline-panel .header-title {
    font-size: 11px;
    color: var(--color-slate-500);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-weight: 700;
  }

  .log-modal.inline-panel .header-title svg {
    color: var(--color-slate-500);
    width: 16px;
    height: 16px;
  }

  .log-modal.inline-panel .close-btn {
    background: var(--color-slate-50);
    border: 1px solid var(--color-slate-200);
    color: var(--color-slate-600);
    border-radius: 8px;
    padding: 6px;
  }

  .log-modal.inline-panel .close-btn:hover {
    background: var(--color-slate-100);
    color: var(--color-slate-900);
  }

  .log-modal.inline-panel .close-btn svg {
    width: 14px;
    height: 14px;
  }

  .log-modal.inline-panel .modal-body {
    padding: 12px;
    gap: 10px;
    overflow: visible;
    flex: none;
  }

  .log-modal.inline-panel .log-header-info {
    padding-bottom: 10px;
  }

  .log-modal.inline-panel .log-section,
  .log-modal.inline-panel .log-section.compact,
  .log-modal.inline-panel .log-section.message-section {
    padding: 0;
    background: transparent;
    border: none;
    border-radius: 0;
    margin-bottom: 12px;
  }

  .log-modal.inline-panel .section-title {
    margin: 0 0 6px 0;
  }

  .log-modal.inline-panel .secondary-info {
    border-top: none;
    padding-top: 0;
    gap: 12px;
  }

  .log-modal.inline-panel .log-message {
    font-size: 13px;
    line-height: 1.5;
  }

  .log-modal.inline-panel .context-grid {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .log-modal.inline-panel .context-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .log-modal.inline-panel .context-label {
    font-size: 10px;
    word-break: break-word;
  }

  .log-modal.inline-panel .context-value {
    font-size: 12px;
  }

  .log-modal.inline-panel .attributes-list,
  .log-modal.inline-panel .attributes-list.compact {
    gap: 4px;
  }

  .log-modal.inline-panel .attr-row {
    display: grid;
    grid-template-columns: minmax(120px, 170px) 1fr;
    gap: 8px;
    padding: 0;
    background: transparent;
    border: none;
    border-radius: 0;
    font-size: 11px;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  }

  .log-modal.inline-panel .attr-row .attr-key {
    font-size: 11px;
    font-weight: 400;
    text-transform: none;
    letter-spacing: 0;
    color: #0ea5e9;
    font-family: inherit;
  }

  .log-modal.inline-panel .attr-row .attr-value {
    font-size: 11px;
    color: var(--color-slate-700);
  }

  .log-modal.inline-panel .code-block {
    font-size: 11px;
    padding: 8px;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 20px;
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
    color: white;
    flex-shrink: 0;
  }

  .header-title {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 16px;
    font-weight: 600;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.1);
    border: none;
    color: white;
    cursor: pointer;
    padding: 8px;
    border-radius: 8px;
    transition: all 0.2s ease;
  }

  .close-btn:hover {
    background: rgba(255, 255, 255, 0.2);
  }

  .close-btn:hover {
    background: rgba(255, 255, 255, 0.2);
  }

  .modal-body {
    flex: 1;
    overflow: auto;
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 20px;
  }

  .log-header-info {
    display: flex;
    align-items: center;
    gap: 16px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--color-slate-200);
  }

  .severity-badge {
    display: inline-flex;
    align-items: center;
    padding: 6px 14px;
    border-radius: 8px;
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .severity-badge.severity-fatal,
  .severity-badge.severity-error {
    background: var(--color-danger-50);
    color: var(--color-danger-600);
    border: 1px solid var(--color-danger-75);
  }

  .severity-badge.severity-warn,
  .severity-badge.severity-warning {
    background: #fffbeb;
    color: #d97706;
    border: 1px solid #fde68a;
  }

  .severity-badge.severity-info {
    background: #eff6ff;
    color: var(--color-info-600);
    border: 1px solid #bfdbfe;
  }

  .severity-badge.severity-debug,
  .severity-badge.severity-trace {
    background: var(--color-slate-50);
    color: var(--color-slate-500);
    border: 1px solid var(--color-slate-200);
  }

  .log-timestamp {
    font-size: 14px;
    color: var(--color-slate-500);
    font-weight: 500;
  }

  .log-section {
    background: white;
    border-radius: 12px;
    padding: 16px;
    border: 1px solid var(--color-slate-200);
  }

  .log-section.compact {
    padding: 14px;
  }

  .log-section.message-section {
    background: var(--color-slate-50);
  }

  .section-title {
    margin: 0 0 10px 0;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--color-slate-500);
  }

  .log-message {
    margin: 0;
    font-size: 15px;
    color: var(--color-slate-950);
    line-height: 1.7;
    word-break: break-word;
  }

  .message-toggle {
    margin-top: 8px;
    padding: 0;
    border: none;
    background: none;
    color: var(--color-info-600);
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }

  .message-toggle:hover {
    text-decoration: underline;
  }

  .secondary-info {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-top: 8px;
    border-top: 1px solid var(--color-slate-200);
  }

  .context-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;
  }

  .context-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .context-label {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-slate-400);
  }

  .context-value {
    font-size: 12px;
    color: var(--color-slate-950);
    word-break: break-all;
  }

  .context-value.mono {
    font-family: "Courier New", monospace;
    color: var(--color-slate-600);
  }

  .context-value.link {
    background: none;
    border: none;
    padding: 0;
    color: var(--color-primary-600);
    font-weight: 600;
    cursor: pointer;
    text-align: left;
    font-size: 12px;
    transition: color 0.2s ease;
  }

  .context-value.link:hover {
    color: var(--color-primary-700);
    text-decoration: underline;
  }

  .code-block {
    margin: 0;
    padding: 12px;
    background: var(--color-slate-950);
    color: var(--color-slate-200);
    border-radius: 8px;
    font-size: 12px;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-word;
    font-family: "Courier New", monospace;
  }

  .attributes-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .attr-row {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 8px 10px;
    background: var(--color-slate-50);
    border-radius: 6px;
    border: 1px solid var(--color-slate-100);
  }

  .attr-row .attr-key {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-slate-500);
    font-family: "Courier New", monospace;
  }

  .attr-row .attr-value {
    font-size: 13px;
    color: var(--color-slate-950);
    word-break: break-all;
  }

  @media (max-width: 900px) {
    .log-details-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 720px) {
    .log-row {
      grid-template-columns: 1fr;
      gap: 6px;
    }
  }
</style>
