<script lang="ts">
  import { createEventDispatcher, onDestroy } from "svelte";
  import CorrelationPanel from "./CorrelationPanel.svelte";
  import TraceSpanTimeline from "./TraceSpanTimeline.svelte";

  export let logs: any[] = [];
  export let pagination: { page: number; totalPages: number } | null = null;

  const dispatch = createEventDispatcher();
  let selectedLog: any | null = null;
  let selectedTraceId: string | null = null;
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

  function openTraceModal(traceId: string) {
    selectedLog = null;
    selectedTraceId = traceId;
  }

  function closeLogModal() {
    selectedLog = null;
  }

  function closeTraceModal() {
    selectedTraceId = null;
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
    } else {
      for (const key of nextKeys) {
        if (!knownKeys.has(key)) {
          markHighlight(key);
        }
      }
      knownKeys = nextKeys;
    }
  }

  onDestroy(() => {
    highlightTimers.forEach((timer) => clearTimeout(timer));
    highlightTimers.clear();
  });
</script>

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
          <span class="message">{extractMessage(log.body, structuredBody)}</span
          >
          {#if log.traceId}
            <span class="trace">Traccia {shortId(log.traceId)}</span>
          {/if}
        </button>
      </li>
    {/each}
  </ul>
{/if}

{#if pagination}
  <div class="pager">
    <button
      type="button"
      on:click={() => changePage(pagination.page - 1)}
      disabled={pagination.page <= 1}
    >
      Precedente
    </button>
    <span>Pagina {pagination.page} di {pagination.totalPages || 1}</span>
    <button
      type="button"
      on:click={() => changePage(pagination.page + 1)}
      disabled={pagination.totalPages > 0
        ? pagination.page >= pagination.totalPages
        : logs.length === 0}
    >
      Successiva
    </button>
  </div>
{/if}

{#if selectedLog}
  {@const structuredBody = parseStructured(selectedLog.body)}
  {@const tags = buildTags(selectedLog)}
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    aria-label="Chiudi dettagli log"
    on:click|self={closeLogModal}
    on:keydown={(event) => handleBackdropKeydown(event, closeLogModal)}
  >
    <div class="modal" role="dialog" aria-modal="true">
      <header>
        <div>
          <p class="kicker">Dettagli log</p>
          <h3>{formatTimestamp(selectedLog.timestamp)}</h3>
        </div>
        <button type="button" class="close" on:click={closeLogModal}
          >Chiudi</button
        >
      </header>
      <div class="meta">
        <span class="pill">Severita: {selectedLog.severity || "info"}</span>
        {#if selectedLog.traceId}
          <button
            type="button"
            class="pill link"
            on:click={() => openTraceModal(selectedLog.traceId)}
          >
            Traccia {selectedLog.traceId}
          </button>
        {/if}
        {#if selectedLog.spanId}
          <span class="pill">Span {selectedLog.spanId}</span>
        {/if}
      </div>
      <div class="body">
        <h4>Messaggio</h4>
        <p>{extractMessage(selectedLog.body, structuredBody)}</p>
      </div>
      {#if tags.length > 0}
        <div class="tags">
          {#each tags as tag}
            <span class="tag"
              ><span class="tag-key">{tag.key}</span>{tag.value}</span
            >
          {/each}
        </div>
      {/if}
      <div class="json-grid">
        {#if structuredBody}
          <div>
            <h4>Corpo</h4>
            <pre>{JSON.stringify(structuredBody, null, 2)}</pre>
          </div>
        {/if}
        {#if selectedLog.resourceAttributes && Object.keys(selectedLog.resourceAttributes).length > 0}
          <div>
            <h4>Attributi risorsa</h4>
            <table class="attributes-table">
              <thead>
                <tr>
                  <th>Chiave</th>
                  <th>Valore</th>
                </tr>
              </thead>
              <tbody>
                {#each Object.entries(selectedLog.resourceAttributes) as [key, value]}
                  <tr>
                    <td class="attr-key">{key}</td>
                    <td class="attr-value">{formatValue(value)}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
        {#if selectedLog.logAttributes && Object.keys(selectedLog.logAttributes).length > 0}
          <div>
            <h4>Attributi log</h4>
            <table class="attributes-table">
              <thead>
                <tr>
                  <th>Chiave</th>
                  <th>Valore</th>
                </tr>
              </thead>
              <tbody>
                {#each Object.entries(selectedLog.logAttributes) as [key, value]}
                  <tr>
                    <td class="attr-key">{key}</td>
                    <td class="attr-value">{formatValue(value)}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

{#if selectedTraceId}
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    aria-label="Chiudi dettagli traccia"
    on:click|self={closeTraceModal}
    on:keydown={(event) => handleBackdropKeydown(event, closeTraceModal)}
  >
    <div class="modal trace-modal" role="dialog" aria-modal="true">
      <header>
        <div>
          <p class="kicker">Dettagli traccia</p>
          <h3>{selectedTraceId}</h3>
        </div>
        <button type="button" class="close" on:click={closeTraceModal}
          >Chiudi</button
        >
      </header>
      <TraceSpanTimeline traceId={selectedTraceId} />
      <CorrelationPanel traceId={selectedTraceId} />
    </div>
  </div>
{/if}

<style>
  .empty {
    color: #94a3b8;
    font-size: 14px;
    text-align: center;
    padding: 40px 0;
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
    background: #eef2ff;
    color: #4338ca;
    font-weight: 600;
  }

  .pager {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    margin-top: 20px;
    padding-top: 20px;
    border-top: 1px solid #f1f5f9;
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
    border-color: #2563eb;
    color: #2563eb;
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

  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(15, 23, 42, 0.45);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    z-index: 60;
  }

  .modal {
    width: min(1100px, 96vw);
    max-height: 90vh;
    overflow: auto;
    background: white;
    border-radius: 16px;
    padding: 24px;
    box-shadow: 0 20px 40px rgba(15, 23, 42, 0.2);
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .modal header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  .kicker {
    margin: 0 0 6px 0;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #94a3b8;
  }

  .modal h3 {
    margin: 0;
    font-size: 18px;
    color: #0f172a;
  }

  .close {
    border: none;
    background: #1d4ed8;
    color: white;
    padding: 8px 14px;
    border-radius: 999px;
    font-weight: 600;
    cursor: pointer;
  }

  .meta {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .pill {
    font-size: 12px;
    padding: 6px 10px;
    border-radius: 999px;
    background: #f1f5f9;
    color: #475569;
    font-weight: 600;
  }

  .pill.link {
    background: #1d4ed8;
    color: white;
    text-decoration: none;
    border: none;
    cursor: pointer;
  }

  .body h4,
  .json-grid h4 {
    margin: 0 0 6px 0;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #64748b;
  }

  .body p {
    margin: 0;
    font-size: 14px;
    color: #0f172a;
  }

  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .tag {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    padding: 4px 8px;
    border-radius: 999px;
    background: #f1f5f9;
    color: #475569;
    border: 1px solid rgba(148, 163, 184, 0.3);
  }

  .tag-key {
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    font-size: 9px;
    color: #64748b;
  }

  .json-grid {
    display: grid;
    gap: 12px;
  }

  pre {
    margin: 0;
    padding: 12px;
    background: #0f172a;
    color: #e2e8f0;
    border-radius: 10px;
    font-size: 12px;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .attributes-table {
    width: 100%;
    border-collapse: collapse;
    border-radius: 10px;
    overflow: hidden;
    border: 1px solid #e2e8f0;
    background: white;
  }

  .attributes-table thead {
    background: #f8fafc;
  }

  .attributes-table th {
    text-align: left;
    padding: 10px 14px;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #64748b;
    border-bottom: 2px solid #e2e8f0;
  }

  .attributes-table tbody tr {
    border-bottom: 1px solid #f1f5f9;
    transition: background 0.15s ease;
  }

  .attributes-table tbody tr:last-child {
    border-bottom: none;
  }

  .attributes-table tbody tr:hover {
    background: #f8fafc;
  }

  .attributes-table td {
    padding: 10px 14px;
    font-size: 13px;
  }

  .attr-key {
    font-weight: 600;
    color: #475569;
    font-family: "Courier New", monospace;
    width: 35%;
    vertical-align: top;
  }

  .attr-value {
    color: #0f172a;
    word-break: break-word;
  }

  .trace-modal h3 {
    word-break: break-all;
  }

  @media (max-width: 720px) {
    .log-row {
      grid-template-columns: 1fr;
      gap: 6px;
    }
  }
</style>
