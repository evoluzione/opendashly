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
      error = err instanceof Error ? err.message : 'Failed to load spans';
    } finally {
      loading = false;
    }
  });

  function toMs(value: string | Date) {
    return new Date(value).getTime();
  }

  const maxDisplayMs = 60000;

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
    const left = ((toMs(span.startTime) - startMs) / rangeMs) * 100;
    const rawWidth = (Math.max(0, toMs(span.endTime) - toMs(span.startTime)) / rangeMs) * 100;
    const cappedWidth = Math.min(rawWidth, (maxDisplayMs / rangeMs) * 100);
    const width = cappedWidth > 0 ? cappedWidth : rawWidth;
    return `left:${left}%;width:${Math.max(0.5, width)}%`;
  }
</script>

<section class="timeline">
  <header>
    <h4>Span timeline</h4>
    <p>Durata totale: {Math.round(rangeMs)} ms</p>
  </header>

  {#if loading}
    <p class="status">Caricamento spans...</p>
  {:else if error}
    <p class="status error">{error}</p>
  {:else if spans.length === 0}
    <p class="status">Nessuno span disponibile.</p>
  {:else}
    <div class="span-list">
      <div class="span-header">
        <span>Span</span>
        <span>Timeline</span>
        <span>Durata</span>
      </div>
      <div class="span-grid">
        {#each spans as span}
          <div class="span-row">
            <div class="meta">
              <span class="name">{span.name || 'Span'}</span>
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

  .name {
    font-size: 12px;
    font-weight: 600;
    color: #0f172a;
    overflow: hidden;
    text-overflow: ellipsis;
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
    background: linear-gradient(90deg, #2563eb, #38bdf8);
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
