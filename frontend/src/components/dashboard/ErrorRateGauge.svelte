<script lang="ts">
  export let errorRate: number = 0;
  export let totalErrors: number = 0;
  export let totalRequests: number = 0;

  function getColor(rate: number): string {
    if (rate <= 1) return '#22c55e'; // Good - green
    if (rate <= 5) return '#eab308'; // Warning - yellow
    if (rate <= 10) return '#f97316'; // Bad - orange
    return '#ef4444'; // Critical - red
  }

  function getLabel(rate: number): string {
    if (rate <= 1) return 'Ottimo';
    if (rate <= 5) return 'Attenzione';
    if (rate <= 10) return 'Problema';
    return 'Critico';
  }

  $: color = getColor(errorRate);
  $: label = getLabel(errorRate);
  $: displayRate = Math.min(errorRate, 100);
  $: circumference = 2 * Math.PI * 45;
  $: dashOffset = circumference * (1 - displayRate / 100);
</script>

<div class="gauge-card">
  <div class="gauge-header">
    <span class="gauge-title">Error Rate</span>
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
      <span class="score" style="color: {color}">{errorRate.toFixed(1)}%</span>
      <span class="label" style="color: {color}">{label}</span>
    </div>
  </div>

  <div class="breakdown">
    <div class="breakdown-row">
      <span class="breakdown-label">Errori</span>
      <span class="breakdown-value error">{totalErrors.toLocaleString()}</span>
    </div>
    <div class="breakdown-row">
      <span class="breakdown-label">Totale richieste</span>
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
    border: 1px solid rgba(15, 23, 42, 0.06);
    display: flex;
    flex-direction: column;
    gap: 16px;
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

  .gauge-container {
    position: relative;
    width: 120px;
    height: 120px;
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
    font-size: 24px;
    font-weight: 700;
    line-height: 1.2;
  }

  .gauge-value .label {
    display: block;
    font-size: 10px;
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
    justify-content: space-between;
    font-size: 12px;
  }

  .breakdown-label {
    color: #64748b;
  }

  .breakdown-value {
    color: #0f172a;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }

  .breakdown-value.error {
    color: #ef4444;
  }
</style>
