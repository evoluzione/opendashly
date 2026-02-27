<script lang="ts">
  import { onMount } from "svelte";
  import { fetchRelated, fetchTraceSpans, type TraceSpan } from "../services/traces";

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
          : "Impossibile caricare gli span";
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
    return new Intl.DateTimeFormat("it-IT", {
      dateStyle: "short",
      timeStyle: "medium",
    }).format(date);
  }

  const maxDisplayMs = 60000;
  const sourcePalette = [
    "#0ea5e9",
    "#22c55e",
    "#f97316",
    "#ef4444",
    "#8b5cf6",
    "#14b8a6",
    "#eab308",
    "#6366f1",
  ];

  function durationMs(span: TraceSpan) {
    return Math.max(0, Math.round((span.duration ?? 0) / 1_000_000));
  }

  function formatDuration(ms: number) {
    if (ms > maxDisplayMs) return "> 60 s";
    if (ms < 1000) return `${ms} ms`;
    return `${(ms / 1000).toFixed(2)} s`;
  }

  function offsetMs(span: TraceSpan) {
    return Math.max(0, Math.round(toMs(span.startTime) - startMs));
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
    return span.source || span.service || "origine sconosciuta";
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
        message: attrs["exception.message"] || "Nessun messaggio",
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
        message: eventAttrs["exception.message"] || "Nessun messaggio",
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
        0: { kind: "internal", label: "Internal", icon: "⚙️", color: "#64748b" },
        1: { kind: "server", label: "Server", icon: "🖥️", color: "#16a34a" },
        2: { kind: "client", label: "Client", icon: "🌐", color: "#2563eb" },
        3: { kind: "producer", label: "Producer", icon: "📤", color: "#9333ea" },
        4: { kind: "consumer", label: "Consumer", icon: "📥", color: "#ea580c" },
      };
      return map[raw] ?? { kind: "unknown", label: `Kind ${raw}`, icon: "🏷️", color: "#94a3b8" };
    }

    const normalized = String(raw).toUpperCase();
    if (normalized.includes("PRODUCER")) return { kind: "producer", label: "Producer", icon: "📤", color: "#9333ea" };
    if (normalized.includes("CONSUMER")) return { kind: "consumer", label: "Consumer", icon: "📥", color: "#ea580c" };
    if (normalized.includes("SERVER")) return { kind: "server", label: "Server", icon: "🖥️", color: "#16a34a" };
    if (normalized.includes("CLIENT")) return { kind: "client", label: "Client", icon: "🌐", color: "#2563eb" };
    if (normalized.includes("INTERNAL")) return { kind: "internal", label: "Internal", icon: "⚙️", color: "#64748b" };

    return {
      kind: "unknown",
      label: normalized,
      icon: "🏷️",
      color: "#94a3b8",
    };
  }

  function extractIpv4Candidates(value: string): string[] {
    if (!value) return [];
    const matches = value.match(/\b(?:\d{1,3}\.){3}\d{1,3}\b/g) ?? [];
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
    <h4>Linea temporale degli span</h4>
    <p>Durata totale: {Math.round(rangeMs)} ms</p>
  </header>

  {#if loading}
    <p class="status">Caricamento span...</p>
  {:else if error}
    <p class="status error">{error}</p>
  {:else if spans.length === 0}
    <p class="status">Nessuno span disponibile.</p>
  {:else}
    <div class="timeline-layout" class:with-details={!!selectedSpan}>
      <div class="span-list">
        <div class="span-header">
          <span>Span</span>
          <span>Linea temporale</span>
          <span>Durata</span>
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
            >
              <div class="meta" style={`--depth:${row.depth}`}>
                <div class="meta-title">
                  <span class="depth-branch" style={`opacity:${row.depth > 0 ? 1 : 0}`}>↳</span>
                  <span class="source-dot" style={`background:${color}`}></span>
                  <span class="name">{span.name || "Span"}</span>
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
                      title={`Chiamata verso IP esterno: ${externalIp.ip} (${externalIp.sourceKey})`}
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
                      title={`Span parallelo con ${row.parallelSiblingCount} sibling nello stesso ramo`}
                    >
                      <svg class="parallel-icon" viewBox="0 0 14 14" aria-hidden="true" focusable="false">
                        <path d="M4 2v10M10 2v10" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
                        <path d="M4 4h3M10 10H7" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
                      </svg>
                      <span>x{row.parallelSiblingCount + 1}</span>
                    </span>
                  {/if}
                </div>
                <span class="service">{span.service || "servizio sconosciuto"}</span>
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
                <span class="bar-label">{formatDuration(offsetMs(span))}</span>
              </div>

              <div class="duration">{formatDuration(durationMs(span))}</div>
            </button>
          {/each}
        </div>
      </div>

      {#if selectedSpan}
        <aside class="span-details">
          <div class="details-header">
            <div class="details-title-row">
              <div>
                <h5>{selectedSpan.name || "Span senza nome"}</h5>
                <p>{selectedSpan.service || "Servizio sconosciuto"}</p>
              </div>
              <button
                type="button"
                class="details-close"
                on:click={() => (selectedSpanId = "")}
              >
                Nascondi dettagli
              </button>
            </div>
          </div>

          <div class="details-section">
            <span class="section-title">Generale</span>
            <div class="kv-grid">
              <div class="kv"><span>Trace ID</span><code>{shortId(selectedSpan.traceId)}</code></div>
              <div class="kv"><span>Span ID</span><code>{shortId(selectedSpan.spanId)}</code></div>
              <div class="kv"><span>Parent</span><code>{shortId(selectedSpan.parentSpanId)}</code></div>
              <div class="kv"><span>Source</span><code>{spanSource(selectedSpan)}</code></div>
              <div class="kv"><span>Status</span><code class:error={isErrorStatus(selectedSpan.status)}>{selectedSpan.status || "UNSET"}</code></div>
              <div class="kv"><span>Start</span><code>{formatTimestamp(selectedSpan.startTime)}</code></div>
              <div class="kv"><span>Durata</span><code>{formatDuration(durationMs(selectedSpan))}</code></div>
            </div>
          </div>

          <div class="details-section">
            <div class="section-head">
              <span class="section-title-wrap">
                <span class="section-title">Eccezioni</span>
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
                {showExceptions ? "Chiudi" : "Espandi"}
              </button>
            </div>
            {#if showExceptions}
              {#if selectedExceptions.length === 0}
                <p class="empty-section">Nessuna eccezione disponibile.</p>
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
                <span class="section-title">Eventi Span</span>
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
                {showSpanEvents ? "Chiudi" : "Espandi"}
              </button>
            </div>
            {#if showSpanEvents}
              {#if selectedEvents.length === 0}
                <p class="empty-section">Nessun evento disponibile nello span.</p>
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
            <div class="section-head">
              <span class="section-title-wrap">
                <span class="section-title">Logs</span>
                <span class="section-count" title={`Totale: ${selectedSpanLogs.length}`}>
                  <span>{selectedSpanLogs.length}</span>
                </span>
              </span>
              <button
                type="button"
                class="section-toggle"
                on:click={() => (showRelatedEvents = !showRelatedEvents)}
                aria-expanded={showRelatedEvents}
              >
                {showRelatedEvents ? "Chiudi" : "Espandi"}
              </button>
            </div>
            {#if showRelatedEvents}
              {#if selectedSpanLogs.length === 0}
                <p class="empty-section">Nessun log correlato trovato con questo span ID.</p>
              {:else}
                <div class="scroll-block">
                  {#each selectedSpanLogs as log}
                    <div class="log-card">
                      <div class="log-head">
                        <strong>{log.severity || "-"}</strong>
                        <span>{formatTimestamp(log.timestamp)}</span>
                      </div>
                      <pre class="log-body">{log.body || "-"}</pre>
                    </div>
                  {/each}
                </div>
              {/if}
            {/if}
          </div>

          <div class="details-section">
            <span class="section-title">Attributi</span>
            {#if !selectedSpan.attributes || Object.keys(selectedSpan.attributes).length === 0}
              <p class="empty-section">Nessun attributo disponibile.</p>
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
    color: #64748b;
  }

  header p {
    margin: 0;
    font-size: 12px;
    color: #94a3b8;
  }

  .status {
    margin: 0;
    font-size: 13px;
    color: #94a3b8;
  }

  .status.error {
    color: #b91c1c;
  }

  .timeline-layout {
    display: grid;
    grid-template-columns: 1fr;
    gap: 12px;
    min-height: 0;
    flex: 1;
  }

  .timeline-layout.with-details {
    grid-template-columns: minmax(420px, 1fr) minmax(340px, 0.8fr);
  }

  .span-list {
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 12px;
    background: #f8fafc;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-height: 0;
  }

  .span-header {
    display: grid;
    grid-template-columns: minmax(260px, 320px) 1fr 80px;
    gap: 12px;
    align-items: center;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #94a3b8;
    font-weight: 600;
    padding: 0 4px 8px;
    border-bottom: 1px solid rgba(148, 163, 184, 0.3);
  }

  .span-grid {
    display: flex;
    flex-direction: column;
    gap: 8px;
    overflow-y: auto;
    min-height: 0;
    padding-right: 4px;
  }

  .span-row {
    border: 1px solid transparent;
    background: white;
    border-radius: 10px;
    display: grid;
    grid-template-columns: minmax(260px, 320px) 1fr 80px;
    gap: 12px;
    align-items: center;
    text-align: left;
    padding: 10px;
    cursor: pointer;
  }

  .span-row:hover {
    border-color: rgba(59, 130, 246, 0.35);
  }

  .span-row.selected {
    border-color: #2563eb;
    box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.12);
  }

  .meta {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
    padding-left: min(calc(var(--depth, 0) * 10px), 48px);
  }

  .meta-title {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  .depth-branch {
    font-size: 10px;
    color: #94a3b8;
    flex-shrink: 0;
  }

  .source-dot {
    width: 8px;
    height: 8px;
    border-radius: 999px;
    flex-shrink: 0;
  }

  .name {
    font-size: 12px;
    font-weight: 600;
    color: #0f172a;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .kind-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 22px;
    padding: 2px 6px;
    border-radius: 999px;
    border: 1px solid;
    font-size: 11px;
    font-weight: 700;
    white-space: nowrap;
    line-height: 1;
  }

  .kind-icon-svg {
    width: 12px;
    height: 12px;
    display: block;
  }

  .external-ip-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 22px;
    font-size: 11px;
    color: #7c3aed;
    background: #f3e8ff;
    border: 1px solid #ddd6fe;
    padding: 2px 6px;
    border-radius: 999px;
    line-height: 1;
  }

  .external-ip-icon {
    width: 11px;
    height: 11px;
    display: block;
  }

  .parallel-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    min-width: 30px;
    font-size: 10px;
    font-weight: 700;
    color: #334155;
    background: #f8fafc;
    border: 1px solid #cbd5e1;
    padding: 2px 7px;
    border-radius: 999px;
    line-height: 1;
  }

  .parallel-icon {
    width: 11px;
    height: 11px;
    display: block;
    opacity: 0.85;
  }

  .service {
    font-size: 11px;
    color: #64748b;
    padding-left: 16px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .bar-track {
    position: relative;
    height: 18px;
    background: repeating-linear-gradient(
      90deg,
      rgba(148, 163, 184, 0.15),
      rgba(148, 163, 184, 0.15) 1px,
      transparent 1px,
      transparent 40px
    );
    border-radius: 999px;
    overflow: hidden;
  }

  .bar {
    position: absolute;
    top: 3px;
    height: 12px;
    border-radius: 999px;
  }

  .bar-error-event {
    position: absolute;
    top: 1px;
    transform: translateX(-50%);
    width: 14px;
    height: 14px;
    border-radius: 999px;
    background: #dc2626;
    color: #ffffff;
    border: 1px solid #ffffff;
    font-size: 10px;
    font-weight: 800;
    line-height: 12px;
    text-align: center;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
  }

  .bar-label {
    position: absolute;
    top: -16px;
    left: 4px;
    font-size: 9px;
    color: #94a3b8;
    font-weight: 600;
  }

  .duration {
    font-size: 11px;
    color: #475569;
    font-weight: 600;
    text-align: right;
    padding-right: 4px;
  }

  .span-details {
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 12px;
    background: #ffffff;
    padding: 12px;
    overflow-y: auto;
    min-height: 0;
  }

  .details-header {
    border-bottom: 1px solid #e2e8f0;
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
    color: #0f172a;
  }

  .details-header p {
    margin: 4px 0 0;
    font-size: 12px;
    color: #64748b;
  }

  .details-close {
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    color: #475569;
    font-size: 11px;
    font-weight: 600;
    border-radius: 8px;
    padding: 6px 8px;
    cursor: pointer;
    white-space: nowrap;
  }

  .details-close:hover {
    background: #eef2f7;
    border-color: #cbd5e1;
  }

  .details-section {
    margin-bottom: 12px;
  }

  .section-title {
    display: inline-block;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    color: #64748b;
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
    background: #eef2ff;
    border: 1px solid #c7d2fe;
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
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    color: #475569;
    font-size: 11px;
    font-weight: 600;
    border-radius: 8px;
    padding: 4px 8px;
    cursor: pointer;
    white-space: nowrap;
  }

  .section-toggle:hover {
    background: #eef2f7;
    border-color: #cbd5e1;
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
    color: #64748b;
  }

  code {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
    font-size: 11px;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    padding: 4px 6px;
    color: #1e293b;
    word-break: break-all;
  }

  code.error {
    color: #b91c1c;
    border-color: #fecaca;
    background: #fff1f2;
  }

  .empty-section {
    margin: 0;
    font-size: 12px;
    color: #94a3b8;
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
    border: 1px solid #e2e8f0;
    background: #f8fafc;
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
    color: #0f172a;
  }

  .exception-head span,
  .event-head span,
  .log-head span {
    font-size: 11px;
    color: #64748b;
  }

  .exception-card p {
    margin: 0;
    font-size: 12px;
    color: #334155;
  }

  .code-block,
  .log-body {
    margin: 0;
    white-space: pre-wrap;
    font-size: 11px;
    line-height: 1.4;
    color: #1e293b;
    background: #ffffff;
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
    color: #334155;
    word-break: break-all;
  }

  @media (max-width: 1120px) {
    .timeline-layout.with-details {
      grid-template-columns: 1fr;
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
