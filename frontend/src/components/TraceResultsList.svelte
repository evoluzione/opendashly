<script lang="ts">
  import { createEventDispatcher, onDestroy } from "svelte";
  import CorrelationPanel from "./CorrelationPanel.svelte";
  import TraceSpanTimeline from "./TraceSpanTimeline.svelte";

  export let traces: any[] = [];
  export let pagination: { page: number; totalPages: number } | null = null;

  const dispatch = createEventDispatcher();
  let selectedTrace: any | null = null;
  let knownKeys = new Set<string>();
  let highlightKeys = new Set<string>();
  let highlightTimers = new Map<string, ReturnType<typeof setTimeout>>();
  let activeTab: "spans" | "logs" = "spans";
  let initialized = false;

  function changePage(nextPage: number) {
    dispatch("pageChange", { page: nextPage });
  }

  function closeModal() {
    selectedTrace = null;
    activeTab = "spans";
  }

  function handleBackdropKeydown(event: KeyboardEvent) {
    if (event.key === "Enter" || event.key === " " || event.key === "Escape") {
      event.preventDefault();
      closeModal();
    }
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

  function formatDuration(value?: number) {
    if (!value || value <= 0) return "0 ms";
    if (value < 1000) return `${Math.round(value)} ms`;
    if (value < 60000) return `${(value / 1000).toFixed(2)} s`;
    return `${(value / 60000).toFixed(2)} min`;
  }

  function getDurationClass(durationMs: number) {
    if (!durationMs || durationMs <= 0) return "duration-fast-extra";
    if (durationMs < 100) return "duration-fast-extra";
    if (durationMs < 500) return "duration-fast";
    if (durationMs < 1000) return "duration-moderate";
    if (durationMs < 2000) return "duration-slow";
    return "duration-critical";
  }

  function traceKey(entry: any) {
    return entry?.traceId ?? "";
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

  $: if (traces) {
    const nextKeys = new Set(traces.map(traceKey));
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

{#if traces.length === 0}
  <div class="empty">Nessuna traccia trovata per i filtri selezionati.</div>
{:else}
  <ul class="trace-list">
    {#each traces as trace}
      {@const key = traceKey(trace)}
      <li>
        <button
          type="button"
          class="trace-row"
          class:new-item={highlightKeys.has(key)}
          on:click={() => (selectedTrace = trace)}
        >
          <span class="name">{trace.name || "Traccia senza nome"}</span>
          <span class="service"
            >{trace.service || "Servizio non specificato"}</span
          >
          <span class="last-seen">{formatTimestamp(trace.lastSeen)}</span>
          <span class="count">Span {trace.spanCount ?? 0}</span>

          <div class="status-cell">
            <span class="duration {getDurationClass(trace.durationMs)}">
              {formatDuration(trace.durationMs)}
            </span>
            {#if trace.errorCount > 0}
              <div class="error-indicator" title="Contiene errori">
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
                >
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="8" x2="12" y2="12"></line>
                  <line x1="12" y1="16" x2="12.01" y2="16"></line>
                </svg>
              </div>
            {/if}
          </div>
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
        : traces.length === 0}
    >
      Successiva
    </button>
  </div>
{/if}

{#if selectedTrace}
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    aria-label="Chiudi dettagli traccia"
    on:click={closeModal}
    on:keydown={handleBackdropKeydown}
  >
    <div class="modal" role="dialog" aria-modal="true" on:click|stopPropagation>
      <header>
        <div>
          <p class="kicker">Dettagli traccia</p>
          <h3>{selectedTrace.name || "Traccia senza nome"}</h3>
        </div>
        <button type="button" class="close" on:click={closeModal}>Chiudi</button
        >
      </header>
      <div class="meta">
        <span class="pill">ID traccia {selectedTrace.traceId}</span>
        <span class="pill">Servizio {selectedTrace.service || "-"}</span>
        <span class="pill">Span {selectedTrace.spanCount ?? 0}</span>
        {#if selectedTrace.errorCount > 0}
          <span class="pill error">Errori {selectedTrace.errorCount}</span>
        {/if}
        <span class="pill"
          >Ultimo span {formatTimestamp(selectedTrace.lastSeen)}</span
        >
      </div>
      <div class="tabs">
        <button
          class="tab"
          class:active={activeTab === "spans"}
          on:click={() => (activeTab = "spans")}
        >
          Span
        </button>
        <button
          class="tab"
          class:active={activeTab === "logs"}
          on:click={() => (activeTab = "logs")}
        >
          Logs
        </button>
      </div>

      <div class="details-panel">
        <div class:hidden={activeTab !== "spans"}>
          <TraceSpanTimeline traceId={selectedTrace.traceId} />
        </div>
        <div class:hidden={activeTab !== "logs"}>
          <CorrelationPanel traceId={selectedTrace.traceId} />
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
  }

  .trace-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .trace-row {
    width: 100%;
    display: grid;
    grid-template-columns: 2.5fr 1fr 160px 100px 120px;
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

  .trace-row:hover {
    border-color: #2563eb;
    box-shadow: 0 2px 8px rgba(37, 99, 235, 0.12);
  }

  .trace-row.new-item {
    background: #fef3c7;
    border-color: #f59e0b;
    box-shadow: 0 0 0 1px rgba(245, 158, 11, 0.2);
    animation: trace-highlight-fade 3s ease-out forwards;
  }

  @keyframes trace-highlight-fade {
    0% {
      background: #fef3c7;
      border-color: #f59e0b;
      box-shadow: 0 0 0 1px rgba(245, 158, 11, 0.2);
    }
    100% {
      background: white;
      border-color: rgba(148, 163, 184, 0.2);
      box-shadow: none;
    }
  }

  .name {
    font-size: 13px;
    color: #0f172a;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .service,
  .last-seen,
  .count {
    font-size: 12px;
    color: #64748b;
    font-weight: 600;
  }

  .status-cell {
    display: flex;
    align-items: center;
    gap: 8px;
    justify-content: flex-end; /* Align to right? or left? User said "a dx". */
    /* If I justify-content: flex-start, it's consistent. */
    /* Let's try flex-start to match other columns. */
    justify-content: flex-start;
  }

  .duration {
    font-size: 12px;
    font-weight: 700;
  }

  .duration-fast-extra {
    color: #22c55e;
  }

  .duration-fast {
    color: #84cc16;
  }

  .duration-moderate {
    color: #ca8a04;
  }

  .duration-slow {
    color: #f97316;
  }

  .duration-critical {
    color: #ef4444;
  }

  .error-indicator {
    color: #ef4444;
    display: flex;
    align-items: center;
    justify-content: center;
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
  }

  .pill.error {
    background: #fee2e2;
    color: #991b1b;
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
    width: min(1600px, 98vw);
    height: 90vh;
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

  .details-panel {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .details-panel > div {
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  .details-panel :global(.panel) {
    padding: 0;
    border: none;
    box-shadow: none;
    height: 100%;
  }

  .tabs {
    display: flex;
    gap: 8px;
    border-bottom: 1px solid #e2e8f0;
    margin-bottom: 0;
  }

  .tab {
    padding: 10px 16px;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    font-size: 13px;
    font-weight: 600;
    color: #64748b;
    cursor: pointer;
    transition: all 0.2s;
  }

  .tab:hover {
    color: #0f172a;
  }

  .tab.active {
    color: #2563eb;
    border-bottom-color: #2563eb;
  }

  .hidden {
    display: none !important;
  }

  @media (max-width: 720px) {
    .trace-row {
      grid-template-columns: 1fr;
      gap: 6px;
    }
  }
</style>
