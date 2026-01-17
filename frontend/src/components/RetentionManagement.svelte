<script lang="ts">
  import { onMount } from 'svelte';
  import {
    getRetentionSettings,
    updateRetentionSetting,
    executeCleanup,
    listCleanupJobs,
    type RetentionSetting,
    type SignalType,
    type CleanupJob
  } from '../services/retention';
  import { fetchServices } from '../services/services';

  let settings: RetentionSetting[] = [];
  let jobs: CleanupJob[] = [];
  let services: string[] = [];
  let loading = false;
  let error = '';
  let successMessage = '';

  let editingSignal: SignalType | null = null;
  let editRetentionDays = 7;

  let selectedSignals: SignalType[] = [];
  let selectedService = '';
  let cleanupLoading = false;
  const signalLabels: Record<SignalType, string> = {
    logs: 'Log',
    traces: 'Tracce',
    metrics: 'Metriche'
  };
  const statusLabels: Record<string, string> = {
    completed: 'Completato',
    running: 'In corso',
    failed: 'Fallito'
  };

  function labelForSignal(signal: SignalType) {
    return signalLabels[signal] ?? signal;
  }

  function labelList(signals: SignalType[]) {
    return signals.map(labelForSignal).join(', ');
  }

  function labelForStatus(status: string) {
    return statusLabels[status] ?? status;
  }

  async function loadSettings() {
    loading = true;
    error = '';
    try {
      const result = await getRetentionSettings();
      settings = result.settings;
    } catch (err) {
      error = err instanceof Error ? err.message : 'Impossibile caricare le impostazioni';
    } finally {
      loading = false;
    }
  }

  async function loadJobs() {
    try {
      const result = await listCleanupJobs();
      jobs = result.jobs;
    } catch (err) {
      console.error('Failed to load jobs', err);
    }
  }

  async function loadServices() {
    try {
      const result = await fetchServices();
      services = result.services;
    } catch (err) {
      console.error('Failed to load services', err);
    }
  }

  function startEdit(setting: RetentionSetting) {
    editingSignal = setting.signalType;
    editRetentionDays = setting.retentionDays;
  }

  function cancelEdit() {
    editingSignal = null;
  }

  async function saveEdit() {
    if (!editingSignal) return;

    loading = true;
    error = '';
    successMessage = '';

    try {
      await updateRetentionSetting(editingSignal, editRetentionDays);
      successMessage = `Conservazione per ${labelForSignal(editingSignal)} aggiornata a ${editRetentionDays} giorni`;
      editingSignal = null;
      await loadSettings();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Impossibile aggiornare la conservazione';
    } finally {
      loading = false;
    }
  }

  async function runCleanup() {
    if (selectedSignals.length === 0) {
      error = 'Seleziona almeno un tipo di segnale';
      return;
    }

    const confirmMsg = selectedService
      ? `Eliminare tutti i dati ${labelList(selectedSignals)} per il servizio "${selectedService}"?`
      : `Eliminare TUTTI i dati ${labelList(selectedSignals)}? Questa azione non può essere annullata!`;

    if (!confirm(confirmMsg)) return;

    cleanupLoading = true;
    error = '';
    successMessage = '';

    try {
      const result = await executeCleanup({
        signalTypes: selectedSignals,
        serviceName: selectedService || undefined
      });

      const totalDeleted = result.results.reduce((sum, r) => sum + r.recordsDeleted, 0);
      successMessage = `Pulizia completata! ${totalDeleted} record eliminati (ID operazione: ${result.jobId})`;

      selectedSignals = [];
      selectedService = '';
      await loadJobs();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Impossibile eseguire la pulizia';
    } finally {
      cleanupLoading = false;
    }
  }

  function toggleSignal(signal: SignalType) {
    if (selectedSignals.includes(signal)) {
      selectedSignals = selectedSignals.filter(s => s !== signal);
    } else {
      selectedSignals = [...selectedSignals, signal];
    }
  }

  onMount(() => {
    void loadSettings();
    void loadJobs();
    void loadServices();
  });
</script>

<section class="retention-management">
  <header>
    <h2>Gestione conservazione e pulizia</h2>
    <p>Configura la conservazione dei dati ed esegui pulizie manuali.</p>
  </header>

  {#if error}
    <div class="alert error">{error}</div>
  {/if}
  {#if successMessage}
    <div class="alert success">{successMessage}</div>
  {/if}

  <div class="panel">
    <h3>Impostazioni conservazione</h3>
    <p class="help-text">
      I dati piu vecchi del periodo di conservazione verranno eliminati automaticamente ogni 24 ore.
    </p>

    {#if loading && settings.length === 0}
      <div class="status">Caricamento...</div>
    {:else}
      <table>
        <thead>
          <tr>
            <th>Tipo Segnale</th>
            <th>Conservazione (giorni)</th>
            <th>Azioni</th>
          </tr>
        </thead>
        <tbody>
          {#each settings as setting}
            <tr>
              <td class="signal-type">{labelForSignal(setting.signalType)}</td>
              <td>
                {#if editingSignal === setting.signalType}
                  <input
                    type="number"
                    bind:value={editRetentionDays}
                    min="1"
                    max="365"
                    class="edit-input"
                  />
                {:else}
                  {setting.retentionDays} giorni
                {/if}
              </td>
              <td>
                {#if editingSignal === setting.signalType}
                  <div class="action-buttons">
                    <button class="btn-small btn-primary" on:click={saveEdit}>Salva</button>
                    <button class="btn-small" on:click={cancelEdit}>Annulla</button>
                  </div>
                {:else}
                  <button class="btn-small" on:click={() => startEdit(setting)}>Modifica</button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  <div class="panel">
    <h3>Pulizia manuale</h3>
    <p class="help-text">
      Elimina manualmente tutti i dati o filtra per servizio specifico.
    </p>

    <div class="cleanup-form">
      <fieldset class="form-group">
        <legend>Tipi di segnale</legend>
        <div class="checkbox-group">
          <label>
            <input
              type="checkbox"
              checked={selectedSignals.includes('logs')}
              on:change={() => toggleSignal('logs')}
            />
            {labelForSignal('logs')}
          </label>
          <label>
            <input
              type="checkbox"
              checked={selectedSignals.includes('traces')}
              on:change={() => toggleSignal('traces')}
            />
            {labelForSignal('traces')}
          </label>
          <label>
            <input
              type="checkbox"
              checked={selectedSignals.includes('metrics')}
              on:change={() => toggleSignal('metrics')}
            />
            {labelForSignal('metrics')}
          </label>
        </div>
      </fieldset>

      <div class="form-group">
        <label for="service-select">Servizio (opzionale)</label>
        <select id="service-select" bind:value={selectedService}>
          <option value="">Tutti i servizi</option>
          {#each services as service}
            <option value={service}>{service}</option>
          {/each}
        </select>
        <p class="hint">Lascia vuoto per eliminare dati di tutti i servizi</p>
      </div>

      <button
        class="btn-danger"
        on:click={runCleanup}
        disabled={cleanupLoading || selectedSignals.length === 0}
      >
        {cleanupLoading ? 'Pulizia in corso...' : 'Esegui pulizia'}
      </button>
    </div>
  </div>

  <div class="panel">
    <h3>Storico pulizia</h3>
    {#if jobs.length === 0}
      <div class="status">Nessuna operazione di pulizia eseguita.</div>
    {:else}
      <table>
        <thead>
          <tr>
            <th>ID operazione</th>
            <th>Tipo</th>
            <th>Segnale</th>
            <th>Servizio</th>
            <th>Stato</th>
            <th>Record Eliminati</th>
            <th>Completato</th>
          </tr>
        </thead>
        <tbody>
          {#each jobs as job}
            <tr>
              <td class="monospace">{job.jobId.slice(0, 8)}</td>
              <td>{job.jobType}</td>
              <td>{job.signalType}</td>
              <td>{job.serviceName || 'tutti'}</td>
              <td>
                <span class="status-badge {job.status}">
                  {labelForStatus(job.status)}
                </span>
              </td>
              <td>{job.recordsDeleted.toLocaleString()}</td>
              <td>{job.completedAt ? new Date(job.completedAt).toLocaleString('it-IT') : '-'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</section>

<style>
  .retention-management {
    display: grid;
    gap: 20px;
  }

  header h2 {
    margin: 0 0 6px 0;
    font-size: 22px;
    color: #0f172a;
  }
  header p {
    margin: 0;
    color: #64748b;
    font-size: 14px;
  }

  .panel {
    background: #fff;
    border-radius: 16px;
    padding: 20px;
    border: 1px solid rgba(15, 23, 42, 0.06);
    box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
  }

  .panel h3 {
    margin: 0 0 12px 0;
    font-size: 16px;
    font-weight: 600;
  }

  .help-text {
    color: #64748b;
    font-size: 14px;
    margin: 0 0 16px 0;
  }

  .alert {
    padding: 12px 16px;
    border-radius: 10px;
    margin-bottom: 16px;
    font-size: 14px;
  }

  .alert.error {
    background: #fee;
    color: #b42318;
    border: 1px solid #fcc;
  }

  .alert.success {
    background: #efe;
    color: #027a48;
    border: 1px solid #cfc;
  }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  th,
  td {
    text-align: left;
    padding: 10px 6px;
    border-bottom: 1px solid rgba(15, 23, 42, 0.06);
  }

  th {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
    font-weight: 600;
  }

  .signal-type {
    text-transform: capitalize;
    font-weight: 500;
  }

  .edit-input {
    width: 80px;
    padding: 8px 10px;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    background: #f8fafc;
  }

  .edit-input:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.12);
    background: #fff;
  }

  .action-buttons {
    display: flex;
    gap: 8px;
  }

  .btn-small {
    padding: 6px 12px;
    font-size: 13px;
    border-radius: 6px;
    border: 1px solid rgba(148, 163, 184, 0.4);
    background: #fff;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .btn-small.btn-primary {
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: white;
    border: none;
  }

  .btn-small:hover:not(:disabled) {
    border-color: #cbd5e1;
  }

  .btn-small.btn-primary:hover:not(:disabled) {
    box-shadow: 0 4px 10px rgba(99, 102, 241, 0.3);
  }

  .cleanup-form {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .form-group {
    border: none;
    padding: 0;
    margin: 0;
  }

  .form-group legend,
  .form-group > label {
    display: block;
    font-size: 12px;
    text-transform: uppercase;
    font-weight: 600;
    margin-bottom: 8px;
    color: #64748b;
    letter-spacing: 0.05em;
  }

  .checkbox-group {
    display: flex;
    gap: 16px;
  }

  .checkbox-group label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 14px;
    text-transform: none;
    font-weight: 400;
  }

  select {
    width: 100%;
    padding: 12px 14px;
    border-radius: 10px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    font-size: 14px;
    color: #0f172a;
    transition: all 0.2s ease;
  }

  select:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.12);
    background: #fff;
  }

  .hint {
    font-size: 12px;
    color: #64748b;
    margin: 4px 0 0 0;
  }

  .btn-danger {
    padding: 12px 20px;
    border-radius: 10px;
    border: none;
    background: #dc2626;
    color: white;
    font-weight: 600;
    cursor: pointer;
    align-self: flex-start;
    transition: all 0.2s ease;
  }

  .btn-danger:hover {
    background: #b91c1c;
  }

  .btn-danger:disabled {
    background: #9ca3af;
    cursor: not-allowed;
  }

  .status {
    color: #6b6f76;
    padding: 20px;
  }

  .monospace {
    font-family: 'Courier New', monospace;
    font-size: 13px;
  }

  .status-badge {
    padding: 4px 8px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
  }

  .status-badge.completed {
    background: #d1fae5;
    color: #065f46;
  }

  .status-badge.running {
    background: #dbeafe;
    color: #1e40af;
  }

  .status-badge.failed {
    background: #fee2e2;
    color: #991b1b;
  }
</style>
