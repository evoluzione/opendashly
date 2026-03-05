<script lang="ts">
  import type { ErrorRatePoint } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';

  export let data: ErrorRatePoint[] = [];
  export let budgetPercent = 0.1; // 99.9% availability budget

  function ts(v: unknown): number | null {
    const n = Date.parse(String(v));
    return Number.isFinite(n) ? n : null;
  }

  const avg = (values: number[]): number => {
    if (values.length === 0) return 0;
    return values.reduce((a, b) => a + b, 0) / values.length;
  };

  function avgWindow(hours: number): number {
    if (data.length === 0) return 0;
    const parsed = data
      .map((p) => ({ t: ts(p.timestamp), rate: Number(p.errorRate) }))
      .filter((p) => p.t !== null && Number.isFinite(p.rate)) as { t: number; rate: number }[];
    if (parsed.length === 0) return 0;

    const maxTs = parsed[parsed.length - 1].t;
    const minTs = maxTs - hours * 3600 * 1000;
    const inWindow = parsed.filter((p) => p.t >= minTs).map((p) => p.rate);
    if (inWindow.length > 0) return avg(inWindow);

    const fallbackSize = hours <= 1 ? 12 : 72;
    return avg(parsed.slice(-fallbackSize).map((p) => p.rate));
  }

  $: rate1h = avgWindow(1);
  $: rate6h = avgWindow(6);
  $: burn1h = budgetPercent > 0 ? rate1h / budgetPercent : 0;
  $: burn6h = budgetPercent > 0 ? rate6h / budgetPercent : 0;

  function tone(burn: number): 'ok' | 'warn' | 'bad' {
    if (burn <= 1) return 'ok';
    if (burn <= 2) return 'warn';
    return 'bad';
  }
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      Error Budget Burn Rate
      <InfoTooltip text="Consumo del budget errori su finestre brevi/lunghe. >1x significa che stai bruciando budget troppo velocemente." />
    </span>
    <span class="table-subtitle">Budget: {budgetPercent.toFixed(2)}% errori</span>
  </div>

  <div class="burn-grid">
    <div class={`burn-card ${tone(burn1h)}`}>
      <span class="label">Burn 1h</span>
      <span class="value">{burn1h.toFixed(2)}x</span>
      <span class="rate">Err avg: {rate1h.toFixed(2)}%</span>
    </div>

    <div class={`burn-card ${tone(burn6h)}`}>
      <span class="label">Burn 6h</span>
      <span class="value">{burn6h.toFixed(2)}x</span>
      <span class="rate">Err avg: {rate6h.toFixed(2)}%</span>
    </div>
  </div>
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

  .table-header { margin-bottom: 14px; }
  .table-title { display: block; font-size: 14px; font-weight: 600; color: #0f172a; }
  .table-subtitle { display: block; font-size: 12px; color: #94a3b8; margin-top: 2px; }

  .burn-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .burn-card {
    border-radius: 12px;
    border: 1px solid #e2e8f0;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .burn-card.ok { background: #f0fdf4; border-color: #bbf7d0; }
  .burn-card.warn { background: #fffbeb; border-color: #fde68a; }
  .burn-card.bad { background: #fef2f2; border-color: #fecaca; }

  .label { font-size: 11px; text-transform: uppercase; letter-spacing: 0.06em; color: #64748b; }
  .value { font-size: 28px; line-height: 1; font-weight: 800; color: #0f172a; font-variant-numeric: tabular-nums; }
  .rate { font-size: 12px; color: #475569; }

  @media (max-width: 700px) {
    .burn-grid { grid-template-columns: 1fr; }
  }
</style>
