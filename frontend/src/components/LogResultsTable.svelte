<script lang="ts">
  import { createEventDispatcher, onDestroy } from "svelte";
  import { goto } from "$app/navigation";

  export let logs: any[] = [];
  export let pagination: { page: number; totalPages: number } | null = null;
  export let isLiveUpdate = false;

  const dispatch = createEventDispatcher();
  let selectedLog: any | null = null;
  let knownKeys = new Set<string>();
  let highlightKeys = new Set<string>();
  let highlightTimers = new Map<string, ReturnType<typeof setTimeout>>();
  let initialized = false;

  const severityStyles: Record<
    string,
    { label: string; color: string; bg?: string }
  > = {
    fatal: { label: "FATAL", color: "#ef4444", bg: "#fee2e2" },
    error: { label: "ERROR", color: "#ef4444", bg: "#fee2e2" },
    warn: { label: "WARN", color: "#f59e0b", bg: "#fef3c7" },
    warning: { label: "WARN", color: "#f59e0b", bg: "#fef3c7" },
    info: { label: "INFO", color: "#3b82f6", bg: "#dbeafe" },
    debug: { label: "DEBUG", color: "#a855f7", bg: "#f3e8ff" },
    trace: { label: "TRACE", color: "#64748b", bg: "#f1f5f9" },
  };

  function changePage(nextPage: number) {
    dispatch("pageChange", { page: nextPage });
  }

  function formatTimestamp(value: string | number | Date) {
    if (!value) return "-";
    const date = value instanceof Date ? value : new Date(value);
    return new Intl.DateTimeFormat("it-IT", {
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
  onDestroy(() => {
    highlightTimers.forEach((timer) => clearTimeout(timer));
    highlightTimers.clear();
  });
</script>

<div class="results-container">
  <div class="results-content">
    {#if logs.length === 0}
      <div class="empty">Nessun log disponibile per questo intervallo.</div>
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
                <span class="trace">Traccia {shortId(log.traceId)}</span>
              {/if}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>

  {#if pagination}
    <div class="pager">
      <button
        type="button"
        class="pager-btn"
        on:click={() => changePage(1)}
        disabled={pagination.page <= 1}
        title="Prima pagina"
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
        title="Pagina precedente"
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
      <span>Pagina {pagination.page} di {pagination.totalPages || 1}</span>
      <button
        type="button"
        class="pager-btn"
        on:click={() => changePage(pagination.page + 1)}
        disabled={pagination.totalPages > 0
          ? pagination.page >= pagination.totalPages
          : logs.length === 0}
        title="Pagina successiva"
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
      <button
        type="button"
        class="pager-btn"
        on:click={() => changePage(pagination.totalPages || 1)}
        disabled={pagination.totalPages > 0
          ? pagination.page >= pagination.totalPages
          : logs.length === 0}
        title="Ultima pagina"
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
          <polyline points="13 17 18 12 13 7"></polyline>
          <polyline points="6 17 11 12 6 7"></polyline>
        </svg>
      </button>
    </div>
  {/if}
</div>

{#if selectedLog}
  {@const structuredBody = parseStructured(selectedLog.body)}
  {@const tags = buildTags(selectedLog)}
  {@const mergedAttributes = mergeAttributes(selectedLog)}
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    aria-label="Chiudi dettagli log"
    on:click|self={closeLogModal}
    on:keydown={(event) => handleBackdropKeydown(event, closeLogModal)}
  >
    <div class="modal log-modal" role="dialog" aria-modal="true">
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
          <span>Dettagli Log</span>
        </div>
        <button
          type="button"
          class="close-btn"
          on:click={closeLogModal}
          aria-label="Chiudi"
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
          <h4 class="section-title">Messaggio</h4>
          <p class="log-message">
            {@html formatLogMessageHtml(selectedLog, structuredBody)}
          </p>
        </section>

        <!-- Attributi Log (prioritari) -->
        {#if selectedLog.logAttributes && Object.keys(selectedLog.logAttributes).length > 0}
          <section class="log-section">
            <h4 class="section-title">Attributi Log</h4>
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
              <h4 class="section-title">Contesto</h4>
              <div class="context-grid">
                {#if selectedLog.traceId}
                  <div class="context-item">
                    <span class="context-label">Trace ID</span>
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
                    <span class="context-label">Span ID</span>
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
              <h4 class="section-title">Corpo strutturato</h4>
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
              <h4 class="section-title">Attributi Risorsa</h4>
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

<style>
  .empty {
    color: #94a3b8;
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
    border-color: #2563eb;
    box-shadow: 0 2px 8px rgba(37, 99, 235, 0.12);
  }

  .log-row.new-item {
    background: #fff7ed;
    border-color: #f59e0b;
    box-shadow: 0 0 0 1px rgba(245, 158, 11, 0.2);
    animation: highlight-fade 3s ease-out forwards;
  }

  @keyframes highlight-fade {
    0% {
      background: #fff7ed;
      border-color: #f59e0b;
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
    color: #64748b;
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
    color: #0f172a;
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
    color: #6366f1;
    font-weight: 600;
  }

  .pager {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    padding: 20px 16px 6px 16px;
    border-top: 1px solid #e2e8f0;
    background: white;
    flex-shrink: 0;
  }

  .pager button {
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 500;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    background: white;
    color: #475569;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .pager button:hover:not(:disabled) {
    border-color: #6366f1;
    color: #6366f1;
  }

  .pager button:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .pager span {
    font-size: 13px;
    color: #64748b;
    padding: 0 8px;
  }

  .pager-btn {
    min-width: 36px;
    padding: 8px 12px !important;
  }

  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(15, 23, 42, 0.7);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    z-index: 100;
  }

  .log-modal {
    width: min(1200px, 96vw);
    max-height: 90vh;
    overflow: hidden;
    background: #f8fafc;
    border-radius: 16px;
    box-shadow:
      0 25px 50px -12px rgba(0, 0, 0, 0.25),
      0 0 0 1px rgba(255, 255, 255, 0.1);
    display: flex;
    flex-direction: column;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 20px;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
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
    border-bottom: 1px solid #e2e8f0;
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
    background: #fef2f2;
    color: #dc2626;
    border: 1px solid #fecaca;
  }

  .severity-badge.severity-warn,
  .severity-badge.severity-warning {
    background: #fffbeb;
    color: #d97706;
    border: 1px solid #fde68a;
  }

  .severity-badge.severity-info {
    background: #eff6ff;
    color: #2563eb;
    border: 1px solid #bfdbfe;
  }

  .severity-badge.severity-debug,
  .severity-badge.severity-trace {
    background: #f8fafc;
    color: #64748b;
    border: 1px solid #e2e8f0;
  }

  .log-timestamp {
    font-size: 14px;
    color: #64748b;
    font-weight: 500;
  }

  .log-section {
    background: white;
    border-radius: 12px;
    padding: 16px;
    border: 1px solid #e2e8f0;
  }

  .log-section.compact {
    padding: 14px;
  }

  .log-section.message-section {
    background: #f8fafc;
  }

  .section-title {
    margin: 0 0 10px 0;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: #64748b;
  }

  .log-message {
    margin: 0;
    font-size: 15px;
    color: #0f172a;
    line-height: 1.7;
    word-break: break-word;
  }

  .secondary-info {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-top: 8px;
    border-top: 1px solid #e2e8f0;
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
    color: #94a3b8;
  }

  .context-value {
    font-size: 12px;
    color: #0f172a;
    word-break: break-all;
  }

  .context-value.mono {
    font-family: "Courier New", monospace;
    color: #475569;
  }

  .context-value.link {
    background: none;
    border: none;
    padding: 0;
    color: #6366f1;
    font-weight: 600;
    cursor: pointer;
    text-align: left;
    font-size: 12px;
    transition: color 0.2s ease;
  }

  .context-value.link:hover {
    color: #4f46e5;
    text-decoration: underline;
  }

  .code-block {
    margin: 0;
    padding: 12px;
    background: #0f172a;
    color: #e2e8f0;
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
    background: #f8fafc;
    border-radius: 6px;
    border: 1px solid #f1f5f9;
  }

  .attr-row .attr-key {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: #64748b;
    font-family: "Courier New", monospace;
  }

  .attr-row .attr-value {
    font-size: 13px;
    color: #0f172a;
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
