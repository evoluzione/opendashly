<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { servicesState } from '../lib/stores/query';
  import { generateSmartQuery } from '../services/query';
  import type { QueryRequest } from '../services/query';

  const dispatch = createEventDispatcher();

  type SearchMode = 'auto' | 'manual' | 'smart';

  let searchMode: SearchMode = 'auto';
  let fromInput = '';
  let toInput = '';
  let autoRangeMinutes: number | null = 5;
  let autoRefreshSeconds: number | null = 10;
  let smartPrompt = '';
  let smartError = '';
  let smartLoading = false;
  let smartRequest: QueryRequest | null = null;

  const quickRanges = [
    { label: 'Ultimi 5 minuti', minutes: 5 },
    { label: 'Ultimi 10 minuti', minutes: 10 },
    { label: 'Ultimi 30 minuti', minutes: 30 },
    { label: 'Ultima ora', minutes: 60 },
    { label: 'Tutto', minutes: null }
  ];
  const autoRefreshOptions = [
    { label: '5 s', seconds: 5 },
    { label: '10 s', seconds: 10 },
    { label: '60 s', seconds: 60 },
    { label: '5 minuti', seconds: 300 }
  ];


  function formatDateTimeLocal(date: Date) {
    const pad = (value: number) => String(value).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }

  function applyQuickRange(minutes: number | null) {
    const now = new Date();
    if (minutes === null) {
      fromInput = '';
      toInput = '';
      return;
    }
    const fromDate = new Date(now.getTime() - minutes * 60 * 1000);
    fromInput = formatDateTimeLocal(fromDate);
    toInput = formatDateTimeLocal(now);
  }

  function formatAutoRefreshLabel(seconds: number | null) {
    if (!seconds) return 'disattivato';
    if (seconds === 60) return '1 minuto';
    if (seconds === 300) return '5 minuti';
    return `${seconds} secondi`;
  }

  function handleAutoRefreshChange(event: Event) {
    const target = event.target as HTMLSelectElement;
    autoRefreshSeconds = Number(target.value);
    submit();
  }

  function toIso(value: string) {
    return value ? new Date(value).toISOString() : '';
  }

  onMount(() => {
    applyQuickRange(autoRangeMinutes);
    if (searchMode === 'auto') {
      submit();
    }
  });

  function handleModeChange(nextMode: SearchMode) {
    if (searchMode === nextMode) return;
    searchMode = nextMode;
    smartError = '';
    if (searchMode === 'auto') {
      applyQuickRange(autoRangeMinutes);
      submit();
    }
    dispatch('modeChange', { mode: searchMode });
  }

  async function generateSql() {
    smartError = '';
    smartRequest = null;
    const prompt = smartPrompt.trim();
    if (!prompt) {
      smartError = 'Inserisci una richiesta in linguaggio naturale.';
      return;
    }
    smartLoading = true;
    try {
      const response = await generateSmartQuery({ prompt });
      smartPrompt = response.sql;
      smartRequest = response.request;
    } catch (err) {
      smartError = err instanceof Error ? err.message : 'Impossibile generare la query.';
    } finally {
      smartLoading = false;
    }
  }

  function submit() {
    const limit = 100;
    const selectedService = $servicesState.selectedService;
    const serviceFilter =
      selectedService && selectedService !== 'Tutti' ? { 'service.name': selectedService } : {};
    if (searchMode === 'smart') {
      if (!smartRequest) {
        return;
      }
      dispatch('run', {
        request: {
          ...smartRequest,
          filters: { ...(smartRequest.filters ?? {}), ...serviceFilter },
          page: 1,
          limit
        },
        autoRefreshSeconds: null,
        autoRefreshRangeMinutes: null
      });
      return;
    }

    const rangeMinutes = searchMode === 'auto' ? autoRangeMinutes : null;
    const refreshSeconds = searchMode === 'auto' ? autoRefreshSeconds ?? 10 : 0;
    if (searchMode === 'auto') {
      applyQuickRange(autoRangeMinutes);
    }
    const zeroTime = '0001-01-01T00:00:00Z';
    const from = rangeMinutes === null ? zeroTime : toIso(fromInput);
    const to = rangeMinutes === null ? zeroTime : toIso(toInput);
    if (!from || !to) {
      return;
    }
    dispatch('run', {
      request: {
        signals: ['logs', 'traces', 'metrics'],
        timeRange: { from, to },
        filters: serviceFilter,
        page: 1,
        limit
      },
      autoRefreshSeconds: refreshSeconds || null,
      autoRefreshRangeMinutes: rangeMinutes
    });
  }
</script>

<div class="query-form">
  <fieldset class="mode-picker">
    <legend>Modalita di ricerca</legend>
    <div class="mode-buttons">
      <button
        type="button"
        class:active={searchMode === 'auto'}
        on:click={() => handleModeChange('auto')}
      >
        Automatica
      </button>
      <button
        type="button"
        class:active={searchMode === 'manual'}
        on:click={() => handleModeChange('manual')}
      >
        Manuale
      </button>
      <button
        type="button"
        class:active={searchMode === 'smart'}
        on:click={() => handleModeChange('smart')}
      >
        Smart
      </button>
    </div>
  </fieldset>

  {#if searchMode === 'auto'}
    <fieldset class="quick-range">
      <legend>Intervallo automatico</legend>
      <div class="quick-range-buttons">
        {#each quickRanges as range}
          <button
            type="button"
            class:active={autoRangeMinutes === range.minutes}
            on:click={() => {
              autoRangeMinutes = range.minutes;
              applyQuickRange(range.minutes);
              submit();
            }}
          >
            {range.label}
          </button>
        {/each}
      </div>
      <div class="auto-refresh">
        <label for="auto-refresh">Aggiornamento</label>
        <select
          id="auto-refresh"
          value={autoRefreshSeconds ?? 10}
          on:change={handleAutoRefreshChange}
        >
          {#each autoRefreshOptions as option}
            <option value={option.seconds}>{option.label}</option>
          {/each}
        </select>
      </div>
      <p class="helper">
        Aggiornamento automatico ogni {formatAutoRefreshLabel(autoRefreshSeconds ?? 10)}.
      </p>
    </fieldset>
  {:else if searchMode === 'manual'}
    <div class="manual-range">
      <div>
        <label for="query-from">Da</label>
        <input id="query-from" type="datetime-local" bind:value={fromInput} />
      </div>
      <div>
        <label for="query-to">A</label>
        <input id="query-to" type="datetime-local" bind:value={toInput} />
      </div>
    </div>
  {:else}
    <div class="smart-box">
      <label for="smart-prompt">Prompt in linguaggio naturale</label>
      <textarea
        id="smart-prompt"
        rows="4"
        bind:value={smartPrompt}
        placeholder="Es: Mostrami gli errori del servizio checkout negli ultimi 10 minuti"
      ></textarea>
      <div class="smart-actions">
        <button type="button" on:click={generateSql} disabled={smartLoading}>
          {smartLoading ? 'Genero...' : 'Genera'}
        </button>
        {#if smartError}
          <span class="error">{smartError}</span>
        {/if}
      </div>
      {#if smartRequest}
        <p class="helper">Query SQL generata e pronta per l'esecuzione.</p>
      {/if}
    </div>
  {/if}

  {#if searchMode !== 'auto'}
    <button
      on:click={submit}
      disabled={searchMode !== 'smart' ? !fromInput || !toInput : !smartRequest}
    >
      Esegui query
    </button>
  {/if}
</div>

<style>
  .query-form {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .mode-picker,
  .quick-range {
    border: none;
    padding: 0;
    margin: 0;
  }

  .mode-picker legend,
  .quick-range legend {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
    margin-bottom: 10px;
  }

  .mode-buttons {
    display: inline-flex;
    gap: 8px;
    padding: 6px;
    border-radius: 999px;
    background: #f1f5f9;
  }

  .mode-buttons button {
    padding: 8px 14px;
    font-size: 12px;
    font-weight: 600;
    border: none;
    border-radius: 999px;
    background: transparent;
    color: #475569;
    cursor: pointer;
  }

  .mode-buttons button.active {
    background: white;
    color: #0f172a;
    box-shadow: 0 4px 12px rgba(15, 23, 42, 0.08);
  }

  .quick-range-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .quick-range .helper {
    margin-top: 8px;
  }

  .auto-refresh {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 12px;
  }

  .auto-refresh label {
    margin-bottom: 0;
  }

  .auto-refresh select {
    width: auto;
    min-width: 120px;
  }

  .manual-range {
    display: grid;
    gap: 12px;
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

  .quick-range-buttons button.active {
    border-color: #6366f1;
    color: #4338ca;
    background: rgba(99, 102, 241, 0.1);
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

  textarea {
    width: 100%;
    padding: 12px 14px;
    font-size: 13px;
    border: 1px solid #e2e8f0;
    border-radius: 10px;
    background: #f8fafc;
    color: #0f172a;
    resize: vertical;
  }

  textarea:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
    background: white;
  }

  .smart-box {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .smart-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .smart-actions .error {
    color: #b91c1c;
    font-size: 12px;
    font-weight: 600;
  }

  .helper {
    margin: 0;
    font-size: 12px;
    color: #64748b;
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
