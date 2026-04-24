<script lang="ts">
  import { onMount } from "svelte";
  import { fetchRelated, fetchTraceSpans, type TraceSpan } from "../services/traces";
  import { getLocaleTag, locale, t } from "../lib/i18n";

  export let traceId: string;

  type RelatedLog = {
    timestamp?: string;
    severity?: string;
    body?: string;
    spanId?: string;
    logAttributes?: Record<string, unknown>;
  };

  type SpanException = {
    type: string;
    message: string;
    stacktrace?: string;
    source: string;
  };

  type SpanEvent = {
    name: string;
    timestamp?: string;
    attributes: Record<string, string>;
  };

  type ErrorEventMarker = {
    id: string;
    left: number;
    title: string;
  };

  type DisplaySpanRow = {
    span: TraceSpan;
    depth: number;
    parallelSiblingCount: number;
  };

  let spans: TraceSpan[] = [];
  let relatedLogs: RelatedLog[] = [];
  let loading = true;
  let error: string | null = null;
  let selectedSpanId = "";
  let showExceptions = false;
  let showSpanEvents = false;
  let showRelatedEvents = false;
  let previousSelectedSpanId = "";

  onMount(async () => {
    loading = true;
    error = null;

    const [spansResult, relatedResult] = await Promise.allSettled([
      fetchTraceSpans(traceId),
      fetchRelated(traceId),
    ]);

    if (spansResult.status === "rejected") {
      error =
        spansResult.reason instanceof Error
          ? spansResult.reason.message
          : t($locale, "correlation.loadError");
      loading = false;
      return;
    }

    spans = spansResult.value;

    if (relatedResult.status === "fulfilled") {
      relatedLogs = (relatedResult.value.logs ?? []) as RelatedLog[];
    }

    loading = false;
  });

  function toMs(value: string | Date | number | undefined) {
    if (value === undefined) return 0;
    return new Date(value).getTime();
  }

  function formatTimestamp(value: string | number | Date | undefined) {
    if (!value) return "-";
    const date = value instanceof Date ? value : new Date(value);
    return new Intl.DateTimeFormat(getLocaleTag($locale), {
      dateStyle: "short",
      timeStyle: "medium",
    }).format(date);
  }

  const maxDisplayMs = 60000;
  const sourcePalette = [
    "#0ea5e9",
    "var(--color-success-500)",
    "#f97316",
    "var(--color-danger-500)",
    "var(--color-primary-500)",
    "var(--color-cyan-400)",
    "#eab308",
    "var(--color-primary-600)",
  ];

  function durationMs(span: TraceSpan) {
    return Math.max(0, (span.duration ?? 0) / 1_000_000);
  }

  function formatDuration(ms: number) {
    if (ms > maxDisplayMs) return "> 60 s";
    if (ms >= 1000) return `${(ms / 1000).toFixed(2)} s`;
    if (ms >= 1) return `${Math.round(ms)} ms`;
    return `${Math.round(ms * 1000)} µs`;
  }

  function offsetMs(span: TraceSpan) {
    return Math.max(0, Math.round(toMs(span.startTime) - startMs));
  }

  function downloadJson(data: unknown, filename: string) {
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  $: validSpans = spans.filter(
    (span) => durationMs(span) > 0 && durationMs(span) <= maxDisplayMs,
  );
  $: rangeSpans = validSpans.length > 0 ? validSpans : spans;

  $: startMs = rangeSpans.length
    ? Math.min(...rangeSpans.map((span) => toMs(span.startTime)))
    : 0;
  $: endMs = rangeSpans.length
    ? Math.max(...rangeSpans.map((span) => toMs(span.endTime)))
    : 0;
  $: rangeMs = Math.max(1, endMs - startMs);

  $: selectedSpan = spans.find((span) => span.spanId === selectedSpanId) ?? null;

  $: if (selectedSpanId && !spans.some((span) => span.spanId === selectedSpanId)) {
    selectedSpanId = "";
  }

  $: if (selectedSpanId !== previousSelectedSpanId) {
    previousSelectedSpanId = selectedSpanId;
    showExceptions = false;
    showSpanEvents = false;
    showRelatedEvents = false;
  }

  $: selectedSpanLogs = selectedSpan ? relatedLogsForSpan(selectedSpan) : [];
  $: selectedExceptions = selectedSpan ? collectSpanExceptions(selectedSpan) : [];
  $: selectedEvents = selectedSpan ? collectSpanEvents(selectedSpan) : [];
  $: displayRows = buildDisplayRows(spans);

  function barStyle(span: TraceSpan) {
    const color = colorForSource(spanSource(span));
    const left = ((toMs(span.startTime) - startMs) / rangeMs) * 100;
    const rawWidth =
      (Math.max(0, toMs(span.endTime) - toMs(span.startTime)) / rangeMs) * 100;
    const cappedWidth = Math.min(rawWidth, (maxDisplayMs / rangeMs) * 100);
    const width = cappedWidth > 0 ? cappedWidth : rawWidth;
    return `left:${left}%;width:${Math.max(0.5, width)}%;background:${color}`;
  }

  function spanSource(span: TraceSpan) {
    return span.source || span.service || t($locale, "traceTimeline.unknownServiceLower");
  }

  function colorForSource(value: string) {
    let hash = 0;
    for (let i = 0; i < value.length; i += 1) {
      hash = (hash << 5) - hash + value.charCodeAt(i);
      hash |= 0;
    }
    return sourcePalette[Math.abs(hash) % sourcePalette.length];
  }

  function isErrorStatus(status?: string) {
    if (!status) return false;
    const normalized = status.toUpperCase();
    return normalized === "ERROR" || normalized === "STATUS_CODE_ERROR" || normalized === "2";
  }

  function shortId(id?: string) {
    if (!id) return "-";
    return id.length > 14 ? `${id.slice(0, 8)}...${id.slice(-4)}` : id;
  }

  function relatedLogsForSpan(span: TraceSpan): RelatedLog[] {
    if (!span.spanId) return [];
    return relatedLogs.filter((log) => log.spanId === span.spanId);
  }

  function collectSpanEvents(span: TraceSpan): SpanEvent[] {
    const events = span.events ?? [];
    return events.map((event) => ({
      name: event.name || "event",
      timestamp: event.timestamp || event.time,
      attributes: event.attributes ?? {},
    }));
  }

  function collectSpanExceptions(span: TraceSpan): SpanException[] {
    const exceptions: SpanException[] = [];

    const attrs = span.attributes ?? {};
    if (attrs["exception.type"] || attrs["exception.message"] || attrs["exception.stacktrace"]) {
      exceptions.push({
        type: attrs["exception.type"] || "Unknown",
        message: attrs["exception.message"] || "-",
        stacktrace: attrs["exception.stacktrace"],
        source: "attributes",
      });
    }

    const events = span.events ?? [];
    for (const event of events) {
      const eventAttrs = event.attributes ?? {};
      const hasExceptionData =
        event.name === "exception" ||
        !!eventAttrs["exception.type"] ||
        !!eventAttrs["exception.message"] ||
        !!eventAttrs["exception.stacktrace"];
      if (!hasExceptionData) continue;

      exceptions.push({
        type: eventAttrs["exception.type"] || "Unknown",
        message: eventAttrs["exception.message"] || "-",
        stacktrace: eventAttrs["exception.stacktrace"],
        source: `event:${event.name || "exception"}`,
      });
    }

    const seen = new Set<string>();
    return exceptions.filter((item) => {
      const key = `${item.type}|${item.message}|${item.stacktrace || ""}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
  }

  function parseEventTimestampMs(event: SpanEvent): number | null {
    const attrs = event.attributes ?? {};
    const raw =
      event.timestamp ??
      attrs["event.time"] ??
      attrs["event.timestamp"] ??
      attrs["event.time_unix_nano"] ??
      attrs["timeUnixNano"] ??
      "";

    if (!raw) return null;

    const asNumber = typeof raw === "number" ? raw : Number(String(raw).trim());
    if (Number.isFinite(asNumber)) {
      // Heuristic for timestamps in seconds / milliseconds / nanoseconds.
      if (asNumber > 1e15) return asNumber / 1_000_000;
      if (asNumber > 1e12) return asNumber;
      if (asNumber > 1e9) return asNumber * 1000;
    }

    const parsed = Date.parse(String(raw));
    return Number.isFinite(parsed) ? parsed : null;
  }

  function isErrorEvent(event: SpanEvent): boolean {
    const name = (event.name || "").toLowerCase();
    if (name.includes("exception") || name.includes("error")) return true;

    const attrs = event.attributes ?? {};
    return !!(
      attrs["exception.type"] ||
      attrs["exception.message"] ||
      attrs["error"] ||
      attrs["error.type"] ||
      attrs["error.message"]
    );
  }

  function errorEventMarkers(span: TraceSpan): ErrorEventMarker[] {
    const events = collectSpanEvents(span);
    if (events.length === 0) return [];

    return events
      .filter(isErrorEvent)
      .map((event, index) => {
        const whenMs = parseEventTimestampMs(event);
        if (whenMs === null) return null;
        const pct = ((whenMs - startMs) / rangeMs) * 100;
        const left = Math.max(0, Math.min(100, pct));
        return {
          id: `${span.spanId || "span"}-${index}`,
          left,
          title: `${event.name || "error"} • ${formatTimestamp(whenMs)}`,
        };
      })
      .filter((item): item is ErrorEventMarker => item !== null);
  }

  function formatValue(value: unknown) {
    if (value === null || value === undefined) return "-";
    if (typeof value === "object") {
      try {
        return JSON.stringify(value);
      } catch {
        return String(value);
      }
    }
    return String(value);
  }

  interface SpanKindResult {
    kind: string;
    label: string;
    icon: string;
    color: string;
  }

  function spanKindInfo(span: TraceSpan): SpanKindResult | null {
    const raw = span.spanKind;
    if (raw === null || raw === undefined || raw === "") {
      return null;
    }

    if (typeof raw === "number") {
      const map: Record<number, SpanKindResult> = {
        0: { kind: "internal", label: "Internal", icon: "⚙️", color: "var(--color-slate-500)" },
        1: { kind: "server", label: "Server", icon: "🖥️", color: "#16a34a" },
        2: { kind: "client", label: "Client", icon: "🌐", color: "var(--color-info-600)" },
        3: { kind: "producer", label: "Producer", icon: "📤", color: "#9333ea" },
        4: { kind: "consumer", label: "Consumer", icon: "📥", color: "#ea580c" },
      };
      return map[raw] ?? { kind: "unknown", label: `Kind ${raw}`, icon: "🏷️", color: "var(--color-slate-400)" };
    }

    const normalized = String(raw).toUpperCase();
    if (normalized.includes("PRODUCER")) return { kind: "producer", label: "Producer", icon: "📤", color: "#9333ea" };
    if (normalized.includes("CONSUMER")) return { kind: "consumer", label: "Consumer", icon: "📥", color: "#ea580c" };
    if (normalized.includes("SERVER")) return { kind: "server", label: "Server", icon: "🖥️", color: "#16a34a" };
    if (normalized.includes("CLIENT")) return { kind: "client", label: "Client", icon: "🌐", color: "var(--color-info-600)" };
    if (normalized.includes("INTERNAL")) return { kind: "internal", label: "Internal", icon: "⚙️", color: "var(--color-slate-500)" };

    return {
      kind: "unknown",
      label: normalized,
      icon: "🏷️",
      color: "var(--color-slate-400)",
    };
  }

  function extractIpv4Candidates(value: string): string[] {
    if (!value) return [];
    const matches: string[] = value.match(/\b(?:\d{1,3}\.){3}\d{1,3}\b/g) ?? [];
    return matches.filter((ip) => {
      const parts = ip.split(".").map((part) => Number(part));
      return parts.length === 4 && parts.every((part) => Number.isInteger(part) && part >= 0 && part <= 255);
    });
  }

  function isPrivateOrReservedIpv4(ip: string): boolean {
    const [a, b] = ip.split(".").map((part) => Number(part));
    if (a === 10) return true;
    if (a === 127) return true;
    if (a === 0) return true;
    if (a === 169 && b === 254) return true;
    if (a === 172 && b >= 16 && b <= 31) return true;
    if (a === 192 && b === 168) return true;
    if (a === 100 && b >= 64 && b <= 127) return true;
    if (a >= 224) return true;
    return false;
  }

  function externalIpInfo(span: TraceSpan): { ip: string; sourceKey: string } | null {
    const attrs = span.attributes ?? {};
    const candidateKeys = [
      "net.peer.ip",
      "peer.ip",
      "network.peer.address",
      "server.address",
      "http.host",
      "client.address",
      "url.full",
      "http.url",
      "http.target",
    ];

    for (const key of candidateKeys) {
      const raw = attrs[key];
      if (!raw) continue;
      const candidates = extractIpv4Candidates(String(raw));
      const externalIp = candidates.find((ip) => !isPrivateOrReservedIpv4(ip));
      if (externalIp) {
        return { ip: externalIp, sourceKey: key };
      }
    }

    return null;
  }

  function toggleSpanDetails(spanId: string) {
    selectedSpanId = selectedSpanId === spanId ? "" : spanId;
  }

  function normalizedParentId(span: TraceSpan): string | null {
    const value = (span.parentSpanId || "").trim();
    return value ? value : null;
  }

  function hasTimeOverlap(a: TraceSpan, b: TraceSpan): boolean {
    const aStart = toMs(a.startTime);
    const aEnd = toMs(a.endTime);
    const bStart = toMs(b.startTime);
    const bEnd = toMs(b.endTime);
    return aStart < bEnd && bStart < aEnd;
  }

  function buildDisplayRows(input: TraceSpan[]): DisplaySpanRow[] {
    if (input.length === 0) return [];

    const byId = new Map<string, TraceSpan>();
    for (const span of input) {
      if (span.spanId) byId.set(span.spanId, span);
    }

    const children = new Map<string, TraceSpan[]>();
    const roots: TraceSpan[] = [];
    for (const span of input) {
      const parentId = normalizedParentId(span);
      if (!parentId || !byId.has(parentId)) {
        roots.push(span);
        continue;
      }
      const siblings = children.get(parentId) ?? [];
      siblings.push(span);
      children.set(parentId, siblings);
    }

    const byStartThenDuration = (left: TraceSpan, right: TraceSpan) => {
      const startDiff = toMs(left.startTime) - toMs(right.startTime);
      if (startDiff !== 0) return startDiff;
      const durDiff = durationMs(right) - durationMs(left);
      if (durDiff !== 0) return durDiff;
      return (left.spanId || "").localeCompare(right.spanId || "");
    };

    roots.sort(byStartThenDuration);
    for (const [, siblingList] of children) {
      siblingList.sort(byStartThenDuration);
    }

    const parallelCounts = new Map<string, number>();
    for (const [, siblingList] of children) {
      for (let i = 0; i < siblingList.length; i += 1) {
        let overlaps = 0;
        for (let j = 0; j < siblingList.length; j += 1) {
          if (i === j) continue;
          if (hasTimeOverlap(siblingList[i], siblingList[j])) overlaps += 1;
        }
        parallelCounts.set(siblingList[i].spanId, overlaps);
      }
    }

    const result: DisplaySpanRow[] = [];
    const visited = new Set<string>();
    const walk = (span: TraceSpan, depth: number) => {
      if (!span.spanId || visited.has(span.spanId)) return;
      visited.add(span.spanId);
      result.push({
        span,
        depth,
        parallelSiblingCount: parallelCounts.get(span.spanId) ?? 0,
      });
      const nested = children.get(span.spanId) ?? [];
      for (const child of nested) {
        walk(child, depth + 1);
      }
    };

    for (const root of roots) {
      walk(root, 0);
    }

    for (const span of input.sort(byStartThenDuration)) {
      if (span.spanId && !visited.has(span.spanId)) {
        walk(span, 0);
      }
    }

    return result;
  }
</script>

<section class="timeline">
  <header>
    <h4>{t($locale, "traceTimeline.title")}</h4>
    <p>{t($locale, "traceTimeline.totalDuration", { ms: Math.round(rangeMs) })}</p>
  </header>

  {#if loading}
    <p class="status">{t($locale, "traceTimeline.loading")}</p>
  {:else if error}
    <p class="status error">{error}</p>
  {:else if spans.length === 0}
    <p class="status">{t($locale, "traceTimeline.none")}</p>
  {:else}
    <div class="timeline-layout" class:with-details={!!selectedSpan}>
      <div class="span-list">
        <div class="span-header">
          <span>{t($locale, "traceTimeline.spanColumn")}</span>
          <span>{t($locale, "traceTimeline.timelineColumn")}</span>
          <span>{t($locale, "traceTimeline.durationColumn")}</span>
        </div>

        <div class="span-grid">
          {#each displayRows as row}
            {@const span = row.span}
            {@const source = spanSource(span)}
            {@const kind = spanKindInfo(span)}
            {@const externalIp = externalIpInfo(span)}
            {@const color = colorForSource(source)}
            {@const eventMarkers = errorEventMarkers(span)}
            <button
              type="button"
              class="span-row"
              class:selected={span.spanId === selectedSpanId}
              on:click={() => toggleSpanDetails(span.spanId)}
              title={offsetMs(span) > 0
                ? t($locale, "traceTimeline.startOffset", { ms: formatDuration(offsetMs(span)) })
                : undefined}
            >
              <div class="meta" style={`--depth:${row.depth}`}>
                <span class="depth-branch" style={`opacity:${row.depth > 0 ? 1 : 0}`}>↳</span>
                <span class="source-dot" style={`background:${color}`}></span>
                <span class="name">{span.name || t($locale, "traceTimeline.span")}</span>
                {#if kind && kind.kind !== "internal"}
                  <span
                    class="kind-badge"
                    title={kind.label}
                    style={`background:${kind.color}20;color:${kind.color};border-color:${kind.color}40`}
                  >
                    {#if kind.kind === "producer"}
                      <svg class="kind-icon-svg" viewBox="0 0 14 14" aria-hidden="true" focusable="false">
                        <rect x="1.75" y="3" width="5.5" height="8" rx="1.2" fill="none" stroke="currentColor" stroke-width="1.3"/>
                        <path d="M6.5 7h5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                        <path d="M9.5 4.5 12 7 9.5 9.5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                      </svg>
                    {:else if kind.kind === "consumer"}
                      <svg class="kind-icon-svg" viewBox="0 0 14 14" aria-hidden="true" focusable="false">
                        <rect x="6.75" y="3" width="5.5" height="8" rx="1.2" fill="none" stroke="currentColor" stroke-width="1.3"/>
                        <path d="M7.5 7h-5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                        <path d="M4.5 9.5 2 7l2.5-2.5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                      </svg>
                    {:else}
                      {kind.icon}
                    {/if}
                  </span>
                {/if}
                {#if externalIp}
                  <span
                    class="external-ip-badge"
                    title={t($locale, "traceTimeline.callToExternalIp", { ip: externalIp.ip, source: externalIp.sourceKey })}
                  >
                    <svg class="external-ip-icon" viewBox="0 0 14 14" aria-hidden="true" focusable="false">
                      <path d="M2.5 11.5 11.5 2.5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                      <path d="M8.5 2.5h3v3" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                      <circle cx="4" cy="10" r="2.2" fill="none" stroke="currentColor" stroke-width="1.3"/>
                    </svg>
                  </span>
                {/if}
                {#if row.parallelSiblingCount > 0}
                  <span
                    class="parallel-badge"
                    title={t($locale, "traceTimeline.parallelSpan", { count: row.parallelSiblingCount })}
                  >
                    <svg class="parallel-icon" viewBox="0 0 14 14" aria-hidden="true" focusable="false">
                      <path d="M4 2v10M10 2v10" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
                      <path d="M4 4h3M10 10H7" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
                    </svg>
                    <span>x{row.parallelSiblingCount + 1}</span>
                  </span>
                {/if}
                <span class="service">{span.service || t($locale, "traceTimeline.unknownServiceLower")}</span>
              </div>

              <div class="bar-track">
                <div class="bar" style={barStyle(span)}></div>
                {#each eventMarkers as marker}
                  <span
                    class="bar-error-event"
                    style={`left:${marker.left}%`}
                    title={marker.title}
                    aria-label={marker.title}
                    >!</span
                  >
                {/each}
              </div>

              <div class="duration">
                <span class="duration-value">{formatDuration(durationMs(span))}</span>
              </div>
            </button>
          {/each}
        </div>
      </div>

      {#if selectedSpan}
        <aside class="span-details">
          <div class="details-header">
            <div class="details-title-row">
              <div>
                <h5>{selectedSpan.name || t($locale, "traceTimeline.unnamedSpan")}</h5>
                <p>{selectedSpan.service || t($locale, "traceTimeline.unknownService")}</p>
              </div>
              <div class="details-actions">
                <button
                  type="button"
                  class="details-close icon-btn"
                  on:click={() => downloadJson(selectedSpan, `span-${selectedSpan.spanId ?? 'unknown'}.json`)}
                  title={t($locale, "traceTimeline.downloadSpan")}
                  aria-label={t($locale, "traceTimeline.downloadSpan")}
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
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
                  class="details-close icon-btn"
                  on:click={() => (selectedSpanId = "")}
                  title={t($locale, "common.close")}
                  aria-label={t($locale, "common.close")}
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
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
            </div>
          </div>

          <div class="details-section">
            <span class="section-title">{t($locale, "traceTimeline.general")}</span>
            <div class="kv-grid">
              <div class="kv"><span>{t($locale, "logs.traceId")}</span><code>{shortId(selectedSpan.traceId)}</code></div>
              <div class="kv"><span>{t($locale, "logs.spanId")}</span><code>{shortId(selectedSpan.spanId)}</code></div>
              <div class="kv"><span>{t($locale, "traceTimeline.parent")}</span><code>{shortId(selectedSpan.parentSpanId)}</code></div>
              <div class="kv"><span>{t($locale, "traceTimeline.source")}</span><code>{spanSource(selectedSpan)}</code></div>
              <div class="kv"><span>{t($locale, "traceTimeline.status")}</span><code class:error={isErrorStatus(selectedSpan.status)}>{selectedSpan.status || "UNSET"}</code></div>
              <div class="kv"><span>{t($locale, "traceTimeline.start")}</span><code>{formatTimestamp(selectedSpan.startTime)}</code></div>
              <div class="kv"><span>{t($locale, "traceTimeline.duration")}</span><code>{formatDuration(durationMs(selectedSpan))}</code></div>
            </div>
          </div>

          <div class="details-section">
            <div class="section-head">
              <span class="section-title-wrap">
                <span class="section-title">{t($locale, "traceTimeline.exceptions")}</span>
                <span class="section-count" title={`Totale: ${selectedExceptions.length}`}>
                  <span>{selectedExceptions.length}</span>
                </span>
              </span>
              <button
                type="button"
                class="section-toggle"
                on:click={() => (showExceptions = !showExceptions)}
                aria-expanded={showExceptions}
              >
                {showExceptions ? t($locale, "common.close") : t($locale, "traceTimeline.expand")}
              </button>
            </div>
            {#if showExceptions}
              {#if selectedExceptions.length === 0}
                <p class="empty-section">{t($locale, "traceTimeline.noExceptions")}</p>
              {:else}
                <div class="scroll-block">
                  {#each selectedExceptions as item}
                    <div class="exception-card">
                      <div class="exception-head">
                        <strong>{item.type}</strong>
                        <span>{item.source}</span>
                      </div>
                      <p>{item.message}</p>
                      {#if item.stacktrace}
                        <pre class="code-block">{item.stacktrace}</pre>
                      {/if}
                    </div>
                  {/each}
                </div>
              {/if}
            {/if}
          </div>

          <div class="details-section">
            <div class="section-head">
              <span class="section-title-wrap">
                <span class="section-title">{t($locale, "traceTimeline.events")}</span>
                <span class="section-count" title={`Totale: ${selectedEvents.length}`}>
                  <span>{selectedEvents.length}</span>
                </span>
              </span>
              <button
                type="button"
                class="section-toggle"
                on:click={() => (showSpanEvents = !showSpanEvents)}
                aria-expanded={showSpanEvents}
              >
                {showSpanEvents ? t($locale, "common.close") : t($locale, "traceTimeline.expand")}
              </button>
            </div>
            {#if showSpanEvents}
              {#if selectedEvents.length === 0}
                <p class="empty-section">{t($locale, "traceTimeline.noEvents")}</p>
              {:else}
                <div class="scroll-block">
                  {#each selectedEvents as event}
                    <div class="event-card">
                      <div class="event-head">
                        <strong>{event.name}</strong>
                        <span>{formatTimestamp(event.timestamp)}</span>
                      </div>
                      {#if Object.keys(event.attributes).length > 0}
                        <div class="attributes-list">
                          {#each Object.entries(event.attributes) as [key, value]}
                            <div class="attr-row">
                              <span class="attr-key">{key}</span>
                              <span class="attr-value">{value}</span>
                            </div>
                          {/each}
                        </div>
                      {/if}
                    </div>
                  {/each}
                </div>
              {/if}
            {/if}
          </div>

          <div class="details-section">
            <span class="section-title">{t($locale, "traceTimeline.attributes")}</span>
            {#if !selectedSpan.attributes || Object.keys(selectedSpan.attributes).length === 0}
              <p class="empty-section">{t($locale, "traceTimeline.noAttributes")}</p>
            {:else}
              <div class="scroll-block attributes-scroll">
                {#each Object.entries(selectedSpan.attributes) as [key, value]}
                  <div class="attr-row">
                    <span class="attr-key">{key}</span>
                    <span class="attr-value">{formatValue(value)}</span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </aside>
      {/if}
    </div>
  {/if}
</section>

<style>
  .timeline {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-bottom: 24px;
    height: 100%;
    min-height: 0;
  }

  header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
  }

  h4 {
    margin: 0;
    font-size: 14px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--color-slate-500);
  }

  header p {
    margin: 0;
    font-size: 12px;
    color: var(--color-slate-400);
  }

  .status {
    margin: 0;
    font-size: 13px;
    color: var(--color-slate-400);
  }

  .status.error {
    color: var(--color-danger-700);
  }

  .timeline-layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: minmax(0, 1fr);
    gap: 12px;
    min-height: 0;
    min-width: 0;
    flex: 1;
  }

  .timeline-layout.with-details {
    grid-template-columns: minmax(0, 2fr) minmax(320px, 1fr);
  }

  .span-list {
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 12px;
    background: var(--color-slate-50);
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-height: 0;
    min-width: 0;
    overflow: hidden;
  }

  .span-header {
    display: grid;
    grid-template-columns: minmax(240px, 320px) 1fr 80px;
    gap: 10px;
    align-items: center;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--color-slate-400);
    font-weight: 600;
    padding: 0 14px 6px 4px;
    border-bottom: 1px solid rgba(148, 163, 184, 0.3);
  }

  .span-header > :last-child {
    text-align: right;
  }

  .span-grid {
    display: flex;
    flex-direction: column;
    gap: 4px;
    overflow-y: auto;
    min-height: 0;
    min-width: 0;
    padding-right: 6px;
    scrollbar-gutter: stable;
  }

  .span-row {
    appearance: none;
    border: 1px solid transparent;
    background: transparent;
    border-radius: 4px;
    display: grid;
    grid-template-columns: minmax(240px, 320px) 1fr 80px;
    gap: 10px;
    align-items: center;
    text-align: left;
    padding: 7px 8px;
    cursor: pointer;
    font: inherit;
    line-height: 1.25;
    min-height: 0;
    min-width: 0;
    min-height: 30px;
    outline: none;
  }

  .span-row:hover {
    background: rgba(59, 130, 246, 0.03);
    border-color: transparent;
  }

  .span-row.selected {
    background: rgba(37, 99, 235, 0.06);
    border-color: var(--color-info-600);
  }

  .span-row:focus-visible {
    box-shadow: 0 0 0 2px rgba(var(--rgb-primary-600), 0.18);
  }

  .meta {
    display: flex;
    flex-direction: row;
    align-items: center;
    gap: 4px;
    min-width: 0;
    padding-left: min(calc(var(--depth, 0) * 10px), 48px);
  }

  .depth-branch {
    font-size: 9px;
    color: var(--color-slate-400);
    flex-shrink: 0;
  }

  .source-dot {
    width: 6px;
    height: 6px;
    border-radius: 999px;
    flex-shrink: 0;
  }

  .name {
    font-size: 12px;
    font-weight: 500;
    color: var(--color-slate-950);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .kind-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 18px;
    padding: 1px 4px;
    border-radius: 999px;
    border: 1px solid;
    font-size: 10px;
    font-weight: 700;
    white-space: nowrap;
    line-height: 1;
    flex-shrink: 0;
  }

  .kind-icon-svg {
    width: 10px;
    height: 10px;
    display: block;
  }

  .external-ip-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 18px;
    font-size: 10px;
    color: var(--color-primary-450);
    background: var(--color-primary-25);
    border: 1px solid #ddd6fe;
    padding: 1px 4px;
    border-radius: 999px;
    line-height: 1;
    flex-shrink: 0;
  }

  .external-ip-icon {
    width: 10px;
    height: 10px;
    display: block;
  }

  .parallel-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 2px;
    min-width: 24px;
    font-size: 9px;
    font-weight: 700;
    color: var(--color-slate-600);
    background: var(--color-slate-100);
    border: 1px solid var(--color-slate-300);
    padding: 1px 5px;
    border-radius: 999px;
    line-height: 1;
    flex-shrink: 0;
  }

  .parallel-icon {
    width: 9px;
    height: 9px;
    display: block;
    opacity: 0.85;
  }

  .service {
    font-size: 10px;
    color: var(--color-slate-400);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex-shrink: 0;
    margin-left: 2px;
  }

  .bar-track {
    position: relative;
    height: 12px;
    background: repeating-linear-gradient(
      90deg,
      rgba(148, 163, 184, 0.12),
      rgba(148, 163, 184, 0.12) 1px,
      transparent 1px,
      transparent 40px
    );
    border-radius: 999px;
    overflow: visible;
  }

  .bar {
    position: absolute;
    top: 2px;
    height: 8px;
    border-radius: 999px;
  }

  .bar-error-event {
    position: absolute;
    top: 0;
    transform: translateX(-50%);
    width: 12px;
    height: 12px;
    border-radius: 999px;
    background: var(--color-danger-600);
    color: var(--color-white);
    border: 1px solid var(--color-white);
    font-size: 9px;
    font-weight: 800;
    line-height: 10px;
    text-align: center;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
  }

  .duration {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    text-align: right;
    white-space: nowrap;
  }

  .duration-value {
    font-size: 11px;
    color: var(--color-slate-700);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }

  .span-details {
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 12px;
    background: var(--color-white);
    padding: 12px;
    overflow-y: auto;
    min-height: 0;
  }

  .details-header {
    border-bottom: 1px solid var(--color-slate-200);
    padding-bottom: 10px;
    margin-bottom: 12px;
  }

  .details-title-row {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
  }

  .details-header h5 {
    margin: 0;
    font-size: 14px;
    color: var(--color-slate-950);
  }

  .details-header p {
    margin: 4px 0 0;
    font-size: 12px;
    color: var(--color-slate-500);
  }

  .details-actions {
    display: flex;
    gap: 6px;
    flex-shrink: 0;
  }

  .details-close {
    border: 1px solid var(--color-slate-200);
    background: var(--color-slate-50);
    color: var(--color-slate-600);
    font-size: 11px;
    font-weight: 600;
    border-radius: 8px;
    padding: 6px 8px;
    cursor: pointer;
    white-space: nowrap;
  }

  .details-close.icon-btn {
    padding: 6px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  .details-close:hover {
    background: var(--color-slate-75);
    border-color: var(--color-slate-300);
  }

  .details-section {
    margin-bottom: 12px;
  }

  .section-title {
    display: inline-block;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    color: var(--color-slate-500);
    margin-bottom: 6px;
  }

  .section-title-wrap {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .section-count {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 6px;
    border-radius: 999px;
    background: var(--color-primary-50);
    border: 1px solid var(--color-primary-200);
    color: #3730a3;
    font-size: 10px;
    font-weight: 700;
  }

  .section-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 6px;
  }

  .section-toggle {
    border: 1px solid var(--color-slate-200);
    background: var(--color-slate-50);
    color: var(--color-slate-600);
    font-size: 11px;
    font-weight: 600;
    border-radius: 8px;
    padding: 4px 8px;
    cursor: pointer;
    white-space: nowrap;
  }

  .section-toggle:hover {
    background: var(--color-slate-75);
    border-color: var(--color-slate-300);
  }

  .kv-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 6px;
  }

  .kv {
    display: grid;
    grid-template-columns: 90px 1fr;
    align-items: center;
    gap: 8px;
    font-size: 12px;
  }

  .kv span {
    color: var(--color-slate-500);
  }

  code {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
    font-size: 11px;
    background: var(--color-slate-50);
    border: 1px solid var(--color-slate-200);
    border-radius: 6px;
    padding: 4px 6px;
    color: var(--color-slate-900);
    word-break: break-all;
  }

  code.error {
    color: var(--color-danger-700);
    border-color: var(--color-danger-75);
    background: #fff1f2;
  }

  .empty-section {
    margin: 0;
    font-size: 12px;
    color: var(--color-slate-400);
  }

  .scroll-block {
    max-height: 180px;
    overflow: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-right: 4px;
  }

  .attributes-scroll {
    max-height: 220px;
  }

  .exception-card,
  .event-card,
  .log-card {
    border: 1px solid var(--color-slate-200);
    background: var(--color-slate-50);
    border-radius: 8px;
    padding: 8px;
  }

  .exception-head,
  .event-head,
  .log-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 6px;
  }

  .exception-head strong,
  .event-head strong,
  .log-head strong {
    font-size: 12px;
    color: var(--color-slate-950);
  }

  .exception-head span,
  .event-head span,
  .log-head span {
    font-size: 11px;
    color: var(--color-slate-500);
  }

  .exception-card p {
    margin: 0;
    font-size: 12px;
    color: var(--color-slate-700);
  }

  .code-block,
  .log-body {
    margin: 0;
    white-space: pre-wrap;
    font-size: 11px;
    line-height: 1.4;
    color: var(--color-slate-900);
    background: var(--color-white);
    border: 1px solid #dbe3ee;
    border-radius: 6px;
    padding: 8px;
    overflow: auto;
    user-select: text;
  }

  .attributes-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .attr-row {
    display: grid;
    grid-template-columns: minmax(120px, 170px) 1fr;
    gap: 8px;
    font-size: 11px;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  }

  .attr-key {
    color: #0ea5e9;
    word-break: break-all;
  }

  .attr-value {
    color: var(--color-slate-700);
    word-break: break-all;
  }

  @media (max-width: 1120px) {
    .timeline-layout.with-details {
      grid-template-columns: minmax(0, 1fr);
    }

    .span-details {
      max-height: 50vh;
    }
  }

  @media (max-width: 720px) {
    .span-header {
      display: none;
    }

    .span-row {
      grid-template-columns: 1fr;
      gap: 8px;
    }

    .kv {
      grid-template-columns: 1fr;
      gap: 4px;
    }

    .attr-row {
      grid-template-columns: 1fr;
      gap: 2px;
    }
  }
</style>
