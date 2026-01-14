<script lang="ts">
  import { onDestroy } from 'svelte';
  import { createEventDispatcher } from 'svelte';
  import uPlot from 'uplot';
  import 'uplot/dist/uPlot.min.css';

  export let series: { points: { timestamp: string; value: number }[] }[] = [];
  export let pagination: { page: number; totalPages: number } | null = null;
  let chartEl: HTMLDivElement;
  let chart: uPlot | null = null;
  const dispatch = createEventDispatcher();

  function renderChart() {
    if (!chartEl || series.length === 0) {
      if (chart) {
        chart.destroy();
        chart = null;
      }
      return;
    }
    if (chart) {
      chart.destroy();
      chart = null;
    }
    const data = [
      series[0].points.map((p) => new Date(p.timestamp).getTime() / 1000),
      series[0].points.map((p) => p.value)
    ];
    chart = new uPlot(
      {
        title: 'Metriche',
        width: 600,
        height: 240
      },
      data,
      chartEl
    );
  }

  onDestroy(() => {
    if (chart) {
      chart.destroy();
      chart = null;
    }
  });

  $: if (chartEl && series) {
    renderChart();
  }

  function changePage(nextPage: number) {
    dispatch('pageChange', { page: nextPage });
  }
</script>

<div bind:this={chartEl} class="chart"></div>

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
    border-color: #6366f1;
    color: #6366f1;
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
