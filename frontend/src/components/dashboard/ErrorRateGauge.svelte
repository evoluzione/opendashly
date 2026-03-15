<script lang="ts">
  import InfoTooltip from '../common/InfoTooltip.svelte';
  import { locale, t } from '../../lib/i18n';

  export let errorRate: number = 0;
  export let totalErrors: number = 0;
  export let totalRequests: number = 0;

  function computeErrorRate(rate: number, errors: number, requests: number): number {
    if (requests > 0) {
      return (errors / requests) * 100;
    }
    return Number.isFinite(rate) ? rate : 0;
  }

  function formatRate(rate: number): string {
    if (rate === 0) return '0.0%';
    if (rate < 0.1) return `${rate.toFixed(3)}%`;
    if (rate < 1) return `${rate.toFixed(2)}%`;
    return `${rate.toFixed(1)}%`;
  }

  function getColor(rate: number): string {
    if (rate <= 1) return 'var(--color-success-500)'; // Good - green
    if (rate <= 5) return '#eab308'; // Warning - yellow
    if (rate <= 10) return '#f97316'; // Bad - orange
    return 'var(--color-danger-500)'; // Critical - red
  }

  function getLabel(rate: number): string {
    if (rate <= 1) return t($locale, 'dashboard.errorRate.excellent');
    if (rate <= 5) return t($locale, 'dashboard.errorRate.warning');
    if (rate <= 10) return t($locale, 'dashboard.errorRate.problem');
    return t($locale, 'dashboard.errorRate.critical');
  }

  $: effectiveRate = computeErrorRate(errorRate, totalErrors, totalRequests);
  $: color = getColor(effectiveRate);
  $: label = getLabel(effectiveRate);
  $: displayRate = Math.min(effectiveRate, 100);
  $: circumference = 2 * Math.PI * 45;
  $: dashOffset = circumference * (1 - displayRate / 100);
</script>

<div class="gauge-card">
  <div class="gauge-header">
    <span class="gauge-title">
      Error Rate
      <InfoTooltip text="Percentuale di richieste che hanno generato errori. Obiettivo ideale: < 1%. Valori superiori al 5% richiedono attenzione." position="bottom" />
    </span>
  </div>

  <div class="gauge-container">
    <svg viewBox="0 0 100 100" class="gauge-svg">
      <circle
        cx="50"
        cy="50"
        r="45"
        fill="none"
        stroke="var(--color-slate-200)"
        stroke-width="8"
      />
      <circle
        cx="50"
        cy="50"
        r="45"
        fill="none"
        stroke={color}
        stroke-width="8"
        stroke-linecap="round"
        stroke-dasharray={circumference}
        stroke-dashoffset={dashOffset}
        transform="rotate(-90 50 50)"
        class="gauge-progress"
      />
    </svg>
    <div class="gauge-value">
      <span class="score" style="color: {color}">{formatRate(effectiveRate)}</span>
      <span class="label" style="color: {color}">{label}</span>
    </div>
  </div>

  <div class="breakdown">
    <div class="breakdown-row">
      <span class="breakdown-label">{t($locale, 'dashboard.errorRate.errors')}</span>
      <span class="breakdown-value error">{totalErrors.toLocaleString()}</span>
    </div>
    <div class="breakdown-row">
      <span class="breakdown-label">{t($locale, 'dashboard.errorRate.totalRequests')}</span>
      <span class="breakdown-value">{totalRequests.toLocaleString()}</span>
    </div>
  </div>
</div>

<style>
  .gauge-card {
    background: white;
    border-radius: 16px;
    padding: 20px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(var(--rgb-slate-950), 0.06);
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-height: 0;
    container-type: inline-size;
  }

  .gauge-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .gauge-title {
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--color-slate-500);
  }

  .gauge-container {
    position: relative;
    width: clamp(92px, 38cqw, 140px);
    aspect-ratio: 1 / 1;
    margin: 0 auto;
  }

  .gauge-svg {
    width: 100%;
    height: 100%;
  }

  .gauge-progress {
    transition: stroke-dashoffset 0.5s ease-out;
  }

  .gauge-value {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    text-align: center;
  }

  .gauge-value .score {
    display: block;
    font-size: clamp(18px, 7cqw, 24px);
    font-weight: 700;
    line-height: 1.2;
  }

  .gauge-value .label {
    display: block;
    font-size: clamp(9px, 3cqw, 10px);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .breakdown {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-top: 12px;
    border-top: 1px solid var(--color-slate-100);
  }

  .breakdown-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: clamp(11px, 3.2cqw, 12px);
  }

  .breakdown-label {
    color: var(--color-slate-500);
  }

  .breakdown-value {
    color: var(--color-slate-950);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }

  .breakdown-value.error {
    color: var(--color-danger-500);
  }
</style>
