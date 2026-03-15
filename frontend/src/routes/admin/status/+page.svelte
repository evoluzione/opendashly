<script lang="ts">
  import { onMount } from "svelte";
  import {
    fetchStatusSummary,
    fetchRuntimeSummary,
    type StatusSummary,
    type TelemetryCounts,
    type RuntimeSummary,
  } from "../../../services/status";

  let summary: StatusSummary | null = null;
  let runtimeSummary: RuntimeSummary | null = null;
  let loading = false;
  let error = "";
  let lastUpdated: Date | null = null;
  let statusRefreshSpinning = false;
  let statusRefreshToken = 0;

  const numberFormat = new Intl.NumberFormat("it-IT");
  const compactFormat = new Intl.NumberFormat("it-IT", {
    notation: "compact",
    maximumFractionDigits: 1,
  });
  const percentFormat = new Intl.NumberFormat("it-IT", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 1,
  });
  const STATUS_REFRESH_MIN_SPIN_MS = 700;

  type SignalKey = "logs" | "traces" | "metrics";
  type HealthTone = "ok" | "warn" | "error";

  type SignalSeries = {
    key: SignalKey;
    label: string;
    color: string;
    tint: string;
    data: TelemetryCounts;
    points: number[];
    max: number;
    lastBucket: number;
    recent15m: number;
    previous15m: number;
    trend15m: number | null;
    freshnessMinutes: number;
  };

  type Insight = {
    tone: HealthTone;
    label: string;
    detail: string;
  };

  const signalConfig: Record<
    SignalKey,
    { label: string; color: string; tint: string }
  > = {
    logs: { label: "Log", color: "#2563eb", tint: "#dbeafe" },
    traces: { label: "Tracce", color: "#14b8a6", tint: "#ccfbf1" },
    metrics: { label: "Metriche", color: "#f59e0b", tint: "#fef3c7" },
  };

  let rows: SignalSeries[] = [];
  $: rows = buildSignalRows(summary);

  let health = appHealthState(summary, rows);
  $: health = appHealthState(summary, rows);

  let globalSeriesMax = 1;
  $: globalSeriesMax = Math.max(...rows.map((row) => row.max), 1);

  let latestTotal = 0;
  $: latestTotal = rows.reduce((sum, row) => sum + row.lastBucket, 0);

  let recent15mTotal = 0;
  $: recent15mTotal = rows.reduce((sum, row) => sum + row.recent15m, 0);

  let previous15mTotal = 0;
  $: previous15mTotal = rows.reduce((sum, row) => sum + row.previous15m, 0);

  let overallTrend15m: number | null = null;
  $: overallTrend15m = computeTrend(recent15mTotal, previous15mTotal);

  let quietSignals = 0;
  $: quietSignals = rows.filter((row) => row.freshnessMinutes >= 10).length;

  let insights: Insight[] = [];
  $: insights = buildInsights(rows, health);

  const timelineWidth = 860;
  const timelineHeight = 260;
  const timelineSvgHeight = 320;

  let hoverIndex: number | null = null;
  let timelinePointCount = 0;
  $: timelinePointCount = rows[0]?.points.length ?? 0;

  let yTicks: Array<{ value: number; y: number }> = [];
  $: yTicks = buildYAxisTicks(globalSeriesMax, timelineHeight);

  let hoverX = 0;
  $: hoverX =
    hoverIndex === null
      ? 0
      : xForIndex(hoverIndex, timelinePointCount, timelineWidth);

  let hoverLabel = "";
  $: hoverLabel =
    hoverIndex === null
      ? ""
      : bucketLabel(hoverIndex, timelinePointCount);

  let hoverValues: Array<{ key: SignalKey; label: string; color: string; value: number }> = [];
  $: hoverValues =
    hoverIndex === null
      ? []
      : rows.map((row) => ({
          key: row.key,
          label: row.label,
          color: row.color,
          value: row.points[hoverIndex] ?? 0,
        }));

  function formatCount(value: number | undefined) {
    if (value === undefined || value === null) return "-";
    return numberFormat.format(value);
  }

  function formatCompact(value: number | undefined) {
    if (value === undefined || value === null) return "-";
    return compactFormat.format(value);
  }

  function formatPercent(value: number | undefined) {
    if (value === undefined || value === null || Number.isNaN(value)) return "-";
    return `${percentFormat.format(value)}%`;
  }

  function formatBytes(value: number | undefined) {
    if (value === undefined || value === null || value <= 0) return "-";
    const units = ["B", "KB", "MB", "GB", "TB"];
    let size = value;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex += 1;
    }
    return `${size.toFixed(size >= 10 ? 0 : 1)} ${units[unitIndex]}`;
  }

  function formatUptime(seconds: number | undefined) {
    if (seconds === undefined || seconds === null || seconds < 0) return "-";
    const total = Math.floor(seconds);
    const hours = Math.floor(total / 3600);
    const minutes = Math.floor((total % 3600) / 60);
    const secs = total % 60;
    if (hours > 0) return `${hours}h ${minutes}m`;
    if (minutes > 0) return `${minutes}m ${secs}s`;
    return `${secs}s`;
  }

  function componentTone(status: string): "ok" | "warn" | "error" {
    if (status === "up") return "ok";
    if (status === "degraded") return "warn";
    return "error";
  }

  function formatLastUpdated(date: Date | null): string {
    if (!date) return "";
    return date.toLocaleTimeString("it-IT", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }

  function sleep(ms: number): Promise<void> {
    return new Promise((resolve) => {
      setTimeout(resolve, ms);
    });
  }

  function appHealthState(
    value: StatusSummary | null,
    seriesRows: SignalSeries[],
  ): {
    label: string;
    tone: HealthTone;
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
    const inactiveSignals = seriesRows.filter(
      (row) => row.freshnessMinutes >= 15,
    ).length;
    if (seriesRows.length > 0 && inactiveSignals === seriesRows.length) {
      return {
        label: "Inattivo",
        tone: "warn",
        reason: "Nessun segnale riceve dati recenti",
      };
    }
    if (seriesRows.some((row) => row.trend15m !== null && row.trend15m <= -45)) {
      return {
        label: "Degradato",
        tone: "warn",
        reason: "Calo significativo nel volume negli ultimi 15 minuti",
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

  function formatTrend(value: number | null) {
    if (value === null) return "n/d";
    const rounded = Math.round(value);
    if (rounded > 0) return `+${rounded}%`;
    return `${rounded}%`;
  }

  function buildPoints(data: TelemetryCounts): number[] {
    if (data.series && data.series.length > 0) {
      return data.series.map((point) => point.count);
    }
    return [
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      Math.max(data.last60m - data.last10m, 0),
      Math.max(data.last10m - data.last5m, 0),
      data.last5m,
      data.last5m,
    ];
  }

  function computeTrend(current: number, previous: number): number | null {
    if (previous === 0) {
      if (current === 0) return 0;
      return null;
    }
    return ((current - previous) / previous) * 100;
  }

  function freshnessFromPoints(points: number[]): number {
    for (let idx = points.length - 1; idx >= 0; idx--) {
      if (points[idx] > 0) {
        return (points.length - 1 - idx) * 5;
      }
    }
    return points.length * 5;
  }

  function buildSignalRows(value: StatusSummary | null): SignalSeries[] {
    if (!value) return [];
    const base: Array<{ key: SignalKey; data: TelemetryCounts }> = [
      { key: "logs", data: value.counts.logs },
      { key: "traces", data: value.counts.traces },
      { key: "metrics", data: value.counts.metrics },
    ];
    return base.map(({ key, data }) => {
      const cfg = signalConfig[key];
      const points = buildPoints(data);
      const max = Math.max(...points, 1);
      const lastBucket = points[points.length - 1] ?? 0;
      const recent15m = points.slice(-3).reduce((sum, point) => sum + point, 0);
      const previous15m = points
        .slice(-6, -3)
        .reduce((sum, point) => sum + point, 0);
      const trend15m = computeTrend(recent15m, previous15m);
      return {
        key,
        label: cfg.label,
        color: cfg.color,
        tint: cfg.tint,
        data,
        points,
        max,
        lastBucket,
        recent15m,
        previous15m,
        trend15m,
        freshnessMinutes: freshnessFromPoints(points),
      };
    });
  }

  function linePath(points: number[], width: number, height: number, max: number): string {
    if (points.length === 0) return "";
    const step = width / Math.max(points.length - 1, 1);
    const mapped = points.map((point, idx) => {
      const x = idx * step;
      const y = height - (point / max) * (height - 10) - 5;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    });
    return `M${mapped.join(" L")}`;
  }

  function areaPath(points: number[], width: number, height: number, max: number): string {
    if (points.length === 0) return "";
    const line = linePath(points, width, height, max);
    const step = width / Math.max(points.length - 1, 1);
    const endX = step * (points.length - 1);
    return `${line} L${endX.toFixed(1)},${height} L0,${height} Z`;
  }

  function heatAlpha(value: number, max: number): number {
    if (value <= 0 || max <= 0) return 0.08;
    const ratio = value / max;
    return Math.min(1, Math.max(0.15, ratio));
  }

  function clamp(value: number, min: number, max: number): number {
    return Math.min(max, Math.max(min, value));
  }

  function xForIndex(index: number, count: number, width: number): number {
    if (count <= 1) return 0;
    const step = width / (count - 1);
    return index * step;
  }

  function yForValue(value: number, max: number, height: number): number {
    return height - (value / Math.max(max, 1)) * (height - 10) - 5;
  }

  function buildYAxisTicks(max: number, height: number): Array<{ value: number; y: number }> {
    const safeMax = Math.max(max, 1);
    const values = [safeMax, safeMax * 0.66, safeMax * 0.33, 0]
      .map((value) => Math.round(value))
      .filter((value, idx, source) => source.indexOf(value) === idx)
      .sort((a, b) => b - a);
    return values.map((value) => ({
      value,
      y: yForValue(value, safeMax, height),
    }));
  }

  function bucketLabel(index: number, count: number): string {
    const minutesAgo = Math.max(0, (count - 1 - index) * 5);
    return minutesAgo === 0 ? "ora" : `-${minutesAgo}m`;
  }

  function handleTimelineMove(event: MouseEvent) {
    if (!timelinePointCount) return;
    const target = event.currentTarget as SVGSVGElement;
    const rect = target.getBoundingClientRect();
    const relativeX = clamp(event.clientX - rect.left, 0, rect.width);
    const raw = rect.width === 0 ? 0 : (relativeX / rect.width) * (timelinePointCount - 1);
    hoverIndex = Math.round(raw);
  }

  function clearTimelineHover() {
    hoverIndex = null;
  }

  function buildInsights(seriesRows: SignalSeries[], currentHealth: { tone: HealthTone }): Insight[] {
    if (seriesRows.length === 0) return [];
    const list: Insight[] = [];

    for (const row of seriesRows) {
      if (row.freshnessMinutes >= 15) {
        list.push({
          tone: "error",
          label: `${row.label} fermo`,
          detail: `Nessun evento da ${row.freshnessMinutes} minuti`,
        });
      } else if (row.trend15m !== null && row.trend15m <= -40) {
        list.push({
          tone: "warn",
          label: `${row.label} in calo`,
          detail: `${formatTrend(row.trend15m)} negli ultimi 15 minuti`,
        });
      } else if (row.trend15m !== null && row.trend15m >= 35) {
        list.push({
          tone: "ok",
          label: `${row.label} in crescita`,
          detail: `${formatTrend(row.trend15m)} negli ultimi 15 minuti`,
        });
      }
    }

    if (list.length === 0) {
      list.push({
        tone: currentHealth.tone,
        label: "Flusso stabile",
        detail: "Nessun segnale di degrado rilevato nella finestra di 60 minuti",
      });
    }

    return list.slice(0, 4);
  }

  async function loadStatus() {
    const spinToken = ++statusRefreshToken;
    const spinStartedAt = Date.now();
    statusRefreshSpinning = true;

    loading = true;
    error = "";
    try {
      const [nextSummary, nextRuntime] = await Promise.allSettled([
        fetchStatusSummary(),
        fetchRuntimeSummary(),
      ]);

      if (nextSummary.status === "rejected") {
        throw nextSummary.reason;
      }

      summary = nextSummary.value;
      error = summary.error ?? "";

      if (nextRuntime.status === "fulfilled") {
        runtimeSummary = nextRuntime.value;
      } else {
        runtimeSummary = null;
      }
      lastUpdated = new Date();
    } catch (err) {
      summary = null;
      runtimeSummary = null;
      error = err instanceof Error ? err.message : "Errore sconosciuto";
    } finally {
      const elapsed = Date.now() - spinStartedAt;
      const remaining = STATUS_REFRESH_MIN_SPIN_MS - elapsed;
      if (remaining > 0) {
        await sleep(remaining);
      }
      if (spinToken === statusRefreshToken) {
        statusRefreshSpinning = false;
      }
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
          class="refresh-icon"
          class:spinning={statusRefreshSpinning}
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
        Aggiorna
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
      <span>Ingestione attuale (5m)</span>
      <strong>{summary ? formatCompact(latestTotal) : "-"}</strong>
    </article>

    <article class="mini-card">
      <span>Trend 15m</span>
      <strong class={"trend " + ((overallTrend15m !== null && overallTrend15m < 0) ? "down" : "up")}>
        {summary ? formatTrend(overallTrend15m) : "-"}
      </strong>
    </article>

    <article class="mini-card">
      <span>Segnali silenziosi</span>
      <strong>{summary ? quietSignals : "-"}</strong>
    </article>
  </section>

  {#if summary}
    <article class="timeline-card">
      <header class="timeline-header">
        <div>
          <h2>Timeline ingestione (ultimi 60m)</h2>
          <p>Confronto tra Log, Tracce e Metriche per bucket da 5 minuti.</p>
        </div>
        <div class="timeline-legend" role="list" aria-label="Legenda segnali">
          {#each rows as row}
            <span role="listitem" class="legend-item">
              <span class="dot" style={`--dot:${row.color}`}></span>{row.label}
            </span>
          {/each}
        </div>
      </header>

      <div class="timeline-chart" aria-label="Andamento ingestione ultimi 60 minuti">
        <svg
          viewBox={`0 0 ${timelineWidth} ${timelineSvgHeight}`}
          preserveAspectRatio="xMidYMid meet"
          role="img"
          on:mousemove={handleTimelineMove}
          on:mouseleave={clearTimelineHover}
        >
          <text x="8" y="14" class="unit-label">eventi / 5m</text>
          {#each yTicks as tick}
            <line x1="0" y1={tick.y} x2={timelineWidth} y2={tick.y} class={tick.value === 0 ? "axis" : "grid"}></line>
            <text x="8" y={tick.y - 6} class="tick-label">{formatCompact(tick.value)}</text>
          {/each}
          {#each rows as row}
            <path class="area" d={areaPath(row.points, timelineWidth, timelineHeight, globalSeriesMax)} style={`--stroke:${row.color};--fill:${row.tint}`}></path>
            <path class="line" d={linePath(row.points, timelineWidth, timelineHeight, globalSeriesMax)} style={`--stroke:${row.color}`}></path>
          {/each}
          {#if hoverIndex !== null}
            <line x1={hoverX} y1="0" x2={hoverX} y2={timelineHeight} class="cursor"></line>
            {#each rows as row}
              <circle
                cx={hoverX}
                cy={yForValue(row.points[hoverIndex] ?? 0, globalSeriesMax, timelineHeight)}
                r="4"
                class="cursor-dot"
                style={`--dot:${row.color}`}
              ></circle>
            {/each}
          {/if}
        </svg>

        {#if hoverIndex !== null}
          <div class="timeline-tooltip" style={`left:${clamp((hoverX / timelineWidth) * 100, 8, 92)}%`}>
            <div class="tooltip-time">{hoverLabel}</div>
            {#each hoverValues as point}
              <div class="tooltip-row">
                <span class="tooltip-dot" style={`--dot:${point.color}`}></span>
                <span>{point.label}</span>
                <strong>{formatCount(point.value)}</strong>
              </div>
            {/each}
          </div>
        {/if}

        <div class="timeline-labels" aria-hidden="true">
          <span>-55m</span>
          <span>-45m</span>
          <span>-35m</span>
          <span>-25m</span>
          <span>-15m</span>
          <span>-5m</span>
          <span>ora</span>
        </div>
      </div>
    </article>

    <div class="status-grid">
      <article class="heatmap-card">
        <header>
          <h3>Buchi di telemetria</h3>
          <p>Intensita per bucket (5 minuti). Celle chiare = segnale debole o assente.</p>
        </header>
        <div class="heatmap">
          {#each rows as row}
            <div class="heatmap-row">
              <span class="row-label">{row.label}</span>
              <div class="cells" role="img" aria-label={`Heatmap ${row.label}`}>
                {#each row.points as point}
                  <span
                    class="cell"
                    style={`--cell:${row.color};--a:${heatAlpha(point, row.max)}`}
                    title={`${row.label}: ${formatCount(point)} eventi`}
                  ></span>
                {/each}
              </div>
              <span class="row-meta">freshness {row.freshnessMinutes}m</span>
            </div>
          {/each}
        </div>
      </article>

      <article class="insights-card">
        <header>
          <h3>Insight automatici</h3>
          <p>Segnali rilevati da trend e freshness.</p>
        </header>
        <div class="insights-list">
          {#each insights as item}
            <div class={`insight ${item.tone}`}>
              <strong>{item.label}</strong>
              <span>{item.detail}</span>
            </div>
          {/each}
        </div>
      </article>

      <article class="signal-card-list">
        <header>
          <h3>Dettaglio per segnale</h3>
          <p>Metriche operative sintetiche per confronto rapido.</p>
        </header>
        <div class="signal-list">
          {#each rows as row}
            <div class="signal-row">
              <div class="signal-meta">
                <span class="signal-name">{row.label}</span>
                <span class="signal-total">totale {formatCompact(row.data.total)}</span>
              </div>
              <div class="signal-kpis">
                <span>{formatCompact(row.lastBucket)} /5m</span>
                <span class={"trend " + ((row.trend15m !== null && row.trend15m < 0) ? "down" : "up")}>
                  {formatTrend(row.trend15m)} 15m
                </span>
                <span>fresh {row.freshnessMinutes}m</span>
              </div>
            </div>
          {/each}
        </div>
      </article>
    </div>

    {#if runtimeSummary}
      <section class="infra-section">
        <header class="infra-header">
          <div>
            <h2>Salute infrastruttura</h2>
            <p>Deploy, componenti runtime e pressione query in tempo reale.</p>
          </div>
          <span class={`infra-badge ${runtimeSummary.ok ? "ok" : "error"}`}>
            {runtimeSummary.ok ? "Stack operativo" : "Stack con criticita"}
          </span>
        </header>

        <div class="infra-grid">
          <article class="infra-card components-card">
            <h3>Componenti deploy</h3>
            <div class="components-list">
              {#each runtimeSummary.components as component}
                <div class="component-row">
                  <div>
                    <strong>{component.name}</strong>
                    {#if component.error}
                      <p>{component.error}</p>
                    {:else if component.latencyMs !== undefined}
                      <p>latenza {component.latencyMs} ms</p>
                    {/if}
                  </div>
                  <span class={`chip ${componentTone(component.status)}`}>{component.status}</span>
                </div>
              {/each}
            </div>
          </article>

          <article class="infra-card queries-card">
            <h3>Salute query database</h3>
            <div class="query-kpis">
              <div>
                <span>Query attive ora</span>
                <strong>{formatCount(runtimeSummary.queries.runningNow)}</strong>
              </div>
              <div>
                <span>Query lente ora (&gt; {runtimeSummary.queries.slowThresholdSec}s)</span>
                <strong>{formatCount(runtimeSummary.queries.slowRunningNow)}</strong>
              </div>
              <div>
                <span>Durata max query attiva</span>
                <strong>{runtimeSummary.queries.maxRunningElapsedSec.toFixed(1)}s</strong>
              </div>
              <div>
                <span>Query lente ultimi 15m</span>
                <strong>{formatCount(runtimeSummary.queries.slowQueriesLast15m)}</strong>
              </div>
              <div>
                <span>Query fallite ultimi 15m</span>
                <strong>{formatCount(runtimeSummary.queries.failedQueriesLast15m)}</strong>
              </div>
            </div>
          </article>

          <article class="infra-card resources-card">
            <h3>Risorse backend</h3>
            <div class="resource-kpis">
              <div>
                <span>CPU disponibili</span>
                <strong>{runtimeSummary.resources.cpuCoresAvailable.toFixed(2)} core</strong>
              </div>
              <div>
                <span>CPU usata (backend, 1 core)</span>
                <strong>{formatPercent(runtimeSummary.resources.cpuUsedPercentOneCore)}</strong>
              </div>
              <div>
                <span>RAM usata</span>
                <strong>{formatBytes(runtimeSummary.resources.memoryUsedBytes)}</strong>
              </div>
              <div>
                <span>RAM disponibile (limite container)</span>
                <strong>{formatBytes(runtimeSummary.resources.memoryLimitBytes)}</strong>
              </div>
              <div>
                <span>Percentuale RAM usata</span>
                <strong>{formatPercent(runtimeSummary.resources.memoryUsedPercent)}</strong>
              </div>
              <div>
                <span>Heap Go</span>
                <strong>{formatBytes(runtimeSummary.resources.goHeapAllocBytes)}</strong>
              </div>
              <div>
                <span>Goroutine</span>
                <strong>{formatCount(runtimeSummary.resources.goRoutines)}</strong>
              </div>
              <div>
                <span>Uptime backend</span>
                <strong>{formatUptime(runtimeSummary.resources.backendUptimeSeconds)}</strong>
              </div>
            </div>
          </article>
        </div>

        {#if runtimeSummary.warnings && runtimeSummary.warnings.length > 0}
          <div class="runtime-warnings">
            {#each runtimeSummary.warnings as warning}
              <p>{warning}</p>
            {/each}
          </div>
        {/if}
      </section>
    {/if}

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

  .mini-card .trend {
    font-variant-numeric: tabular-nums;
  }

  .trend.up {
    color: #047857;
  }

  .trend.down {
    color: #b91c1c;
  }

  .timeline-card,
  .heatmap-card,
  .insights-card,
  .signal-card-list {
    border-radius: 14px;
    border: 1px solid rgba(15, 23, 42, 0.08);
    background: #ffffff;
    padding: 14px;
    box-shadow: 0 10px 24px rgba(15, 23, 42, 0.06);
  }

  .timeline-header {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: flex-start;
    margin-bottom: 12px;
    flex-wrap: wrap;
  }

  .timeline-header h2 {
    margin: 0;
    font-size: 18px;
    color: #0f172a;
  }

  .timeline-header p {
    margin: 4px 0 0;
    font-size: 12px;
  }

  .timeline-legend {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
  }

  .legend-item {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    border-radius: 999px;
    font-size: 11px;
    color: #334155;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 999px;
    background: var(--dot);
  }

  .timeline-chart {
    position: relative;
    border: 1px solid #e2e8f0;
    border-radius: 10px;
    background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
    padding: 10px 10px 6px;
  }

  .timeline-chart svg {
    width: 100%;
    height: auto;
    aspect-ratio: 860 / 320;
    display: block;
  }

  .timeline-chart .axis {
    stroke: #94a3b8;
    stroke-width: 1;
  }

  .timeline-chart .grid {
    stroke: #e2e8f0;
    stroke-width: 1;
    stroke-dasharray: 4 6;
  }

  .timeline-chart .area {
    fill: color-mix(in srgb, var(--fill) 28%, transparent);
    stroke: none;
  }

  .timeline-chart .line {
    fill: none;
    stroke: var(--stroke);
    stroke-width: 2.4;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  .timeline-chart .unit-label,
  .timeline-chart .tick-label {
    fill: #64748b;
    font-size: 11px;
    font-family: inherit;
  }

  .timeline-chart .cursor {
    stroke: #475569;
    stroke-width: 1;
    stroke-dasharray: 4 4;
    opacity: 0.7;
  }

  .timeline-chart .cursor-dot {
    fill: var(--dot);
    stroke: #ffffff;
    stroke-width: 2;
  }

  .timeline-tooltip {
    position: absolute;
    top: 18px;
    transform: translateX(-50%);
    min-width: 150px;
    border: 1px solid #cbd5e1;
    border-radius: 10px;
    padding: 8px 10px;
    background: rgba(255, 255, 255, 0.96);
    box-shadow: 0 10px 22px rgba(15, 23, 42, 0.14);
    backdrop-filter: blur(2px);
    pointer-events: none;
  }

  .tooltip-time {
    font-size: 11px;
    font-weight: 700;
    color: #334155;
    margin-bottom: 6px;
  }

  .tooltip-row {
    display: grid;
    grid-template-columns: auto 1fr auto;
    gap: 6px;
    align-items: center;
    font-size: 12px;
    color: #334155;
  }

  .tooltip-row strong {
    font-variant-numeric: tabular-nums;
    color: #0f172a;
  }

  .tooltip-dot {
    width: 8px;
    height: 8px;
    border-radius: 999px;
    background: var(--dot);
  }

  .timeline-labels {
    margin-top: 4px;
    display: grid;
    grid-template-columns: repeat(7, minmax(0, 1fr));
    font-size: 11px;
    color: #64748b;
  }

  .heatmap-card header h3,
  .insights-card header h3,
  .signal-card-list header h3 {
    margin: 0;
    font-size: 15px;
    color: #0f172a;
  }

  .heatmap-card header p,
  .insights-card header p,
  .signal-card-list header p {
    margin: 4px 0 0;
    font-size: 12px;
  }

  .heatmap {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-top: 10px;
  }

  .heatmap-row {
    display: grid;
    grid-template-columns: 54px 1fr auto;
    gap: 8px;
    align-items: center;
  }

  .row-label {
    font-size: 12px;
    color: #0f172a;
    font-weight: 600;
  }

  .cells {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    gap: 4px;
  }

  .cell {
    height: 18px;
    border-radius: 4px;
    background: color-mix(in srgb, var(--cell) calc(var(--a) * 100%), #f1f5f9);
    border: 1px solid rgba(148, 163, 184, 0.35);
  }

  .row-meta {
    font-size: 11px;
    color: #64748b;
    white-space: nowrap;
  }

  .insights-list {
    margin-top: 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .insight {
    padding: 10px;
    border-radius: 10px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .insight strong {
    font-size: 12px;
    color: #0f172a;
  }

  .insight span {
    font-size: 12px;
    color: #475569;
  }

  .insight.ok {
    border-color: #86efac;
    background: #f0fdf4;
  }

  .insight.warn {
    border-color: #fde68a;
    background: #fefce8;
  }

  .insight.error {
    border-color: #fecaca;
    background: #fef2f2;
  }

  .signal-list {
    margin-top: 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .signal-row {
    border-radius: 10px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    padding: 10px;
    display: flex;
    justify-content: space-between;
    gap: 10px;
    flex-wrap: wrap;
  }

  .signal-meta {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .signal-name {
    font-size: 13px;
    font-weight: 700;
    color: #0f172a;
  }

  .signal-total {
    font-size: 12px;
    color: #64748b;
  }

  .signal-kpis {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
    font-size: 12px;
    color: #334155;
    font-variant-numeric: tabular-nums;
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

  .infra-section {
    margin-top: 2px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .infra-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }

  .infra-header h2 {
    margin: 0;
    font-size: 18px;
    color: #0f172a;
  }

  .infra-header p {
    margin: 4px 0 0;
    font-size: 13px;
  }

  .infra-badge {
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 6px 10px;
    border-radius: 999px;
    border: 1px solid #cbd5e1;
  }

  .infra-badge.ok {
    color: #166534;
    background: #dcfce7;
    border-color: #86efac;
  }

  .infra-badge.error {
    color: #991b1b;
    background: #fee2e2;
    border-color: #fecaca;
  }

  .infra-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 12px;
  }

  .infra-card {
    border-radius: 14px;
    border: 1px solid rgba(15, 23, 42, 0.08);
    background: #ffffff;
    padding: 14px;
    box-shadow: 0 10px 24px rgba(15, 23, 42, 0.06);
  }

  .infra-card h3 {
    margin: 0 0 10px;
    font-size: 15px;
    color: #0f172a;
  }

  .components-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .component-row {
    border-radius: 10px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    padding: 10px;
    display: flex;
    justify-content: space-between;
    gap: 8px;
    align-items: flex-start;
  }

  .component-row strong {
    text-transform: capitalize;
    font-size: 13px;
    color: #0f172a;
  }

  .component-row p {
    margin: 3px 0 0;
    font-size: 12px;
    color: #64748b;
  }

  .chip {
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    border-radius: 999px;
    padding: 4px 8px;
    white-space: nowrap;
  }

  .chip.ok {
    color: #166534;
    background: #dcfce7;
  }

  .chip.warn {
    color: #854d0e;
    background: #fef9c3;
  }

  .chip.error {
    color: #991b1b;
    background: #fee2e2;
  }

  .query-kpis,
  .resource-kpis {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .query-kpis > div,
  .resource-kpis > div {
    border-radius: 10px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .query-kpis span,
  .resource-kpis span {
    font-size: 11px;
    color: #64748b;
  }

  .query-kpis strong,
  .resource-kpis strong {
    font-size: 14px;
    color: #0f172a;
    font-variant-numeric: tabular-nums;
  }

  .runtime-warnings {
    border-radius: 10px;
    border: 1px solid #fde68a;
    background: #fefce8;
    padding: 10px 12px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .runtime-warnings p {
    margin: 0;
    font-size: 12px;
    color: #713f12;
  }

  @media (max-width: 1200px) {
    .health-strip {
      grid-template-columns: 1fr 1fr;
    }

    .status-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .signal-card-list {
      grid-column: span 2;
    }

    .infra-grid {
      grid-template-columns: 1fr 1fr;
    }

    .resources-card {
      grid-column: span 2;
    }
  }

  @media (max-width: 760px) {
    .health-strip {
      grid-template-columns: 1fr;
    }

    .status-grid {
      grid-template-columns: 1fr;
    }

    .signal-card-list {
      grid-column: auto;
    }

    .timeline-chart svg {
      aspect-ratio: 860 / 360;
    }

    .heatmap-row {
      grid-template-columns: 50px 1fr;
    }

    .row-meta {
      grid-column: 1 / -1;
    }

    .infra-grid {
      grid-template-columns: 1fr;
    }

    .resources-card {
      grid-column: auto;
    }

    .query-kpis,
    .resource-kpis {
      grid-template-columns: 1fr;
    }
  }
</style>
