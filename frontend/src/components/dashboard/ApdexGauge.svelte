<script lang="ts">
  import type { ApdexScore } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';
  import { locale, t } from '../../lib/i18n';

  export let data: ApdexScore | null = null;

  function getColor(score: number): string {
    if (score >= 0.94) return '#22c55e'; // Excellent - green
    if (score >= 0.85) return '#84cc16'; // Good - lime
    if (score >= 0.70) return '#eab308'; // Fair - yellow
    if (score >= 0.50) return '#f97316'; // Poor - orange
    return '#ef4444'; // Unacceptable - red
  }

  function getLabel(score: number): string {
    if (score >= 0.94) return t($locale, 'dashboard.apdex.excellent');
    if (score >= 0.85) return t($locale, 'dashboard.apdex.good');
    if (score >= 0.70) return t($locale, 'dashboard.apdex.fair');
    if (score >= 0.50) return t($locale, 'dashboard.apdex.poor');
    return t($locale, 'dashboard.apdex.unacceptable');
  }

  $: score = data?.score ?? 0;
  $: color = getColor(score);
  $: label = getLabel(score);
  $: percentage = score * 100;
  $: circumference = 2 * Math.PI * 45;
  $: dashOffset = circumference * (1 - score);
</script>

<div class="gauge-card">
  <div class="gauge-header">
    <span class="gauge-title">
      APDEX
      <InfoTooltip text="Application Performance Index: misura la soddisfazione degli utenti. Valori tra 0 e 1, dove 1 = tutti soddisfatti. Soglia T: tempo di risposta accettabile." position="bottom" align="left" />
    </span>
    <span class="gauge-subtitle">{t($locale, 'dashboard.apdex.threshold', { threshold: data?.threshold ?? 2000 })}</span>
  </div>

  <div class="gauge-container">
    <svg viewBox="0 0 100 100" class="gauge-svg">
      <circle
        cx="50"
        cy="50"
        r="45"
        fill="none"
        stroke="#e2e8f0"
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
      <span class="score" style="color: {color}">{score.toFixed(2)}</span>
      <span class="label" style="color: {color}">{label}</span>
    </div>
  </div>

  <div class="breakdown">
    <div class="breakdown-row">
      <span class="dot satisfied"></span>
      <span class="breakdown-label">{t($locale, 'dashboard.apdex.satisfied')}</span>
      <span class="breakdown-value">{data?.satisfied ?? 0}</span>
    </div>
    <div class="breakdown-row">
      <span class="dot tolerating"></span>
      <span class="breakdown-label">{t($locale, 'dashboard.apdex.tolerating')}</span>
      <span class="breakdown-value">{data?.tolerating ?? 0}</span>
    </div>
    <div class="breakdown-row">
      <span class="dot frustrated"></span>
      <span class="breakdown-label">{t($locale, 'dashboard.apdex.frustrated')}</span>
      <span class="breakdown-value">{data?.frustrated ?? 0}</span>
    </div>
  </div>
</div>

<style>
  .gauge-card {
    background: white;
    border-radius: 16px;
    padding: 20px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(15, 23, 42, 0.06);
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
    color: #64748b;
  }

  .gauge-subtitle {
    font-size: 11px;
    color: #94a3b8;
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
    border-top: 1px solid #f1f5f9;
  }

  .breakdown-row {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: clamp(11px, 3.2cqw, 12px);
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .dot.satisfied {
    background: #22c55e;
  }

  .dot.tolerating {
    background: #eab308;
  }

  .dot.frustrated {
    background: #ef4444;
  }

  .breakdown-label {
    color: #64748b;
    flex: 1;
  }

  .breakdown-value {
    color: #0f172a;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }
</style>
