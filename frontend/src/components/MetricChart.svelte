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
        title: 'Metrics',
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
      Previous
    </button>
    <span>Page {pagination.page} of {pagination.totalPages || 1}</span>
    <button
      type="button"
      on:click={() => changePage(pagination.page + 1)}
      disabled={pagination.totalPages > 0 ? pagination.page >= pagination.totalPages : series.length === 0}
    >
      Next
    </button>
  </div>
{/if}

<style>
  .chart {
    border: 1px solid #e5e5e5;
    padding: 8px;
  }
  .pager {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
  }
</style>
