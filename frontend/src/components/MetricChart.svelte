<script lang="ts">
  import { onDestroy, onMount, afterUpdate } from 'svelte';
  import { createEventDispatcher } from 'svelte';
  import uPlot from 'uplot';
  import 'uplot/dist/uPlot.min.css';

  export let series: { name?: string; unit?: string; points: { timestamp: string; value: number }[] }[] = [];
  export let pagination: { page: number; totalPages: number } | null = null;

  let containerEl: HTMLDivElement;
  let chartEl: HTMLDivElement;
  let chart: uPlot | null = null;
  let containerWidth = 0;
  let activeSeries = 0;
  let resizeObserver: ResizeObserver | null = null;
  let lastSeriesLength = 0;
  let lastActiveSeries = -1;
  let lastWidth = 0;

  const dispatch = createEventDispatcher();
  const palette = ['#2563eb', '#16a34a', '#f97316', '#ef4444', '#0ea5e9', '#0f766e'];

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

  function buildChartData(selected: typeof series[number]) {
    if (!selected || !selected.points || selected.points.length === 0) return null;
    const rows = selected.points
      .map((point) => {
        const x = toEpochSeconds(point.timestamp);
        if (x === null) return null;
        return {
          x,
          y: toFiniteNumber(point.value)
        };
      })
      .filter((row): row is { x: number; y: number | null } => row !== null)
      .sort((a, b) => a.x - b.x);

    if (rows.length === 0) return null;

    const xValues = rows.map((row) => row.x);
    const yValues = rows.map((row) => row.y);
    return [xValues, yValues];
  }

  function renderChart(selectedIndex: number, metricSeries: typeof series, width: number) {
    if (!chartEl || !width) return;
    if (chart) {
      chart.destroy();
      chart = null;
    }

    const selected = metricSeries[selectedIndex];
    const data = selected ? buildChartData(selected) : null;
    if (!data) return;

    chart = new uPlot(
      {
        title: selected?.name ? `${selected.name}${selected.unit ? ` (${selected.unit})` : ''}` : 'Metriche',
        width,
        height: 260,
        series: [
          {},
          {
            label: selected?.name ?? 'serie',
            stroke: palette[selectedIndex % palette.length],
            width: 2
          }
        ],
        scales: {
          x: { time: true }
        }
      },
      data,
      chartEl
    );

    requestAnimationFrame(() => {
      chart?.setSize({ width, height: 260 });
      const canvases = chartEl.querySelectorAll('canvas');
      canvases.forEach((canvas) => {
        canvas.style.width = '100%';
        canvas.style.height = '100%';
      });
      chart?.setData(data);
    });
  }

  function changePage(nextPage: number) {
    dispatch('pageChange', { page: nextPage });
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

  $: if (activeSeries >= series.length) {
    activeSeries = 0;
  }

  $: if (series.length === 0 && chart) {
    chart.destroy();
    chart = null;
  }

  afterUpdate(() => {
    if (chartEl && containerWidth > 0 && series.length > 0) {
      const needsRender = !chart || lastSeriesLength !== series.length || lastActiveSeries !== activeSeries || lastWidth !== containerWidth;
      if (needsRender) {
        lastSeriesLength = series.length;
        lastActiveSeries = activeSeries;
        lastWidth = containerWidth;
        renderChart(activeSeries, series, Math.max(320, containerWidth - 40));
      }
    }
  });
</script>

<div class="metric-panel" bind:this={containerEl}>
  {#if series.length === 0}
    <div class="empty">Nessuna metrica disponibile per l'intervallo selezionato.</div>
  {:else}
    <div class="metric-toolbar">
      <label for="metric-select">Metriche</label>
      <select id="metric-select" bind:value={activeSeries}>
        {#each series as metric, idx}
          <option value={idx}>
            {metric.name ?? 'Serie'}{metric.unit ? ` (${metric.unit})` : ''}
          </option>
        {/each}
      </select>
    </div>
    <div class="chart">
      {#if !series[activeSeries]?.points || series[activeSeries].points.length === 0}
        <div class="empty">Nessun punto disponibile per questa metrica.</div>
      {/if}
    </div>
  {/if}
  <div class="chart-canvas" class:hidden={series.length === 0} bind:this={chartEl}></div>
</div>

{#if pagination}
  <div class="pager">
    <button
      type="button"
      on:click={() => changePage(pagination.page - 1)}
      disabled={pagination.page <= 1}
    >
      Precedente
    </button>
    <span>Pagina {pagination.page} di {pagination.totalPages || 1}</span>
    <button
      type="button"
      on:click={() => changePage(pagination.page + 1)}
      disabled={pagination.totalPages > 0 ? pagination.page >= pagination.totalPages : series.length === 0}
    >
      Successiva
    </button>
  </div>
{/if}

<style>
  .metric-panel {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .empty {
    color: #94a3b8;
    font-size: 14px;
    text-align: center;
    padding: 40px 0;
  }

  .metric-toolbar {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .metric-toolbar label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #94a3b8;
  }

  .metric-toolbar select {
    padding: 10px 12px;
    border-radius: 10px;
    border: 1px solid rgba(148, 163, 184, 0.35);
    background: #f8fafc;
    color: #0f172a;
    font-size: 13px;
    font-weight: 600;
  }

  .chart {
    background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
    border: 1px solid #e2e8f0;
    border-radius: 16px;
    padding: 20px;
    min-height: 280px;
  }

  .chart :global(.uplot) {
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  }

  .chart :global(.u-title) {
    font-size: 14px !important;
    font-weight: 600 !important;
    color: #334155 !important;
  }

  .chart-canvas :global(.u-wrap) {
    position: relative;
  }

  .chart-canvas :global(.u-under),
  .chart-canvas :global(.u-over) {
    position: absolute;
  }

  .chart-canvas :global(.uplot canvas) {
    display: block;
    width: 100% !important;
    height: 100% !important;
  }

  .chart-canvas {
    min-height: 240px;
  }

  .chart-canvas.hidden {
    display: none;
  }

  .pager {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    margin-top: 20px;
    padding-top: 20px;
    border-top: 1px solid #f1f5f9;
  }

  .pager button {
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 500;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    background: white;
    color: #475569;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .pager button:hover:not(:disabled) {
    border-color: #2563eb;
    color: #2563eb;
  }

  .pager button:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .pager span {
    font-size: 13px;
    color: #64748b;
  }

</style>
