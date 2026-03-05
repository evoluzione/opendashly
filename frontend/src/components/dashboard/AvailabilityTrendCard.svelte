<script lang="ts">
  import type { ErrorRatePoint } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';

  export let data: ErrorRatePoint[] = [];

  $: points = data
    .map((p) => {
      const errRate = Number(p.errorRate);
      if (!Number.isFinite(errRate)) return null;
      return Math.max(0, Math.min(100, 100 - errRate));
    })
    .filter((v): v is number => v !== null)
    .slice(-48);

  $: current = points.length > 0 ? points[points.length - 1] : 0;
  $: minVal = points.length > 0 ? Math.min(...points) : 0;
  $: avgVal = points.length > 0 ? points.reduce((a, b) => a + b, 0) / points.length : 0;

  $: path = (() => {
    if (points.length === 0) return '';
    const w = 100;
    const h = 36;
    const min = Math.max(0, Math.min(...points) - 1);
    const max = Math.min(100, Math.max(...points) + 1);
    const span = Math.max(1, max - min);

    return points
      .map((v, i) => {
        const x = points.length === 1 ? 0 : (i / (points.length - 1)) * w;
        const y = h - ((v - min) / span) * h;
        return `${x.toFixed(2)},${y.toFixed(2)}`;
      })
      .join(' ');
  })();
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      Disponibilita nel Tempo
      <InfoTooltip text="Trend di availability calcolata come 100% - error rate." />
    </span>
    <span class="table-subtitle">Ultime finestre temporali</span>
  </div>

  {#if points.length === 0}
    <div class="empty">Nessun dato disponibile</div>
  {:else}
    <div class="kpis">
      <div><strong>{current.toFixed(2)}%</strong><span>attuale</span></div>
      <div><strong>{avgVal.toFixed(2)}%</strong><span>media</span></div>
      <div><strong>{minVal.toFixed(2)}%</strong><span>min</span></div>
    </div>

    <div class="trend">
      <svg viewBox="0 0 100 36" preserveAspectRatio="none" aria-label="Availability trend">
        <polyline points={path} fill="none" stroke="#22c55e" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"></polyline>
      </svg>
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

  .kpis {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 10px;
    margin-bottom: 12px;
  }

  .kpis div {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 10px;
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .kpis strong {
    font-size: 17px;
    color: #0f172a;
    font-variant-numeric: tabular-nums;
    line-height: 1;
  }

  .kpis span {
    font-size: 10px;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .trend {
    height: 96px;
    border-radius: 10px;
    border: 1px solid #dcfce7;
    background: linear-gradient(180deg, #f0fdf4 0%, #ffffff 100%);
    padding: 10px;
  }

  .trend svg {
    width: 100%;
    height: 100%;
    display: block;
  }
</style>
