<script lang="ts">
  import { onDestroy, onMount, afterUpdate } from 'svelte';
  import uPlot from 'uplot';
  import 'uplot/dist/uPlot.min.css';
  import type { ThroughputPoint } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';

  export let data: ThroughputPoint[] = [];

  let containerEl: HTMLDivElement;
  let chartEl: HTMLDivElement;
  let chart: uPlot | null = null;
  let containerWidth = 0;
  let containerHeight = 0;
  let resizeObserver: ResizeObserver | null = null;
  let lastDataLength = 0;
  let lastWidth = 0;
  let lastHeight = 0;
  let lastDataRef: ThroughputPoint[] | null = null;

  function toEpochSeconds(value: unknown): number | null {
    if (value instanceof Date) {
      const ms = value.getTime();
      return Number.isFinite(ms) ? ms / 1000 : null;
    }
    if (typeof value === 'number') {
      if (!Number.isFinite(value)) return null;
      return value > 1e11 ? value / 1000 : value;
    }
    if (typeof value === 'string') {
      const numeric = Number(value);
      if (Number.isFinite(numeric)) {
        return numeric > 1e11 ? numeric / 1000 : numeric;
      }
      const parsed = Date.parse(value);
      if (Number.isFinite(parsed)) return parsed / 1000;
    }
    return null;
  }

  function toFiniteNumber(value: unknown): number | null {
    const num = typeof value === 'number' ? value : Number(value);
    return Number.isFinite(num) ? num : null;
  }

  function buildChartData(points: ThroughputPoint[]) {
    if (!points || points.length === 0) return null;
    const rows = points
      .map((p) => {
        const x = toEpochSeconds(p.timestamp);
        if (x === null) return null;
        const requestCount = toFiniteNumber(p.requestCount);
        const errorCount = toFiniteNumber(p.errorCount);
        if (requestCount === null && errorCount === null) return null;
        return {
          x,
          request: requestCount ?? 0,
          error: errorCount ?? 0
        };
      })
      .filter((row): row is { x: number; request: number; error: number } => row !== null)
      .sort((a, b) => a.x - b.x);

    if (rows.length === 0) return null;

    const xValues = rows.map((row) => row.x);
    const requestValues = rows.map((row) => row.request);
    const errorValues = rows.map((row) => row.error);
    return [xValues, requestValues, errorValues];
  }

  function getPlotHeight(containerH: number): number {
    const available = containerH - 110;
    return Math.max(180, Math.min(420, Math.floor(available)));
  }

  function renderChart(points: ThroughputPoint[], width: number, height: number) {
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
        height,
        series: [
          {},
          {
            show: true,
            label: 'Richieste',
            stroke: '#2563eb',
            width: 2,
            fill: 'rgba(37, 99, 235, 0.1)',
            points: { show: true, size: 6 }
          },
          {
            show: true,
            label: 'Errori',
            stroke: '#ef4444',
            width: 2,
            fill: 'rgba(239, 68, 68, 0.1)',
            points: { show: true, size: 6 }
          }
        ],
        scales: {
          x: { time: true },
          y: { min: 0, auto: true }
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

    requestAnimationFrame(() => {
      chart?.setSize({ width, height });
      const canvases = chartEl.querySelectorAll('canvas');
      canvases.forEach((canvas) => {
        canvas.style.width = '100%';
        canvas.style.height = '100%';
      });
      chart?.setData(chartData);
    });
  }

  onMount(() => {
    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver((entries) => {
        for (const entry of entries) {
          containerWidth = Math.max(320, Math.floor(entry.contentRect.width));
          containerHeight = Math.max(260, Math.floor(entry.contentRect.height));
        }
      });
      if (containerEl) {
        resizeObserver.observe(containerEl);
        containerWidth = Math.max(320, Math.floor(containerEl.clientWidth));
        containerHeight = Math.max(260, Math.floor(containerEl.clientHeight));
      }
    } else if (containerEl) {
      containerWidth = Math.max(320, Math.floor(containerEl.clientWidth));
      containerHeight = Math.max(260, Math.floor(containerEl.clientHeight));
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

  afterUpdate(() => {
    if (chartEl && containerWidth > 0 && containerHeight > 0 && data.length > 0) {
      const nextHeight = getPlotHeight(containerHeight);
      const needsRender =
        !chart ||
        lastDataRef !== data ||
        lastDataLength !== data.length ||
        lastWidth !== containerWidth ||
        lastHeight !== nextHeight;
      if (needsRender) {
        lastDataRef = data;
        lastDataLength = data.length;
        lastWidth = containerWidth;
        lastHeight = nextHeight;
        renderChart(data, Math.max(320, containerWidth - 40), nextHeight);
      }
    }
  });

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
  {/if}
  <div class="chart-container" class:hidden={data.length === 0} bind:this={chartEl}></div>
</div>

<style>
  .chart-card {
    background: white;
    border-radius: 16px;
    padding: 20px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(15, 23, 42, 0.06);
    display: flex;
    flex-direction: column;
    min-height: 0;
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
    width: 100%;
    flex: 1;
    min-height: 180px;
  }

  .chart-container.hidden {
    display: none;
  }

  .chart-card :global(.uplot) {
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
    width: 100% !important;
  }

  .chart-card :global(.u-wrap) {
    position: relative;
  }

  .chart-card :global(.u-under),
  .chart-card :global(.u-over) {
    position: absolute;
  }

  .chart-card :global(.uplot canvas) {
    display: block;
    width: 100% !important;
    height: 100% !important;
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
