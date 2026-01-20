<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchTraceSpans } from '../services/traces';

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
      error = err instanceof Error ? err.message : 'Impossibile caricare gli span';
    } finally {
      loading = false;
    }
  });

  function toMs(value: string | Date) {
    return new Date(value).getTime();
  }

  const maxDisplayMs = 60000;
  const sourcePalette = [
    '#0ea5e9',
    '#22c55e',
    '#f97316',
    '#ef4444',
    '#8b5cf6',
    '#14b8a6',
    '#eab308',
    '#6366f1'
  ];

  function durationMs(span: any) {
    return Math.max(0, Math.round(span.duration / 1_000_000));
  }

  function formatDuration(ms: number) {
    if (ms > maxDisplayMs) return '> 60 s';
    if (ms < 1000) return `${ms} ms`;
    return `${(ms / 1000).toFixed(2)} s`;
  }

  function offsetMs(span: any) {
    return Math.max(0, Math.round((toMs(span.startTime) - startMs)));
  }

  $: rangeSpans =
    spans.filter((span) => durationMs(span) > 0 && durationMs(span) <= maxDisplayMs).length > 0
      ? spans.filter((span) => durationMs(span) > 0 && durationMs(span) <= maxDisplayMs)
      : spans;

  $: startMs = rangeSpans.length ? Math.min(...rangeSpans.map((span) => toMs(span.startTime))) : 0;
  $: endMs = rangeSpans.length ? Math.max(...rangeSpans.map((span) => toMs(span.endTime))) : 0;
  $: rangeMs = Math.max(1, endMs - startMs);

  function barStyle(span: any) {
    const color = colorForSource(spanSource(span));
    const left = ((toMs(span.startTime) - startMs) / rangeMs) * 100;
    const rawWidth = (Math.max(0, toMs(span.endTime) - toMs(span.startTime)) / rangeMs) * 100;
    const cappedWidth = Math.min(rawWidth, (maxDisplayMs / rangeMs) * 100);
    const width = cappedWidth > 0 ? cappedWidth : rawWidth;
    return `left:${left}%;width:${Math.max(0.5, width)}%;background:${color}`;
  }

  function spanSource(span: any) {
    return span?.source || span?.service || 'origine sconosciuta';
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
    if (!id) return '-';
    return id.length > 10 ? `${id.slice(0, 6)}...${id.slice(-4)}` : id;
  }

  function spanKindInfo(span: any) {
    const raw = span?.spanKind ?? span?.kind;
    if (raw === null || raw === undefined || raw === '') {
      return null;
    }
    if (typeof raw === 'number') {
      const map: Record<number, { label: string; short: string }> = {
        1: { label: 'Internal', short: 'I' },
        2: { label: 'Server', short: 'S' },
        3: { label: 'Client', short: 'CL' },
        4: { label: 'Producer', short: 'P' },
        5: { label: 'Consumer', short: 'C' }
      };
      return map[raw] ?? { label: `Kind ${raw}`, short: 'K' };
    }
    const normalized = String(raw).toUpperCase();
    if (normalized.includes('PRODUCER')) return { label: 'Producer', short: 'P' };
    if (normalized.includes('CONSUMER')) return { label: 'Consumer', short: 'C' };
    if (normalized.includes('SERVER')) return { label: 'Server', short: 'S' };
    if (normalized.includes('CLIENT')) return { label: 'Client', short: 'CL' };
    if (normalized.includes('INTERNAL')) return { label: 'Internal', short: 'I' };
    return { label: normalized, short: normalized.slice(0, 2) };
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
      <div class="span-grid">
        {#each spans as span}
          {@const source = spanSource(span)}
          {@const kind = spanKindInfo(span)}
          {@const color = colorForSource(source)}
          <div class="span-row">
            <div class="meta">
              <div class="meta-title">
                <span class="source-dot" style={`background:${color}`}></span>
                <span class="name">{span.name || 'Span'}</span>
                {#if kind}
                  <span class="kind-badge" title={kind.label}>{kind.short}</span>
                {/if}
              </div>
              <span class="service">{span.service || 'servizio sconosciuto'}</span>
            </div>
            <div class="bar-track">
              <div class="bar" style={barStyle(span)} title={formatDuration(durationMs(span))}></div>
              <span class="bar-label">{formatDuration(offsetMs(span))}</span>
            </div>
            <div class="duration">{formatDuration(durationMs(span))}</div>
          </div>
        {/each}
      </div>
    </div>
  {/if}
</section>

<style>
  .timeline {
    display: flex;
    flex-direction: column;
    gap: 12px;
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
    max-height: 320px;
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
    min-width: 18px;
    padding: 2px 6px;
    border-radius: 999px;
    background: #e2e8f0;
    color: #0f172a;
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
  }

  .service {
    font-size: 11px;
    color: #64748b;
  }

  .details {
    font-size: 10px;
    color: #94a3b8;
    font-weight: 600;
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
