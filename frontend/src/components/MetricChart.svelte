<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
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

  const dispatch = createEventDispatcher();
  const palette = ['#2563eb', '#16a34a', '#f97316', '#ef4444', '#0ea5e9', '#0f766e'];

  function buildChartData(selected: typeof series[number]) {
    if (!selected || !selected.points || selected.points.length === 0) return null;
    const xValues = selected.points.map((point) => new Date(point.timestamp).getTime() / 1000);
    const yValues = selected.points.map((point) => point.value ?? 0);
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
            label: selected?.name ?? 'series',
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

  $: if (chartEl && containerWidth > 0) {
    renderChart(activeSeries, series, Math.max(320, containerWidth - 40));
  }
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
            {metric.name ?? 'Series'}{metric.unit ? ` (${metric.unit})` : ''}
          </option>
        {/each}
      </select>
    </div>
    <div class="chart">
      <div class="chart-canvas" bind:this={chartEl}></div>
      {#if !series[activeSeries]?.points || series[activeSeries].points.length === 0}
        <div class="empty">Nessun punto disponibile per questa metrica.</div>
      {/if}
    </div>
  {/if}
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

  .chart-canvas {
    min-height: 240px;
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
