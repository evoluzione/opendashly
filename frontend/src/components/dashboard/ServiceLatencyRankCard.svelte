<script lang="ts">
  import type { EndpointLatency } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';

  export let data: EndpointLatency[] = [];

  type Row = { service: string; p95: number; count: number };

  $: ranked = (() => {
    const map = new Map<string, { weightedP95: number; count: number }>();
    for (const row of data) {
      const weight = Number.isFinite(row.count) ? row.count : 0;
      if (weight <= 0) continue;
      const cur = map.get(row.service) ?? { weightedP95: 0, count: 0 };
      cur.weightedP95 += row.p95 * weight;
      cur.count += weight;
      map.set(row.service, cur);
    }

    const out: Row[] = [];
    for (const [service, agg] of map.entries()) {
      const p95 = agg.count > 0 ? agg.weightedP95 / agg.count : 0;
      out.push({ service, p95, count: agg.count });
    }

    return out.sort((a, b) => b.p95 - a.p95).slice(0, 8);
  })();

  $: maxP95 = Math.max(...ranked.map((r) => r.p95), 1);

  function fmtMs(ms: number): string {
    if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`;
    return `${ms.toFixed(0)}ms`;
  }
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      Top Servizi per Latenza P95
      <InfoTooltip text="Classifica servizi con latenza P95 peggiore. Utile per prioritizzare ottimizzazioni." />
    </span>
    <span class="table-subtitle">Ranking servizi</span>
  </div>

  {#if ranked.length === 0}
    <div class="empty">Nessun dato disponibile</div>
  {:else}
    <div class="rows">
      {#each ranked as row}
        <div class="row">
          <div class="service">{row.service}</div>
          <div class="bar-bg">
            <span class="bar" style={`width:${(row.p95 / maxP95) * 100}%`}></span>
          </div>
          <div class="value">{fmtMs(row.p95)}</div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .table-card {
    background: white;
    border-radius: 16px;
    padding: 20px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(15, 23, 42, 0.06);
    min-width: 0;
  }

  .table-header { margin-bottom: 12px; }
  .table-title { display: block; font-size: 14px; font-weight: 600; color: #0f172a; }
  .table-subtitle { display: block; font-size: 12px; color: #94a3b8; margin-top: 2px; }
  .empty { text-align: center; color: #94a3b8; padding: 40px 0; font-size: 14px; }

  .rows { display: flex; flex-direction: column; gap: 8px; }

  .row {
    display: grid;
    grid-template-columns: minmax(90px, 1fr) minmax(120px, 3fr) auto;
    gap: 10px;
    align-items: center;
  }

  .service {
    font-size: 12px;
    color: #334155;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .bar-bg {
    height: 8px;
    background: #e2e8f0;
    border-radius: 999px;
    overflow: hidden;
  }

  .bar {
    display: block;
    height: 100%;
    border-radius: 999px;
    background: linear-gradient(90deg, #f59e0b 0%, #ef4444 100%);
  }

  .value {
    font-size: 12px;
    font-weight: 700;
    color: #0f172a;
    font-variant-numeric: tabular-nums;
  }
</style>
