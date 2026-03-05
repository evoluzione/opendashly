<script lang="ts">
  import type { LatencyPercentilePoint } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';

  export let data: LatencyPercentilePoint[] = [];
  export let targetMs = 500;

  const toNumber = (v: unknown): number | null => {
    const n = typeof v === 'number' ? v : Number(v);
    return Number.isFinite(n) ? n : null;
  };

  $: samples = data
    .map((p) => ({ p95: toNumber(p.p95) }))
    .filter((p): p is { p95: number } => p.p95 !== null);

  $: total = samples.length;
  $: compliant = samples.filter((s) => s.p95 <= targetMs).length;
  $: compliancePct = total > 0 ? (compliant / total) * 100 : 0;
  $: latestP95 = total > 0 ? samples[total - 1].p95 : 0;
  $: tail = samples.slice(-20);

  function statusLabel(value: number): string {
    if (value >= 99) return 'Ottimo';
    if (value >= 95) return 'Buono';
    if (value >= 90) return 'Attenzione';
    return 'Critico';
  }

  function statusColor(value: number): string {
    if (value >= 99) return '#22c55e';
    if (value >= 95) return '#3b82f6';
    if (value >= 90) return '#f59e0b';
    return '#ef4444';
  }
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      SLO Compliance
      <InfoTooltip text="Percentuale finestre in cui il P95 e entro la soglia target." />
    </span>
    <span class="table-subtitle">Target P95 &lt;= {targetMs}ms</span>
  </div>

  {#if total === 0}
    <div class="empty">Nessun dato disponibile</div>
  {:else}
    <div class="slo-main">
      <div class="score" style={`color:${statusColor(compliancePct)}`}>{compliancePct.toFixed(1)}%</div>
      <div class="meta">
        <span>{statusLabel(compliancePct)}</span>
        <span>{compliant}/{total} finestre conformi</span>
        <span>P95 attuale: {latestP95.toFixed(0)}ms</span>
      </div>
    </div>

    <div class="sparkline">
      {#each tail as point}
        <span
          class="tick"
          style={`height:${Math.max(16, Math.min(100, (point.p95 / (targetMs * 2)) * 100))}%;background:${point.p95 <= targetMs ? '#22c55e' : '#ef4444'};`}
        ></span>
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
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  .table-header { margin-bottom: 12px; }
  .table-title { display: block; font-size: 14px; font-weight: 600; color: #0f172a; }
  .table-subtitle { display: block; font-size: 12px; color: #94a3b8; margin-top: 2px; }
  .empty { text-align: center; color: #94a3b8; padding: 40px 0; font-size: 14px; }

  .slo-main {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 8px 0 10px;
  }

  .score {
    font-size: 38px;
    font-weight: 800;
    line-height: 1;
    font-variant-numeric: tabular-nums;
  }

  .meta {
    display: flex;
    flex-direction: column;
    gap: 4px;
    color: #475569;
    font-size: 12px;
    text-align: right;
  }

  .sparkline {
    margin-top: 6px;
    min-height: 84px;
    display: grid;
    grid-template-columns: repeat(20, minmax(0, 1fr));
    align-items: end;
    gap: 4px;
    padding-top: 6px;
    border-top: 1px solid #f1f5f9;
  }

  .tick {
    display: block;
    width: 100%;
    border-radius: 4px 4px 0 0;
  }
</style>
