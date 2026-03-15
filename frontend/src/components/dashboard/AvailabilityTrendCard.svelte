<script lang="ts">
  import type { ErrorRatePoint } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';
  import { locale, t } from '../../lib/i18n';

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
      {t($locale, 'dashboard.availability.title')}
      <InfoTooltip text={t($locale, 'dashboard.availability.tooltip')} />
    </span>
    <span class="table-subtitle">{t($locale, 'dashboard.availability.subtitle')}</span>
  </div>

  {#if points.length === 0}
    <div class="empty">{t($locale, 'dashboard.noData')}</div>
  {:else}
    <div class="kpis">
      <div><strong>{current.toFixed(2)}%</strong><span>{t($locale, 'dashboard.availability.current')}</span></div>
      <div><strong>{avgVal.toFixed(2)}%</strong><span>{t($locale, 'dashboard.availability.average')}</span></div>
      <div><strong>{minVal.toFixed(2)}%</strong><span>{t($locale, 'dashboard.availability.min')}</span></div>
    </div>

    <div class="trend">
      <svg viewBox="0 0 100 36" preserveAspectRatio="none" aria-label={t($locale, 'dashboard.availability.ariaTrend')}>
        <polyline points={path} fill="none" stroke="var(--color-success-500)" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"></polyline>
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
    border: 1px solid rgba(var(--rgb-slate-950), 0.06);
    min-width: 0;
  }

  .table-header { margin-bottom: 12px; }
  .table-title { display: block; font-size: 14px; font-weight: 600; color: var(--color-slate-950); }
  .table-subtitle { display: block; font-size: 12px; color: var(--color-slate-400); margin-top: 2px; }
  .empty { text-align: center; color: var(--color-slate-400); padding: 40px 0; font-size: 14px; }

  .kpis {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 10px;
    margin-bottom: 12px;
  }

  .kpis div {
    background: var(--color-slate-50);
    border: 1px solid var(--color-slate-200);
    border-radius: 10px;
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .kpis strong {
    font-size: 17px;
    color: var(--color-slate-950);
    font-variant-numeric: tabular-nums;
    line-height: 1;
  }

  .kpis span {
    font-size: 10px;
    color: var(--color-slate-500);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .trend {
    height: 96px;
    border-radius: 10px;
    border: 1px solid #dcfce7;
    background: linear-gradient(180deg, var(--color-success-50) 0%, var(--color-white) 100%);
    padding: 10px;
  }

  .trend svg {
    width: 100%;
    height: 100%;
    display: block;
  }
</style>
