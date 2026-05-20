<script lang="ts">
  import { onMount } from "svelte";
  import {
    fetchRuntimeSummary,
    fetchStatusSummary,
    type RuntimeComponentHealth,
    type RuntimeSummary,
    type StatusSummary,
    type TelemetryCounts,
  } from "../../../services/status";
  import { getLocaleTag, locale, t } from "../../../lib/i18n";

  type Tone = "ok" | "warn" | "error" | "muted";
  type SignalKey = "logs" | "traces";

  type SignalRow = {
    key: SignalKey;
    label: string;
    data: TelemetryCounts;
    points: number[];
    trend: number | null;
    freshness: number;
    tone: Tone;
  };

  type MonitorState = {
    tone: Tone;
    label: string;
    reason: string;
  };

  type Issue = {
    tone: Tone;
    title: string;
    detail: string;
  };

  let summary: StatusSummary | null = null;
  let runtimeSummary: RuntimeSummary | null = null;
  let loading = false;
  let error = "";
  let lastUpdated: Date | null = null;
  let refreshSpinning = false;
  let refreshToken = 0;

  const REFRESH_MIN_SPIN_MS = 500;

  $: signals = buildSignals(summary);
  $: monitorState = buildMonitorState(summary, runtimeSummary, signals, error);
  $: issues = buildIssues(summary, runtimeSummary, signals, error);
  $: totalLast5m = signals.reduce((sum, row) => sum + row.data.last5m, 0);
  $: totalLast60m = signals.reduce((sum, row) => sum + row.data.last60m, 0);
  $: componentCounts = countComponents(runtimeSummary?.components ?? []);
  $: slowQueryCount = runtimeSummary?.queries.slowRunningNow ?? 0;
  $: failedQueryCount = runtimeSummary?.queries.failedQueriesLast15m ?? 0;
  $: memoryPercent = runtimeSummary?.resources.memoryUsedPercent ?? 0;

  function formatCount(value: number | undefined) {
    if (value === undefined || value === null) return "-";
    return new Intl.NumberFormat(getLocaleTag($locale)).format(value);
  }

  function formatCompact(value: number | undefined) {
    if (value === undefined || value === null) return "-";
    return new Intl.NumberFormat(getLocaleTag($locale), {
      notation: "compact",
      maximumFractionDigits: 1,
    }).format(value);
  }

  function formatPercent(value: number | undefined) {
    if (value === undefined || value === null || Number.isNaN(value)) return "-";
    return `${new Intl.NumberFormat(getLocaleTag($locale), {
      maximumFractionDigits: 1,
    }).format(value)}%`;
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
    const days = Math.floor(total / 86400);
    const hours = Math.floor((total % 86400) / 3600);
    const minutes = Math.floor((total % 3600) / 60);
    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
  }

  function formatLastUpdated(date: Date | null) {
    if (!date) return "";
    return date.toLocaleTimeString(getLocaleTag($locale), {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }

  function formatTrend(value: number | null) {
    if (value === null) return t($locale, "status.noBaseline");
    const rounded = Math.round(value);
    if (rounded > 0) return `+${rounded}%`;
    return `${rounded}%`;
  }

  function sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  function buildSignals(value: StatusSummary | null): SignalRow[] {
    if (!value) return [];
    const labels: Record<SignalKey, string> = {
      logs: t($locale, "sidebar.logs"),
      traces: t($locale, "sidebar.traces"),
    };
    const entries: Array<{ key: SignalKey; data: TelemetryCounts }> = [
      { key: "logs", data: value.counts.logs },
      { key: "traces", data: value.counts.traces },
    ];

    return entries.map(({ key, data }) => {
      const points = data.series?.length ? data.series.map((point) => point.count) : fallbackPoints(data);
      const recent15m = points.slice(-3).reduce((sum, point) => sum + point, 0);
      const previous15m = points.slice(-6, -3).reduce((sum, point) => sum + point, 0);
      const trend = computeTrend(recent15m, previous15m);
      const freshness = freshnessFromPoints(points);
      return {
        key,
        label: labels[key],
        data,
        points,
        trend,
        freshness,
        tone: signalTone(data, freshness, trend),
      };
    });
  }

  function fallbackPoints(data: TelemetryCounts): number[] {
    return [
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
      data.last5m,
      data.last5m,
    ];
  }

  function computeTrend(current: number, previous: number): number | null {
    if (previous === 0) return current === 0 ? 0 : null;
    return ((current - previous) / previous) * 100;
  }

  function freshnessFromPoints(points: number[]) {
    for (let index = points.length - 1; index >= 0; index -= 1) {
      if ((points[index] ?? 0) > 0) {
        return (points.length - 1 - index) * 5;
      }
    }
    return points.length * 5;
  }

  function signalTone(data: TelemetryCounts, freshness: number, trend: number | null): Tone {
    if (data.last60m === 0 || freshness >= 20) return "error";
    if (freshness >= 10 || (trend !== null && trend <= -45)) return "warn";
    return "ok";
  }

  function buildMonitorState(
    status: StatusSummary | null,
    runtime: RuntimeSummary | null,
    signalRows: SignalRow[],
    currentError: string,
  ): MonitorState {
    if (currentError) {
      return {
        tone: "error",
        label: t($locale, "status.stateCritical"),
        reason: currentError,
      };
    }
    if (!status) {
      return {
        tone: "muted",
        label: t($locale, "status.stateLoading"),
        reason: t($locale, "status.stateLoadingReason"),
      };
    }
    if (!status.ok || !status.checks.database) {
      return {
        tone: "error",
        label: t($locale, "status.stateCritical"),
        reason: status.error || t($locale, "status.databaseUnavailable"),
      };
    }
    if (runtime && (!runtime.ok || runtime.components.some((component) => component.status === "down"))) {
      return {
        tone: "error",
        label: t($locale, "status.stateCritical"),
        reason: t($locale, "status.runtimeHasDownComponents"),
      };
    }
    if (runtime && runtime.resources.memoryUsedPercent >= 85) {
      return {
        tone: "warn",
        label: t($locale, "status.stateAttention"),
        reason: t($locale, "status.memoryPressure"),
      };
    }
    if (runtime && (runtime.queries.slowRunningNow > 0 || runtime.queries.failedQueriesLast15m > 0)) {
      return {
        tone: "warn",
        label: t($locale, "status.stateAttention"),
        reason: t($locale, "status.queryPressure"),
      };
    }
    if (signalRows.length > 0 && signalRows.every((row) => row.tone === "error")) {
      return {
        tone: "warn",
        label: t($locale, "status.stateAttention"),
        reason: t($locale, "status.noRecentTelemetry"),
      };
    }
    if (signalRows.some((row) => row.tone !== "ok")) {
      return {
        tone: "warn",
        label: t($locale, "status.stateAttention"),
        reason: t($locale, "status.someTelemetryQuiet"),
      };
    }
    return {
      tone: "ok",
      label: t($locale, "status.stateOperational"),
      reason: t($locale, "status.stateOperationalReason"),
    };
  }

  function buildIssues(
    status: StatusSummary | null,
    runtime: RuntimeSummary | null,
    signalRows: SignalRow[],
    currentError: string,
  ): Issue[] {
    const list: Issue[] = [];
    if (currentError) {
      list.push({ tone: "error", title: t($locale, "status.issueRequestFailed"), detail: currentError });
    }
    if (status && (!status.ok || !status.checks.database)) {
      list.push({
        tone: "error",
        title: t($locale, "status.issueDatabase"),
        detail: status.error || t($locale, "status.databaseUnavailable"),
      });
    }
    for (const warning of status?.warnings ?? []) {
      list.push({
        tone: "warn",
        title: t($locale, "status.issueRuntimeWarning"),
        detail: sanitizeIssueDetail(warning),
      });
    }
    for (const row of signalRows) {
      if (row.tone === "error") {
        list.push({
          tone: "error",
          title: t($locale, "status.issueSignalStopped", { label: row.label }),
          detail: t($locale, "status.issueSignalStoppedDetail", { minutes: row.freshness }),
        });
      } else if (row.tone === "warn") {
        list.push({
          tone: "warn",
          title: t($locale, "status.issueSignalWeak", { label: row.label }),
          detail: t($locale, "status.issueSignalWeakDetail", { trend: formatTrend(row.trend) }),
        });
      }
    }
    if (runtime) {
      for (const component of runtime.components) {
        if (component.status === "down" || component.status === "degraded") {
          list.push({
            tone: component.status === "down" ? "error" : "warn",
            title: t($locale, "status.issueComponent", { name: component.name }),
            detail: component.error || component.status,
          });
        }
      }
      if (runtime.resources.memoryUsedPercent >= 85) {
        list.push({
          tone: "warn",
          title: t($locale, "status.issueMemory"),
          detail: `${formatPercent(runtime.resources.memoryUsedPercent)} · ${formatBytes(runtime.resources.memoryUsedBytes)}`,
        });
      }
      if (runtime.queries.slowRunningNow > 0 || runtime.queries.failedQueriesLast15m > 0) {
        list.push({
          tone: "warn",
          title: t($locale, "status.issueQueries"),
          detail: t($locale, "status.issueQueriesDetail", {
            slow: runtime.queries.slowRunningNow,
            failed: runtime.queries.failedQueriesLast15m,
          }),
        });
      }
      for (const warning of runtime.warnings ?? []) {
        list.push({
          tone: "warn",
          title: t($locale, "status.issueRuntimeWarning"),
          detail: sanitizeIssueDetail(warning),
        });
      }
    }
    if (list.length === 0) {
      list.push({
        tone: "ok",
        title: t($locale, "status.noIssues"),
        detail: t($locale, "status.noIssuesDetail"),
      });
    }
    return list.slice(0, 6);
  }

  function countComponents(components: RuntimeComponentHealth[]) {
    return {
      total: components.length,
      up: components.filter((component) => component.status === "up").length,
      degraded: components.filter((component) => component.status === "degraded").length,
      down: components.filter((component) => component.status === "down").length,
    };
  }

  function sanitizeIssueDetail(detail: string) {
    if (!detail) return "";
    if (detail.length <= 180) return detail;
    return `${detail.slice(0, 177)}...`;
  }

  function componentTone(status: string): Tone {
    if (status === "up") return "ok";
    if (status === "degraded") return "warn";
    return "error";
  }

  function barPercent(value: number, max: number) {
    if (max <= 0) return 0;
    return Math.min(100, Math.max(0, (value / max) * 100));
  }

  function sparkMax(points: number[]) {
    return Math.max(...points, 1);
  }

  async function loadStatus() {
    const token = ++refreshToken;
    const startedAt = Date.now();
    refreshSpinning = true;
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
      runtimeSummary = nextRuntime.status === "fulfilled" ? nextRuntime.value : null;
      lastUpdated = new Date();
    } catch (err) {
      summary = null;
      runtimeSummary = null;
      error = err instanceof Error ? err.message : t($locale, "status.unknownError");
    } finally {
      const remaining = REFRESH_MIN_SPIN_MS - (Date.now() - startedAt);
      if (remaining > 0) await sleep(remaining);
      if (token === refreshToken) {
        refreshSpinning = false;
      }
      loading = false;
    }
  }

  onMount(() => {
    void loadStatus();
  });
</script>

<section class="monitor-page">
  <header class="monitor-header">
    <div>
      <h1>{t($locale, "status.title")}</h1>
      <p>{t($locale, "status.subtitleLean")}</p>
    </div>
    <div class="header-actions">
      {#if lastUpdated}
        <span>{t($locale, "status.lastRefresh", { time: formatLastUpdated(lastUpdated) })}</span>
      {/if}
      <button type="button" on:click={loadStatus} disabled={loading}>
        <svg
          class:spinning={refreshSpinning}
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
        {t($locale, "status.refresh")}
      </button>
    </div>
  </header>

  <section class={`state-panel ${monitorState.tone}`}>
    <div class="state-copy">
      <span>{t($locale, "status.currentState")}</span>
      <strong>{monitorState.label}</strong>
      <p>{monitorState.reason}</p>
    </div>
    <div class="state-metrics">
      <div>
        <span>{t($locale, "status.ingestion5m")}</span>
        <strong>{summary ? formatCompact(totalLast5m) : "-"}</strong>
      </div>
      <div>
        <span>{t($locale, "status.queryPressureShort")}</span>
        <strong>{runtimeSummary ? formatCount(slowQueryCount) : "-"}</strong>
      </div>
      <div>
        <span>{t($locale, "status.memoryShort")}</span>
        <strong>{runtimeSummary ? formatPercent(memoryPercent) : "-"}</strong>
      </div>
      <div>
        <span>{t($locale, "status.componentsShort")}</span>
        <strong>{runtimeSummary ? `${componentCounts.up}/${componentCounts.total}` : "-"}</strong>
      </div>
    </div>
  </section>

  <div class="monitor-grid">
    <section class="panel ingest-panel">
      <header class="panel-header">
        <div>
          <h2>{t($locale, "status.ingestionTitle")}</h2>
          <p>{t($locale, "status.ingestionSubtitleLean")}</p>
        </div>
        <strong>{summary ? formatCompact(totalLast60m) : "-"}</strong>
      </header>

      {#if signals.length > 0}
        <div class="signal-table">
          {#each signals as signal}
            <article class={`signal-line ${signal.tone}`}>
              <div class="signal-name">
                <span class={`status-dot ${signal.tone}`}></span>
                <strong>{signal.label}</strong>
              </div>
              <div class="signal-kpis">
                <span>{formatCompact(signal.data.last5m)} / 5m</span>
                <span>{formatCompact(signal.data.last60m)} / 60m</span>
                <span>{t($locale, "status.freshMinutes", { minutes: signal.freshness })}</span>
                <span class:negative={signal.trend !== null && signal.trend < 0}>{formatTrend(signal.trend)}</span>
              </div>
              <div class="spark" aria-hidden="true">
                {#each signal.points as point}
                  <span style={`height:${Math.max(10, barPercent(point, sparkMax(signal.points)))}%`}></span>
                {/each}
              </div>
            </article>
          {/each}
        </div>
      {:else}
        <div class="empty-state">{loading ? t($locale, "common.loading") : t($locale, "status.empty")}</div>
      {/if}
    </section>

    <section class="panel attention-panel">
      <header class="panel-header">
        <div>
          <h2>{t($locale, "status.attentionTitle")}</h2>
          <p>{t($locale, "status.attentionSubtitle")}</p>
        </div>
      </header>

      <div class="issues">
        {#each issues as issue}
          <article class={`issue ${issue.tone}`}>
            <span class={`status-dot ${issue.tone}`}></span>
            <div>
              <strong>{issue.title}</strong>
              <p>{issue.detail}</p>
            </div>
          </article>
        {/each}
      </div>
    </section>
  </div>

  <section class="runtime-layout">
    <article class="panel pressure-panel">
      <header class="panel-header">
        <div>
          <h2>{t($locale, "status.runtimeTitleLean")}</h2>
          <p>{t($locale, "status.runtimeSubtitleLean")}</p>
        </div>
      </header>

      {#if runtimeSummary}
        <div class="runtime-kpis">
          <div>
            <span>{t($locale, "status.dbRunningNow")}</span>
            <strong>{formatCount(runtimeSummary.queries.runningNow)}</strong>
          </div>
          <div>
            <span>{t($locale, "status.dbSlowNow", { sec: runtimeSummary.queries.slowThresholdSec })}</span>
            <strong class:attention={runtimeSummary.queries.slowRunningNow > 0}>
              {formatCount(runtimeSummary.queries.slowRunningNow)}
            </strong>
          </div>
          <div>
            <span>{t($locale, "status.dbFailed15m")}</span>
            <strong class:attention={failedQueryCount > 0}>{formatCount(failedQueryCount)}</strong>
          </div>
          <div>
            <span>{t($locale, "status.dbMaxElapsed")}</span>
            <strong>{runtimeSummary.queries.maxRunningElapsedSec.toFixed(1)}s</strong>
          </div>
        </div>
      {:else}
        <div class="empty-state">{t($locale, "status.runtimeUnavailable")}</div>
      {/if}
    </article>

    <article class="panel resource-panel">
      <header class="panel-header">
        <div>
          <h2>{t($locale, "status.resourcesTitleLean")}</h2>
          <p>{t($locale, "status.resourcesSubtitleLean")}</p>
        </div>
      </header>

      {#if runtimeSummary}
        <div class="resource-list">
          <div class="resource-row">
            <div>
              <span>{t($locale, "status.memoryUsed")}</span>
              <strong>{formatBytes(runtimeSummary.resources.memoryUsedBytes)} / {formatBytes(runtimeSummary.resources.memoryLimitBytes)}</strong>
            </div>
            <div class="meter"><span style={`width:${Math.min(100, memoryPercent)}%`}></span></div>
          </div>
          <div class="resource-row">
            <div>
              <span>{t($locale, "status.cpuUsedOneCore")}</span>
              <strong>{formatPercent(runtimeSummary.resources.cpuUsedPercentOneCore)}</strong>
            </div>
            <div class="meter"><span style={`width:${Math.min(100, runtimeSummary.resources.cpuUsedPercentOneCore)}%`}></span></div>
          </div>
          <div class="resource-meta">
            <span>{t($locale, "status.goHeap")}: {formatBytes(runtimeSummary.resources.goHeapAllocBytes)}</span>
            <span>{t($locale, "status.goroutines")}: {formatCount(runtimeSummary.resources.goRoutines)}</span>
            <span>{t($locale, "status.backendUptime")}: {formatUptime(runtimeSummary.resources.backendUptimeSeconds)}</span>
          </div>
        </div>
      {:else}
        <div class="empty-state">{t($locale, "status.runtimeUnavailable")}</div>
      {/if}
    </article>

    <article class="panel components-panel">
      <header class="panel-header">
        <div>
          <h2>{t($locale, "status.componentsTitleLean")}</h2>
          <p>{t($locale, "status.componentsSubtitleLean")}</p>
        </div>
      </header>

      {#if runtimeSummary}
        <div class="components-list">
          {#each runtimeSummary.components as component}
            <div class="component-row">
              <div>
                <strong>{component.name}</strong>
                <span>
                  {#if component.error}
                    {component.error}
                  {:else if component.latencyMs !== undefined}
                    {t($locale, "status.latency", { ms: component.latencyMs })}
                  {:else}
                    {component.status}
                  {/if}
                </span>
              </div>
              <span class={`component-chip ${componentTone(component.status)}`}>{component.status}</span>
            </div>
          {/each}
        </div>
      {:else}
        <div class="empty-state">{t($locale, "status.runtimeUnavailable")}</div>
      {/if}
    </article>
  </section>
</section>

<style>
  .monitor-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .monitor-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 16px;
    flex-wrap: wrap;
  }

  h1,
  h2 {
    margin: 0;
    color: var(--color-slate-950);
  }

  h1 {
    font-size: 22px;
  }

  h2 {
    font-size: 16px;
  }

  p {
    margin: 4px 0 0;
    color: var(--color-slate-500);
    font-size: 13px;
  }

  .header-actions {
    display: inline-flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .header-actions > span {
    font-size: 12px;
    color: var(--color-slate-500);
  }

  button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    height: 34px;
    padding: 8px 12px;
    border: 0;
    border-radius: 8px;
    background: var(--color-primary-600);
    color: var(--color-white);
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }

  button:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }

  button svg {
    width: 13px;
    height: 13px;
  }

  .spinning {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .state-panel,
  .panel {
    border: 1px solid rgba(var(--rgb-slate-950), 0.08);
    border-radius: 8px;
    background: var(--color-white);
    box-shadow: 0 10px 24px rgba(var(--rgb-slate-950), 0.06);
  }

  .state-panel {
    display: grid;
    grid-template-columns: minmax(260px, 1.2fr) minmax(0, 2fr);
    gap: 16px;
    padding: 18px;
    border-left: 4px solid var(--state-color);
  }

  .state-panel.ok {
    --state-color: #22c55e;
  }

  .state-panel.warn {
    --state-color: #f59e0b;
  }

  .state-panel.error {
    --state-color: var(--color-danger-500);
  }

  .state-panel.muted {
    --state-color: var(--color-slate-400);
  }

  .state-copy {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .state-copy span,
  .state-metrics span,
  .runtime-kpis span,
  .resource-row span,
  .resource-meta,
  .component-row span,
  .signal-kpis {
    font-size: 12px;
    color: var(--color-slate-500);
  }

  .state-copy strong {
    font-size: 24px;
    color: var(--color-slate-950);
  }

  .state-copy p {
    margin: 0;
    font-size: 13px;
    color: var(--color-slate-600);
  }

  .state-metrics {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
  }

  .state-metrics > div,
  .runtime-kpis > div {
    display: flex;
    flex-direction: column;
    gap: 5px;
    min-width: 0;
    border: 1px solid var(--color-slate-200);
    border-radius: 8px;
    background: var(--color-slate-50);
    padding: 10px;
  }

  .state-metrics strong,
  .runtime-kpis strong,
  .resource-row strong {
    color: var(--color-slate-950);
    font-size: 18px;
    font-variant-numeric: tabular-nums;
  }

  .monitor-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.45fr) minmax(300px, 0.85fr);
    gap: 16px;
  }

  .runtime-layout {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 16px;
  }

  .panel {
    padding: 14px;
  }

  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 12px;
  }

  .panel-header strong {
    color: var(--color-slate-950);
    font-size: 18px;
  }

  .signal-table,
  .issues,
  .components-list,
  .resource-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .signal-line {
    display: grid;
    grid-template-columns: 110px minmax(220px, 1fr) 150px;
    gap: 12px;
    align-items: center;
    border: 1px solid var(--color-slate-200);
    border-radius: 8px;
    background: var(--color-slate-50);
    padding: 10px;
  }

  .signal-name {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .signal-name strong,
  .issue strong,
  .component-row strong {
    color: var(--color-slate-950);
    font-size: 13px;
  }

  .signal-kpis {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    font-variant-numeric: tabular-nums;
  }

  .negative {
    color: var(--color-danger-700);
  }

  .spark {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    align-items: end;
    gap: 3px;
    height: 32px;
  }

  .spark span {
    display: block;
    border-radius: 3px 3px 0 0;
    background: var(--color-primary-500);
    opacity: 0.72;
  }

  .signal-line.warn .spark span {
    background: #f59e0b;
  }

  .signal-line.error .spark span {
    background: var(--color-danger-500);
  }

  .issue {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 10px;
    align-items: flex-start;
    border: 1px solid var(--color-slate-200);
    border-radius: 8px;
    background: var(--color-slate-50);
    padding: 10px;
  }

  .issue.ok {
    border-color: #86efac;
    background: #f0fdf4;
  }

  .issue.warn {
    border-color: #fde68a;
    background: #fffbeb;
  }

  .issue.error {
    border-color: var(--color-danger-75);
    background: var(--color-danger-50);
  }

  .issue p {
    margin: 3px 0 0;
    font-size: 12px;
  }

  .status-dot {
    width: 9px;
    height: 9px;
    margin-top: 4px;
    flex: 0 0 auto;
    border-radius: 999px;
    background: var(--color-slate-400);
  }

  .status-dot.ok {
    background: #22c55e;
  }

  .status-dot.warn {
    background: #f59e0b;
  }

  .status-dot.error {
    background: var(--color-danger-500);
  }

  .runtime-kpis {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .runtime-kpis strong.attention {
    color: var(--color-danger-700);
  }

  .resource-row {
    display: flex;
    flex-direction: column;
    gap: 8px;
    border: 1px solid var(--color-slate-200);
    border-radius: 8px;
    background: var(--color-slate-50);
    padding: 10px;
  }

  .resource-row > div:first-child {
    display: flex;
    justify-content: space-between;
    gap: 10px;
    align-items: baseline;
  }

  .meter {
    height: 8px;
    overflow: hidden;
    border-radius: 999px;
    background: var(--color-slate-200);
  }

  .meter span {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: var(--color-primary-500);
  }

  .resource-meta {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
  }

  .component-row {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: flex-start;
    border: 1px solid var(--color-slate-200);
    border-radius: 8px;
    background: var(--color-slate-50);
    padding: 10px;
  }

  .component-row > div {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }

  .component-chip {
    padding: 4px 8px;
    border-radius: 999px;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    white-space: nowrap;
  }

  .component-chip.ok {
    color: #166534;
    background: #dcfce7;
  }

  .component-chip.warn {
    color: #854d0e;
    background: #fef9c3;
  }

  .component-chip.error {
    color: #991b1b;
    background: var(--color-danger-100);
  }

  .empty-state {
    border: 1px dashed var(--color-slate-300);
    border-radius: 8px;
    padding: 18px;
    color: var(--color-slate-500);
    font-size: 13px;
    text-align: center;
  }

  @media (max-width: 1050px) {
    .state-panel,
    .monitor-grid,
    .runtime-layout {
      grid-template-columns: 1fr;
    }

    .state-metrics {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 720px) {
    .state-metrics,
    .runtime-kpis {
      grid-template-columns: 1fr;
    }

    .signal-line {
      grid-template-columns: 1fr;
    }
  }
</style>
