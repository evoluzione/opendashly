<script lang="ts">
  import { onDestroy, onMount, afterUpdate } from 'svelte';
  import uPlot from 'uplot';
  import 'uplot/dist/uPlot.min.css';
  import type { LatencyPercentilePoint } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';
  import { locale, t } from '../../lib/i18n';

  export let data: LatencyPercentilePoint[] = [];

  let containerEl: HTMLDivElement;
  let chartEl: HTMLDivElement;
  let chart: uPlot | null = null;
  let containerWidth = 0;
  let containerHeight = 0;
  let resizeObserver: ResizeObserver | null = null;
  let lastDataLength = 0;
  let lastWidth = 0;
  let lastHeight = 0;
  let lastDataRef: LatencyPercentilePoint[] | null = null;

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

  function buildChartData(points: LatencyPercentilePoint[]) {
    if (!points || points.length === 0) return null;
    const rows = points
      .map((p) => {
        const x = toEpochSeconds(p.timestamp);
        if (x === null) return null;
        return {
          x,
          p50: toFiniteNumber(p.p50),
          p95: toFiniteNumber(p.p95),
          p99: toFiniteNumber(p.p99)
        };
      })
      .filter((row): row is { x: number; p50: number | null; p95: number | null; p99: number | null } => row !== null)
      .sort((a, b) => a.x - b.x);

    if (rows.length === 0) return null;

    const xValues = rows.map((row) => row.x);
    const p50Values = rows.map((row) => row.p50);
    const p95Values = rows.map((row) => row.p95);
    const p99Values = rows.map((row) => row.p99);
    return [xValues, p50Values, p95Values, p99Values];
  }

  function getPlotHeight(containerH: number): number {
    const available = containerH - 110;
    return Math.max(180, Math.min(420, Math.floor(available)));
  }

  function renderChart(points: LatencyPercentilePoint[], width: number, height: number) {
    if (!chartEl || !width || points.length === 0) return;
    if (chart) {
      chart.destroy();
      chart = null;
    }

    const chartData = buildChartData(points);
    if (!chartData) return;

    chart = new uPlot(
      {
        title: 'Latenza percentili',
        width,
        height,
        series: [
          {},
          { label: 'P50', stroke: '#0ea5e9', width: 2 },
          { label: 'P95', stroke: '#f97316', width: 2 },
          { label: 'P99', stroke: 'var(--color-danger-500)', width: 2 }
        ],
        scales: {
          x: { time: true },
          y: { min: 0 }
        },
        axes: [{}, { label: 'ms', labelSize: 12, size: 50 }],
        legend: { show: true }
      },
      chartData as uPlot.AlignedData,
      chartEl
    );

    requestAnimationFrame(() => {
      chart?.setSize({ width, height });
      const canvases = chartEl.querySelectorAll('canvas');
      canvases.forEach((canvas) => {
        canvas.style.width = '100%';
        canvas.style.height = '100%';
      });
      chart?.setData(chartData as uPlot.AlignedData);
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
      Latenza Percentili
      <InfoTooltip text="Andamento dei percentili P50/P95/P99 nel tempo. Aiuta a individuare regressioni di performance." />
    </span>
    <span class="chart-subtitle">{t($locale, 'dashboard.latencyPercentiles.subtitle')}</span>
  </div>

  {#if data.length === 0}
    <div class="empty">{t($locale, 'dashboard.noData')}</div>
  {/if}
  <div class="chart-container" class:hidden={data.length === 0} bind:this={chartEl}></div>
</div>

<style>
  .chart-card {
    background: white;
    border-radius: 16px;
    padding: 20px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(var(--rgb-slate-950), 0.06);
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
    color: var(--color-slate-950);
  }

  .chart-subtitle {
    display: block;
    font-size: 12px;
    color: var(--color-slate-400);
    margin-top: 2px;
  }

  .empty {
    text-align: center;
    color: var(--color-slate-400);
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
</style>
