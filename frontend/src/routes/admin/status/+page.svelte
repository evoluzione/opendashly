<script lang="ts">
  import { onMount } from "svelte";
  import {
    fetchStatusSummary,
    type StatusSummary,
    type TelemetryCounts,
  } from "../../../services/status";

  let summary: StatusSummary | null = null;
  let loading = false;
  let error = "";
  let lastUpdated: Date | null = null;

  const numberFormat = new Intl.NumberFormat("it-IT");
  const compactFormat = new Intl.NumberFormat("it-IT", {
    notation: "compact",
    maximumFractionDigits: 1,
  });

  type SignalKey = "logs" | "traces" | "metrics";

  const signalConfig: Record<
    SignalKey,
    { label: string; color: string; tint: string }
  > = {
    logs: { label: "Log", color: "#2563eb", tint: "#dbeafe" },
    traces: { label: "Tracce", color: "#14b8a6", tint: "#ccfbf1" },
    metrics: { label: "Metriche", color: "#f59e0b", tint: "#fef3c7" },
  };

  let health = appHealthState(summary);
  $: health = appHealthState(summary);

  function formatCount(value: number | undefined) {
    if (value === undefined || value === null) return "-";
    return numberFormat.format(value);
  }

  function formatCompact(value: number | undefined) {
    if (value === undefined || value === null) return "-";
    return compactFormat.format(value);
  }

  function formatLastUpdated(date: Date | null): string {
    if (!date) return "";
    return date.toLocaleTimeString("it-IT", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }

  function appHealthState(value: StatusSummary | null): {
    label: string;
    tone: "ok" | "warn" | "error";
    reason: string;
  } {
    if (!value) {
      return {
        label: "In attesa",
        tone: "warn",
        reason: "Nessun dato ancora caricato",
      };
    }
    if (!value.checks.database || !value.ok) {
      return {
        label: "Critico",
        tone: "error",
        reason: value.error || "Problemi nel controllo database",
      };
    }
    const recentTotal =
      value.counts.logs.last60m +
      value.counts.traces.last60m +
      value.counts.metrics.last60m;
    if (recentTotal === 0) {
      return {
        label: "Inattivo",
        tone: "warn",
        reason: "Nessun segnale recente negli ultimi 60 minuti",
      };
    }
    return {
      label: "Operativo",
      tone: "ok",
      reason: "Sistema e raccolta telemetria regolari",
    };
  }

  function signalRows(value: StatusSummary | null): Array<{
    key: SignalKey;
    data: TelemetryCounts;
  }> {
    if (!value) return [];
    return [
      { key: "logs", data: value.counts.logs },
      { key: "traces", data: value.counts.traces },
      { key: "metrics", data: value.counts.metrics },
    ];
  }

  function maxWindowValue(data: TelemetryCounts): number {
    return Math.max(data.last5m, data.last10m, data.last60m, 1);
  }

  function windowPercentage(value: number, max: number): number {
    if (!max) return 0;
    return Math.max(4, Math.round((value / max) * 100));
  }

  function sparklinePath(data: TelemetryCounts): string {
    const points =
      data.series && data.series.length > 0
        ? data.series.map((point) => point.count)
        : [data.last5m, data.last10m, data.last60m];
    const max = Math.max(...points, 1);
    const width = 160;
    const height = 48;
    const step = width / (points.length - 1);
    const mapped = points.map((point, idx) => {
      const x = idx * step;
      const y = height - (point / max) * (height - 6) - 3;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    });
    return `M${mapped.join(" L")}`;
  }

  async function loadStatus() {
    loading = true;
    error = "";
    try {
      summary = await fetchStatusSummary();
      error = summary.error ?? "";
      lastUpdated = new Date();
    } catch (err) {
      summary = null;
      error = err instanceof Error ? err.message : "Errore sconosciuto";
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void loadStatus();
  });
</script>

<section class="status-page">
  <header class="status-header">
    <div>
      <h1>Monitor sistema</h1>
      <p>Controllo stato servizi e volumi telemetrici.</p>
    </div>
    <div class="status-actions">
      {#if lastUpdated}
        <span class="last-updated">Ultimo aggiornamento: {formatLastUpdated(lastUpdated)}</span>
      {/if}
      <button type="button" class="refresh-btn" on:click={loadStatus} disabled={loading}>
        <svg
          class={`refresh-icon ${loading ? "spinning" : ""}`}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <polyline points="23 4 23 10 17 10"></polyline>
          <polyline points="1 20 1 14 7 14"></polyline>
          <path d="M3.51 9a9 9 0 0 1 14.14-3.36L23 10M1 14l5.35 4.36A9 9 0 0 0 20.49 15"></path>
        </svg>
        {loading ? "Aggiornamento..." : "Aggiorna"}
      </button>
    </div>
  </header>

  {#if error}
    <div class="status-error">{error}</div>
  {/if}

  <section class="health-strip">
    <article class={`health-card ${health.tone}`}>
      <span class="health-kicker">Salute applicazione</span>
      <strong>{health.label}</strong>
      <p>{health.reason}</p>
    </article>

    <article class="mini-card">
      <span>Database</span>
      <strong>{summary ? (summary.checks.database ? "Connesso" : "Errore") : "-"}</strong>
    </article>

    <article class="mini-card">
      <span>Eventi 60m</span>
      <strong>
        {summary
          ? formatCompact(
              summary.counts.logs.last60m +
                summary.counts.traces.last60m +
                summary.counts.metrics.last60m,
            )
          : "-"}
      </strong>
    </article>

    <article class="mini-card">
      <span>Totale segnali</span>
      <strong>
        {summary
          ? formatCompact(
              summary.counts.logs.total +
                summary.counts.traces.total +
                summary.counts.metrics.total,
            )
          : "-"}
      </strong>
    </article>
  </section>

  {#if summary}
    <div class="status-grid">
      {#each signalRows(summary) as signal}
        {@const cfg = signalConfig[signal.key]}
        {@const maxWindow = maxWindowValue(signal.data)}
        <article class="signal-card">
          <header>
            <div>
              <h3>{cfg.label}</h3>
              <p>Totale: {formatCount(signal.data.total)}</p>
            </div>
            <span class="signal-chip" style={`--chip-bg:${cfg.tint};--chip-color:${cfg.color}`}
              >{formatCompact(signal.data.last60m)} ultimi 60m</span
            >
          </header>

          <div class="signal-chart" style={`--stroke:${cfg.color}`}>
            <svg viewBox="0 0 160 48" preserveAspectRatio="none" aria-hidden="true">
              <path d={sparklinePath(signal.data)}></path>
            </svg>
          </div>

          <div class="window-bars">
            <div class="window-row">
              <span>5m</span>
              <div class="bar-track">
                <div
                  class="bar-fill"
                  style={`--fill:${cfg.color};width:${windowPercentage(signal.data.last5m, maxWindow)}%`}
                ></div>
              </div>
              <strong>{formatCount(signal.data.last5m)}</strong>
            </div>
            <div class="window-row">
              <span>10m</span>
              <div class="bar-track">
                <div
                  class="bar-fill"
                  style={`--fill:${cfg.color};width:${windowPercentage(signal.data.last10m, maxWindow)}%`}
                ></div>
              </div>
              <strong>{formatCount(signal.data.last10m)}</strong>
            </div>
            <div class="window-row">
              <span>60m</span>
              <div class="bar-track">
                <div
                  class="bar-fill"
                  style={`--fill:${cfg.color};width:${windowPercentage(signal.data.last60m, maxWindow)}%`}
                ></div>
              </div>
              <strong>{formatCount(signal.data.last60m)}</strong>
            </div>
          </div>
        </article>
      {/each}
    </div>

  {:else if !loading}
    <div class="status-empty">Nessun dato disponibile.</div>
  {/if}
</section>

<style>
  .status-page {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }

  .status-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
  }

  h1 {
    margin: 0;
    font-size: 24px;
    color: #0f172a;
  }

  p {
    margin: 6px 0 0;
    color: #64748b;
    font-size: 14px;
  }

  .status-actions {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .last-updated {
    font-size: 12px;
    color: #64748b;
  }

  .refresh-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    height: 34px;
    box-sizing: border-box;
    padding: 10px 14px;
    border: none;
    border-radius: 8px;
    background: #6366f1;
    color: #ffffff;
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

  .refresh-icon {
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

  .status-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 14px;
  }

  .health-strip {
    display: grid;
    grid-template-columns: 1.4fr repeat(3, minmax(0, 1fr));
    gap: 12px;
  }

  .health-card,
  .mini-card,
  .signal-card {
    border-radius: 14px;
    border: 1px solid rgba(15, 23, 42, 0.08);
    background: #ffffff;
    padding: 14px;
    box-shadow: 0 10px 24px rgba(15, 23, 42, 0.06);
  }

  .health-card {
    display: flex;
    flex-direction: column;
    gap: 6px;
    background: linear-gradient(140deg, #f8fafc 0%, #ffffff 100%);
  }

  .health-card.ok {
    border-color: #86efac;
  }

  .health-card.warn {
    border-color: #fde68a;
  }

  .health-card.error {
    border-color: #fecaca;
  }

  .health-kicker {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #64748b;
  }

  .health-card strong {
    font-size: 20px;
    color: #0f172a;
  }

  .health-card p {
    margin: 0;
    font-size: 13px;
    color: #475569;
  }

  .mini-card {
    display: flex;
    flex-direction: column;
    gap: 6px;
    justify-content: center;
  }

  .mini-card span {
    font-size: 12px;
    color: #64748b;
  }

  .mini-card strong {
    font-size: 20px;
    color: #0f172a;
  }

  .signal-card {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .signal-card header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 10px;
  }

  .signal-card h3 {
    margin: 0;
    font-size: 16px;
    color: #0f172a;
  }

  .signal-card p {
    margin: 4px 0 0;
    font-size: 12px;
    color: #64748b;
  }

  .signal-chip {
    font-size: 11px;
    font-weight: 700;
    color: var(--chip-color);
    background: var(--chip-bg);
    border-radius: 999px;
    padding: 5px 10px;
    white-space: nowrap;
  }

  .signal-chart {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 10px;
    padding: 8px;
  }

  .signal-chart svg {
    width: 100%;
    height: 52px;
  }

  .signal-chart path {
    fill: none;
    stroke: var(--stroke);
    stroke-width: 2.4;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  .window-bars {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .window-row {
    display: grid;
    grid-template-columns: 32px 1fr auto;
    align-items: center;
    gap: 8px;
  }

  .window-row span {
    font-size: 11px;
    color: #64748b;
    font-weight: 600;
  }

  .window-row strong {
    font-size: 12px;
    color: #0f172a;
    font-weight: 700;
  }

  .bar-track {
    height: 8px;
    border-radius: 999px;
    background: #e2e8f0;
    overflow: hidden;
  }

  .bar-fill {
    height: 100%;
    background: var(--fill);
    border-radius: 999px;
  }

  .status-error {
    font-size: 13px;
    color: #b91c1c;
    background: #fef2f2;
    border: 1px solid #fecaca;
    border-radius: 8px;
    padding: 10px 12px;
  }

  .status-empty {
    font-size: 13px;
    color: #64748b;
  }

  @media (max-width: 1200px) {
    .health-strip {
      grid-template-columns: 1fr 1fr;
    }

    .status-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 760px) {
    .health-strip {
      grid-template-columns: 1fr;
    }
  }
</style>
