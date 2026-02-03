<script lang="ts">
  import { onMount } from "svelte";
  import { fetchTraceSpans } from "../services/traces";

  export let traceId: string;

  let spans: any[] = [];
  let loading = true;
  let error: string | null = null;

  onMount(async () => {
    loading = true;
    error = null;
    try {
      spans = await fetchTraceSpans(traceId);
    } catch (err) {
      error =
        err instanceof Error ? err.message : "Impossibile caricare gli span";
    } finally {
      loading = false;
    }
  });

  function toMs(value: string | Date) {
    return new Date(value).getTime();
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

  function durationMs(span: any) {
    return Math.max(0, Math.round(span.duration / 1_000_000));
  }

  function formatDuration(ms: number) {
    if (ms > maxDisplayMs) return "> 60 s";
    if (ms < 1000) return `${ms} ms`;
    return `${(ms / 1000).toFixed(2)} s`;
  }

  function offsetMs(span: any) {
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

  let hoveredSpan: any = null;
  let tooltipX = 0;
  let tooltipY = 0;

  function handleContainerMouseMove(e: MouseEvent) {
    const target = e.target as HTMLElement;
    const bar = target.closest(".bar");
    if (bar instanceof HTMLElement && bar.dataset.index) {
      const index = parseInt(bar.dataset.index, 10);
      const span = spans[index];
      if (span) {
        handleMouseMove(e, span);
        return;
      }
    }
    handleMouseLeave();
  }

  function handleMouseMove(e: MouseEvent, span: any) {
    hoveredSpan = span;
    tooltipX = e.clientX + 16;
    tooltipY = e.clientY + 16;

    // Boundary check (simple) - if too close to right edge, move left
    if (window.innerWidth - tooltipX < 300) {
      tooltipX = e.clientX - 316;
    }
    // Boundary check - if too close to bottom, move up
    if (window.innerHeight - tooltipY < 200) {
      tooltipY = e.clientY - 216;
    }
  }

  function handleMouseLeave() {
    hoveredSpan = null;
  }

  function barStyle(span: any) {
    // Error highlighting logic
    const isError =
      span.status === "ERROR" ||
      span.status === "STATUS_CODE_ERROR" ||
      span.status === "2";
    const color = isError ? "#ef4444" : colorForSource(spanSource(span));

    const left = ((toMs(span.startTime) - startMs) / rangeMs) * 100;
    const rawWidth =
      (Math.max(0, toMs(span.endTime) - toMs(span.startTime)) / rangeMs) * 100;
    const cappedWidth = Math.min(rawWidth, (maxDisplayMs / rangeMs) * 100);
    const width = cappedWidth > 0 ? cappedWidth : rawWidth;
    return `left:${left}%;width:${Math.max(0.5, width)}%;background:${color}`;
  }

  function spanSource(span: any) {
    return span?.source || span?.service || "origine sconosciuta";
  }

  function colorForSource(value: string) {
    let hash = 0;
    for (let i = 0; i < value.length; i += 1) {
      hash = (hash << 5) - hash + value.charCodeAt(i);
      hash |= 0;
    }
    return sourcePalette[Math.abs(hash) % sourcePalette.length];
  }

  function shortId(id?: string) {
    if (!id) return "-";
    return id.length > 10 ? `${id.slice(0, 6)}...${id.slice(-4)}` : id;
  }

  function isDbSpan(span: any): boolean {
    const name = span?.name?.toLowerCase() ?? "";
    if (span?.attributes) {
      const attrs = span.attributes;
      if (attrs["db.system"] || attrs["db.name"] || attrs["db.type"])
        return true;
    }
    return false;
  }

  interface SpanKindResult {
    label: string;
    short: string;
    color: string;
    icon: string;
  }

  const knownTechnologies = new Set([
    "postgres",
    "postgresql",
    "mysql",
    "mariadb",
    "redis",
    "mongodb",
    "mongo",
    "elasticsearch",
    "elastic",
    "cassandra",
    "clickhouse",
    "sqlite",
    "oracle",
    "sqlserver",
    "mssql",
    "dynamodb",
    "cosmosdb",
    "neo4j",
    "cockroachdb",
    "cockroach",
  ]);

  function getDbInfo(
    span: any,
  ): { system: string; name: string; operation: string } | null {
    const attrs = span.attributes || {};

    // Get db.system from attributes
    let system = attrs["db.system"] || attrs["db.type"] || "";
    let name = attrs["db.name"] || "";
    let operation =
      attrs["db.operation"] || attrs["db.statement"]?.split(" ")[0] || "";

    // Fallback: Check if span service indicates the system
    if (!system && span.service) {
      const svc = span.service.toLowerCase();
      for (const tech of knownTechnologies) {
        if (svc.includes(tech)) {
          system = tech;
          break;
        }
      }
    }

    // Fallback: Check if span name indicates the system ONLY if it is a known technology
    if (!system && span.name) {
      const spanNameLower = span.name.toLowerCase();
      if (knownTechnologies.has(spanNameLower)) {
        system = span.name;
      }
    }

    // Fallback logic for operation if missing
    if (!operation && span.name) {
      // Common SQL/DB verbs
      const firstWord = span.name.split(" ")[0].toUpperCase();
      if (
        [
          "SELECT",
          "INSERT",
          "UPDATE",
          "DELETE",
          "GET",
          "SET",
          "FIND",
          "QUERY",
          "COMMIT",
          "ROLLBACK",
        ].includes(firstWord)
      ) {
        operation = firstWord;
      }
    }

    // If we only have name/system but it's identical to span.name, it's not adding info.
    // But we return what we found, formatting handles the display.
    if (!system && !name && !operation) return null;
    return { system, name, operation };
  }

  function formatDbSystem(system: string): string {
    const systemMap: Record<string, string> = {
      postgresql: "PostgreSQL",
      postgres: "PostgreSQL",
      mysql: "MySQL",
      mariadb: "MariaDB",
      redis: "Redis",
      mongodb: "MongoDB",
      elasticsearch: "Elastic",
      cassandra: "Cassandra",
      clickhouse: "ClickHouse",
      sqlite: "SQLite",
      oracle: "Oracle",
      sqlserver: "SQL Server",
      dynamodb: "DynamoDB",
      cosmosdb: "CosmosDB",
      neo4j: "Neo4j",
      cockroachdb: "CockroachDB",
    };
    return systemMap[system.toLowerCase()] || system;
  }

  function spanKindInfo(span: any): SpanKindResult | null {
    // Check for DB span first
    if (isDbSpan(span)) {
      const dbInfo = getDbInfo(span);
      let shortLabel = "DB";
      let fullLabel = "Database Query";

      if (dbInfo) {
        const parts: string[] = [];
        let systemLabel = "";

        if (dbInfo.system) {
          systemLabel = formatDbSystem(dbInfo.system);
          parts.push(systemLabel);
        }
        if (dbInfo.name && dbInfo.name !== dbInfo.system) {
          parts.push(dbInfo.name);
        }
        if (dbInfo.operation) {
          parts.push(dbInfo.operation.toUpperCase());
        }

        if (parts.length > 0) {
          fullLabel = parts.join(" • ");
        }

        // Logic for short label (Badge)
        if (dbInfo.operation) {
          // Priority to operation: "Postgres SEL" or just "SELECT" if system is generic/unknown
          if (
            systemLabel &&
            knownTechnologies.has(dbInfo.system.toLowerCase())
          ) {
            const opShort = dbInfo.operation.slice(0, 3).toUpperCase();
            shortLabel = `${systemLabel} ${opShort}`;
          } else {
            shortLabel = dbInfo.operation.toUpperCase();
          }
        } else if (
          systemLabel &&
          knownTechnologies.has(dbInfo.system.toLowerCase())
        ) {
          // Show system only if it is a known technology (e.g. Postgres)
          shortLabel = systemLabel;
        } else {
          // Fallback to "DB" if we don't have operation and system is not a known tech
          // (avoids showing "Delivery" if that's just the span name)
          shortLabel = "DB";
        }
      }

      return {
        label: fullLabel,
        short: shortLabel,
        color: "#0891b2",
        icon: "🗄️",
      };
    }

    const raw = span?.spanKind ?? span?.kind;
    if (raw === null || raw === undefined || raw === "") {
      return null;
    }

    // Color mapping for ActivityKind:
    // Internal (0) = gray, Server (1) = green, Client (2) = blue, Producer (3) = purple, Consumer (4) = orange
    if (typeof raw === "number") {
      const map: Record<number, SpanKindResult> = {
        0: { label: "Internal", short: "I", color: "#64748b", icon: "⚙️" },
        1: { label: "Server", short: "S", color: "#16a34a", icon: "🌐" },
        2: { label: "Client", short: "CL", color: "#2563eb", icon: "📤" },
        3: { label: "Producer", short: "P", color: "#9333ea", icon: "📨" },
        4: { label: "Consumer", short: "C", color: "#ea580c", icon: "📩" },
      };
      return (
        map[raw] ?? {
          label: `Kind ${raw}`,
          short: "K",
          color: "#94a3b8",
          icon: "❓",
        }
      );
    }

    const normalized = String(raw).toUpperCase();
    if (normalized.includes("PRODUCER"))
      return { label: "Producer", short: "P", color: "#9333ea", icon: "📨" };
    if (normalized.includes("CONSUMER"))
      return { label: "Consumer", short: "C", color: "#ea580c", icon: "📩" };
    if (normalized.includes("SERVER"))
      return { label: "Server", short: "S", color: "#16a34a", icon: "🌐" };
    if (normalized.includes("CLIENT"))
      return { label: "Client", short: "CL", color: "#2563eb", icon: "📤" };
    if (normalized.includes("INTERNAL"))
      return { label: "Internal", short: "I", color: "#64748b", icon: "⚙️" };
    return {
      label: normalized,
      short: normalized.slice(0, 2),
      color: "#94a3b8",
      icon: "❓",
    };
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
    <div class="span-list">
      <div class="span-header">
        <span>Span</span>
        <span>Linea temporale</span>
        <span>Durata</span>
      </div>
      <div
        class="span-grid"
        role="presentation"
        on:mousemove={handleContainerMouseMove}
        on:mouseleave={handleMouseLeave}
      >
        {#each spans as span, i}
          {@const source = spanSource(span)}
          {@const kind = spanKindInfo(span)}
          {@const color = colorForSource(source)}
          <div class="span-row">
            <div class="meta">
              <div class="meta-title">
                <span class="source-dot" style={`background:${color}`}></span>
                <span class="name">{span.name || "Span"}</span>
                {#if kind}
                  <span
                    class="kind-badge"
                    title={kind.label}
                    style={`background:${kind.color}20;color:${kind.color};border-color:${kind.color}40`}
                    >{kind.icon} {kind.short}</span
                  >
                {/if}
              </div>
              <span class="service"
                >{span.service || "servizio sconosciuto"}</span
              >
            </div>
            <div class="bar-track">
              <div
                class="bar"
                style={barStyle(span)}
                role="tooltip"
                aria-label={formatDuration(durationMs(span))}
                data-index={i}
              ></div>
              <span class="bar-label">{formatDuration(offsetMs(span))}</span>
            </div>
            <div class="duration">{formatDuration(durationMs(span))}</div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  {#if hoveredSpan}
    <div class="span-tooltip" style="top: {tooltipY}px; left: {tooltipX}px;">
      <div class="tooltip-header">
        <span class="tooltip-title"
          >{hoveredSpan.name || "Span senza nome"}</span
        >
        <span class="tooltip-service"
          >{hoveredSpan.service || "Servizio sconosciuto"}</span
        >
      </div>

      <div class="tooltip-row">
        <span class="label">Source:</span>
        <span class="value">{spanSource(hoveredSpan)}</span>
      </div>

      <div class="tooltip-row">
        <span class="label">Status:</span>
        <span
          class="value"
          class:error={hoveredSpan.status === "ERROR" ||
            hoveredSpan.status === "STATUS_CODE_ERROR" ||
            hoveredSpan.status === "2"}
        >
          {hoveredSpan.status || "UNSET"}
        </span>
      </div>

      <div class="tooltip-metrics">
        <div class="metric">
          <span class="label">Start</span>
          <span class="value">{formatDuration(offsetMs(hoveredSpan))}</span>
        </div>
        <div class="metric">
          <span class="label">Duration</span>
          <span class="value">{formatDuration(durationMs(hoveredSpan))}</span>
        </div>
      </div>

      {#if hoveredSpan.attributes && Object.keys(hoveredSpan.attributes).length > 0}
        <div class="tooltip-section">
          <span class="section-title">Attributes</span>
          <div class="attributes-list">
            {#each Object.entries(hoveredSpan.attributes) as [key, value]}
              <div class="attr-row">
                <span class="attr-key">{key}:</span>
                <span class="attr-value">{value}</span>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  {/if}
</section>

<style>
  .span-tooltip {
    position: fixed;
    z-index: 1000;
    background: #0f172a;
    color: white;
    padding: 12px;
    border-radius: 8px;
    font-size: 12px;
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.3);
    pointer-events: none;
    max-width: 300px;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .tooltip-header {
    display: flex;
    flex-direction: column;
    margin-bottom: 8px;
    padding-bottom: 8px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .tooltip-title {
    font-weight: 700;
    font-size: 13px;
    margin-bottom: 2px;
    color: #f1f5f9;
  }

  .tooltip-service {
    font-size: 11px;
    color: #94a3b8;
  }

  .tooltip-row {
    display: flex;
    justify-content: space-between;
    margin-bottom: 4px;
    gap: 12px;
  }

  .label {
    color: #94a3b8;
  }

  .value {
    color: #f1f5f9;
    font-weight: 500;
  }

  .value.error {
    color: #ef4444;
    font-weight: 700;
  }

  .tooltip-metrics {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
    margin: 8px 0;
    padding: 8px 0;
    border-top: 1px solid rgba(255, 255, 255, 0.1);
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .metric {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .tooltip-section {
    margin-top: 8px;
  }

  .section-title {
    display: block;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    color: #94a3b8;
    margin-bottom: 4px;
  }

  .attributes-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 200px;
    overflow-y: hidden; /* Hide overflow to prevent too long lists */
  }

  .attr-row {
    display: flex;
    gap: 6px;
    font-family: monospace;
    font-size: 11px;
    line-height: 1.3;
    word-break: break-all;
  }

  .attr-key {
    color: #60a5fa;
    flex-shrink: 0;
  }

  .attr-value {
    color: #e2e8f0;
  }

  .timeline {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-bottom: 24px;
    height: 100%;
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

  .span-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding-right: 6px;
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 12px;
    background: #f8fafc;
    padding: 12px;
  }

  .span-header {
    display: grid;
    grid-template-columns: 220px 1fr 80px;
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
    display: grid;
    grid-template-columns: 220px 1fr 80px;
    gap: 10px 12px;
    padding-top: 10px;
  }

  .span-row {
    display: contents;
  }

  .meta {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .meta-title {
    display: flex;
    align-items: center;
    gap: 6px;
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
    gap: 2px;
    min-width: 18px;
    padding: 2px 6px;
    border-radius: 999px;
    border: 1px solid;
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    white-space: nowrap;
  }

  .service {
    font-size: 11px;
    color: #64748b;
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

  @media (max-width: 720px) {
    .span-header {
      display: none;
    }

    .span-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
