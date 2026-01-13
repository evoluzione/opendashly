<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  let service = '';
  let from = '';
  let to = '';

  function submit() {
    dispatch('run', {
      signals: ['logs', 'traces', 'metrics'],
      timeRange: { from, to },
      filters: service ? { 'service.name': service } : {}
    });
  }
</script>

<div class="query-form">
  <div>
    <label>Service</label>
    <input bind:value={service} placeholder="service.name" />
  </div>
  <div>
    <label>From</label>
    <input bind:value={from} placeholder="2026-01-01T00:00:00Z" />
  </div>
  <div>
    <label>To</label>
    <input bind:value={to} placeholder="2026-01-02T00:00:00Z" />
  </div>
  <button on:click={submit}>Run Query</button>
</div>

<style>
  .query-form {
    display: grid;
    gap: 12px;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    margin-bottom: 16px;
  }
  label {
    display: block;
    font-size: 12px;
    text-transform: uppercase;
  }
  input {
    width: 100%;
    padding: 8px;
  }
</style>
