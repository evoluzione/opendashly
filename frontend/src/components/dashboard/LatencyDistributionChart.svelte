<script lang="ts">
  import type { LatencyBucket } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';

  export let data: LatencyBucket[] = [];

  $: maxPercentage = Math.max(...data.map(d => d.percentage), 1);

  function getBarColor(index: number, total: number): string {
    const colors = [
      '#22c55e', // 0-100ms - green
      '#84cc16', // 100-250ms - lime
      '#a3e635', // 250-500ms - light lime
      '#eab308', // 500ms-1s - yellow
      '#f97316', // 1-2s - orange
      '#fb923c', // 2-5s - light orange
      '#ef4444', // 5-10s - red
      '#dc2626'  // >10s - dark red
    ];
    return colors[Math.min(index, colors.length - 1)];
  }
</script>

<div class="chart-card">
  <div class="chart-header">
    <span class="chart-title">
      Distribuzione Latenza
      <InfoTooltip text="Distribuzione dei tempi di risposta. Mostra quante richieste rientrano in ogni intervallo di latenza. I colori vanno dal verde (veloce) al rosso (lento)." />
    </span>
    <span class="chart-subtitle">Istogramma delle latenze</span>
  </div>

  {#if data.length === 0}
    <div class="empty">Nessun dato disponibile</div>
  {:else}
    <div class="histogram">
      {#each data as bucket, i}
        <div class="bar-container">
          <div class="bar-wrapper">
            <div
              class="bar"
              style="height: {(bucket.percentage / maxPercentage) * 100}%; background: {getBarColor(i, data.length)}"
            >
              <span class="bar-value">{bucket.percentage.toFixed(1)}%</span>
            </div>
          </div>
          <span class="bar-label">{bucket.label}</span>
          <span class="bar-count">{bucket.count.toLocaleString()}</span>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .chart-card {
    background: white;
    border-radius: 16px;
    padding: 20px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(15, 23, 42, 0.06);
  }

  .chart-header {
    margin-bottom: 20px;
  }

  .chart-title {
    display: block;
    font-size: 14px;
    font-weight: 600;
    color: #0f172a;
  }

  .chart-subtitle {
    display: block;
    font-size: 12px;
    color: #94a3b8;
    margin-top: 2px;
  }

  .empty {
    text-align: center;
    color: #94a3b8;
    padding: 40px 0;
    font-size: 14px;
  }

  .histogram {
    display: flex;
    align-items: flex-end;
    gap: 8px;
    height: 200px;
    padding-top: 20px;
  }

  .bar-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    height: 100%;
  }

  .bar-wrapper {
    flex: 1;
    width: 100%;
    display: flex;
    align-items: flex-end;
    justify-content: center;
  }

  .bar {
    width: 100%;
    max-width: 48px;
    min-height: 4px;
    border-radius: 4px 4px 0 0;
    transition: height 0.3s ease-out;
    position: relative;
  }

  .bar-value {
    position: absolute;
    top: -20px;
    left: 50%;
    transform: translateX(-50%);
    font-size: 10px;
    font-weight: 600;
    color: #64748b;
    white-space: nowrap;
  }

  .bar-label {
    font-size: 10px;
    color: #64748b;
    margin-top: 8px;
    text-align: center;
    white-space: nowrap;
  }

  .bar-count {
    font-size: 10px;
    font-weight: 600;
    color: #0f172a;
    margin-top: 2px;
    font-variant-numeric: tabular-nums;
  }

  @media (max-width: 640px) {
    .histogram {
      gap: 4px;
    }

    .bar-label {
      font-size: 8px;
    }

    .bar-value {
      display: none;
    }
  }
</style>
