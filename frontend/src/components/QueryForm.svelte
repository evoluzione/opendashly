<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';

  const dispatch = createEventDispatcher();

  let service = '';
  let fromInput = '';
  let toInput = '';
  let autoRefreshSeconds = '0';
  let pageSize = '100';

  const quickRanges = [
    { label: 'Last 5 minutes', minutes: 5 },
    { label: 'Last 30 minutes', minutes: 30 },
    { label: 'Last 3 days', minutes: 60 * 24 * 3 }
  ];

  const refreshOptions = [
    { label: 'Off', value: '0' },
    { label: '5s', value: '5' },
    { label: '15s', value: '15' },
    { label: '30s', value: '30' },
    { label: '60s', value: '60' }
  ];

  const pageSizeOptions = ['25', '50', '100', '200'];

  function formatDateTimeLocal(date: Date) {
    const pad = (value: number) => String(value).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }

  function applyQuickRange(minutes: number) {
    const now = new Date();
    const fromDate = new Date(now.getTime() - minutes * 60 * 1000);
    fromInput = formatDateTimeLocal(fromDate);
    toInput = formatDateTimeLocal(now);
  }

  function toIso(value: string) {
    return value ? new Date(value).toISOString() : '';
  }

  onMount(() => {
    applyQuickRange(5);
  });

  function submit() {
    const from = toIso(fromInput);
    const to = toIso(toInput);
    const refreshSeconds = Number(autoRefreshSeconds) || 0;
    const limit = Number(pageSize) || 100;
    if (!from || !to) {
      return;
    }
    dispatch('run', {
      request: {
        signals: ['logs', 'traces', 'metrics'],
        timeRange: { from, to },
        filters: service ? { 'service.name': service } : {},
        page: 1,
        limit
      },
      autoRefreshSeconds: refreshSeconds || null
    });
  }
</script>

<div class="query-form">
  <div>
    <label for="query-service">Service</label>
    <input id="query-service" bind:value={service} placeholder="service.name" />
  </div>
  <div>
    <label>Quick Range</label>
    <div class="quick-range">
      {#each quickRanges as range}
        <button type="button" on:click={() => applyQuickRange(range.minutes)}>
          {range.label}
        </button>
      {/each}
    </div>
  </div>
  <div>
    <label for="query-from">From</label>
    <input id="query-from" type="datetime-local" bind:value={fromInput} />
  </div>
  <div>
    <label for="query-to">To</label>
    <input id="query-to" type="datetime-local" bind:value={toInput} />
  </div>
  <div>
    <label for="query-refresh">Auto Refresh</label>
    <select id="query-refresh" bind:value={autoRefreshSeconds}>
      {#each refreshOptions as option}
        <option value={option.value}>{option.label}</option>
      {/each}
    </select>
  </div>
  <div>
    <label for="query-page-size">Page Size</label>
    <select id="query-page-size" bind:value={pageSize}>
      {#each pageSizeOptions as size}
        <option value={size}>{size}</option>
      {/each}
    </select>
  </div>
  <button on:click={submit} disabled={!fromInput || !toInput}>Run Query</button>
</div>

<style>
  .query-form {
    display: grid;
    gap: 12px;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    margin-bottom: 16px;
  }
  .quick-range {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  label {
    display: block;
    font-size: 12px;
    text-transform: uppercase;
  }
  input,
  select {
    width: 100%;
    padding: 8px;
  }
  button[disabled] {
    opacity: 0.6;
    cursor: not-allowed;
  }
</style>
