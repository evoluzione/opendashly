<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { servicesState } from '../lib/stores/query';

  const dispatch = createEventDispatcher();

  let fromInput = '';
  let toInput = '';
  let autoRefreshSeconds = '0';
  let pageSize = '100';

  const quickRanges = [
    { label: 'Ultimi 5 minuti', minutes: 5 },
    { label: 'Ultimi 30 minuti', minutes: 30 },
    { label: 'Ultimi 3 giorni', minutes: 60 * 24 * 3 }
  ];

  const refreshOptions = [
    { label: 'Disattivato', value: '0' },
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
    const selectedService = $servicesState.selectedService;
    const filters =
      selectedService && selectedService !== 'Tutti' ? { 'service.name': selectedService } : {};
    dispatch('run', {
      request: {
        signals: ['logs', 'traces', 'metrics'],
        timeRange: { from, to },
        filters,
        page: 1,
        limit
      },
      autoRefreshSeconds: refreshSeconds || null
    });
  }
</script>

<div class="query-form">
  <fieldset class="quick-range">
    <legend>Intervallo rapido</legend>
    <div class="quick-range-buttons">
      {#each quickRanges as range}
        <button type="button" on:click={() => applyQuickRange(range.minutes)}>
          {range.label}
        </button>
      {/each}
    </div>
  </fieldset>
  <div>
    <label for="query-from">Da</label>
    <input id="query-from" type="datetime-local" bind:value={fromInput} />
  </div>
  <div>
    <label for="query-to">A</label>
    <input id="query-to" type="datetime-local" bind:value={toInput} />
  </div>
  <div>
    <label for="query-refresh">Aggiornamento automatico</label>
    <select id="query-refresh" bind:value={autoRefreshSeconds}>
      {#each refreshOptions as option}
        <option value={option.value}>{option.label}</option>
      {/each}
    </select>
  </div>
  <div>
    <label for="query-page-size">Risultati per pagina</label>
    <select id="query-page-size" bind:value={pageSize}>
      {#each pageSizeOptions as size}
        <option value={size}>{size}</option>
      {/each}
    </select>
  </div>
  <button on:click={submit} disabled={!fromInput || !toInput}>Esegui query</button>
</div>

<style>
  .query-form {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  
  .quick-range {
    border: none;
    padding: 0;
    margin: 0;
  }
  
  .quick-range legend {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
    margin-bottom: 10px;
  }
  
  .quick-range-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  
  .quick-range-buttons button {
    padding: 8px 12px;
    font-size: 12px;
    font-weight: 500;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    background: #f8fafc;
    color: #475569;
    cursor: pointer;
    transition: all 0.2s ease;
  }
  
  .quick-range-buttons button:hover {
    border-color: #6366f1;
    color: #6366f1;
    background: rgba(99, 102, 241, 0.05);
  }
  
  label {
    display: block;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
    margin-bottom: 6px;
  }
  
  input,
  select {
    width: 100%;
    padding: 12px 14px;
    font-size: 14px;
    border: 1px solid #e2e8f0;
    border-radius: 10px;
    background: #f8fafc;
    color: #0f172a;
    transition: all 0.2s ease;
  }
  
  input:focus,
  select:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
    background: white;
  }
  
  .query-form > button {
    margin-top: 8px;
    padding: 14px 20px;
    font-size: 14px;
    font-weight: 600;
    border: none;
    border-radius: 10px;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: white;
    cursor: pointer;
    transition: all 0.2s ease;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
  }
  
  .query-form > button:hover:not([disabled]) {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(99, 102, 241, 0.4);
  }
  
  .query-form > button[disabled] {
    opacity: 0.5;
    cursor: not-allowed;
    transform: none;
  }
</style>
