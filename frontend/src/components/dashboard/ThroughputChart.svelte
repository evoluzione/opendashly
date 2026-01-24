<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import uPlot from 'uplot';
  import 'uplot/dist/uPlot.min.css';
  import type { ThroughputPoint } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';

  export let data: ThroughputPoint[] = [];

  let containerEl: HTMLDivElement;
  let chartEl: HTMLDivElement;
  let chart: uPlot | null = null;
  let containerWidth = 0;
  let resizeObserver: ResizeObserver | null = null;

  function buildChartData(points: ThroughputPoint[]) {
    if (!points || points.length === 0) return null;
    const xValues = points.map((p) => new Date(p.timestamp).getTime() / 1000);
    const requestValues = points.map((p) => p.requestCount);
    const errorValues = points.map((p) => p.errorCount);
    return [xValues, requestValues, errorValues];
  }

  function renderChart(points: ThroughputPoint[], width: number) {
    if (!chartEl || !width || points.length === 0) return;
    if (chart) {
      chart.destroy();
      chart = null;
    }

    const chartData = buildChartData(points);
    if (!chartData) return;

    chart = new uPlot(
      {
        title: 'Throughput nel tempo',
        width,
        height: 220,
        series: [
          {},
          {
            label: 'Richieste',
            stroke: '#2563eb',
            width: 2,
            fill: 'rgba(37, 99, 235, 0.1)'
          },
          {
            label: 'Errori',
            stroke: '#ef4444',
            width: 2,
            fill: 'rgba(239, 68, 68, 0.1)'
          }
        ],
        scales: {
          x: { time: true },
          y: { min: 0 }
        },
        axes: [
          {},
          {
            label: 'Count',
            labelSize: 12,
            size: 50
          }
        ],
        legend: {
          show: true
        }
      },
      chartData,
      chartEl
    );
  }

  onMount(() => {
    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver((entries) => {
        for (const entry of entries) {
          containerWidth = Math.max(320, Math.floor(entry.contentRect.width));
        }
      });
      if (containerEl) {
        resizeObserver.observe(containerEl);
      }
      if (containerEl) {
        containerWidth = Math.max(320, Math.floor(containerEl.clientWidth));
      }
    } else if (containerEl) {
      containerWidth = Math.max(320, Math.floor(containerEl.clientWidth));
    }
  });

  onDestroy(() => {
    if (chart) {
      chart.destroy();
      chart = null;
    }
    if (resizeObserver && containerEl) {
      resizeObserver.unobserve(containerEl);
    }
  });

  $: if (chartEl && containerWidth > 0 && data.length > 0) {
    renderChart(data, Math.max(320, containerWidth - 40));
  }

  $: if (data.length === 0 && chart) {
    chart.destroy();
    chart = null;
  }
</script>

<div class="chart-card" bind:this={containerEl}>
  <div class="chart-header">
    <span class="chart-title">
      Throughput nel Tempo
      <InfoTooltip text="Andamento delle richieste e degli errori nel tempo. Utile per identificare picchi di carico e correlazioni tra traffico ed errori." />
    </span>
    <span class="chart-subtitle">Richieste ed errori nel tempo</span>
  </div>

  {#if data.length === 0}
    <div class="empty">Nessun dato disponibile</div>
  {:else}
    <div class="chart-container" bind:this={chartEl}></div>
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
    margin-bottom: 16px;
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

  .chart-container {
    min-height: 220px;
  }

  .chart-card :global(.uplot) {
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  }

  .chart-card :global(.u-title) {
    font-size: 13px !important;
    font-weight: 600 !important;
    color: #334155 !important;
  }

  .chart-card :global(.u-legend) {
    font-size: 12px;
  }
</style>
