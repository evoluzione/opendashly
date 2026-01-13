<script lang="ts">
  import { onMount } from 'svelte';
  import uPlot from 'uplot';
  import 'uplot/dist/uPlot.min.css';

  export let series: { points: { timestamp: string; value: number }[] }[] = [];
  let chartEl: HTMLDivElement;

  onMount(() => {
    if (!chartEl || series.length === 0) return;
    const data = [
      series[0].points.map((p) => new Date(p.timestamp).getTime() / 1000),
      series[0].points.map((p) => p.value)
    ];
    new uPlot(
      {
        title: 'Metrics',
        width: 600,
        height: 240
      },
      data,
      chartEl
    );
  });
</script>

<div bind:this={chartEl} class="chart"></div>

<style>
  .chart {
    border: 1px solid #e5e5e5;
    padding: 8px;
  }
</style>
