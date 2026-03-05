<script lang="ts">
  import { onMount } from "svelte";
  import ConfirmModal from "./common/ConfirmModal.svelte";
  import {
    getRetentionSettings,
    updateRetentionSetting,
    executeCleanup,
    listCleanupJobs,
    type RetentionSetting,
    type SignalType,
    type CleanupJob,
  } from "../services/retention";
  import { fetchServices } from "../services/services";

  let settings: RetentionSetting[] = [];
  let jobs: CleanupJob[] = [];
  let services: string[] = [];
  let loading = false;
  let error = "";
  let successMessage = "";

  let editingSignal: SignalType | null = null;
  let editRetentionDays = 7;

  let selectedSignals: SignalType[] = [];
  let selectedService = "";
  let cleanupLoading = false;
  let confirmOpen = false;
  let confirmMessage = "";
  let showAllJobs = false;
  let totalJobs = 0;
  const signalLabels: Record<SignalType, string> = {
    logs: "Log",
    traces: "Tracce",
    metrics: "Metriche",
  };
  const statusLabels: Record<string, string> = {
    completed: "Completato",
    running: "In corso",
    failed: "Fallito",
  };
  const MAX_RETENTION_DAYS = 365;
  const MAX_TRACES_RETENTION_DAYS = 15;

  function labelForSignal(signal: SignalType) {
    return signalLabels[signal] ?? signal;
  }

  function labelList(signals: SignalType[]) {
    return signals.map(labelForSignal).join(", ");
  }

  function labelForStatus(status: string) {
    return statusLabels[status] ?? status;
  }

  function maxRetentionForSignal(signal: SignalType) {
    return signal === "traces" ? MAX_TRACES_RETENTION_DAYS : MAX_RETENTION_DAYS;
  }

  async function loadSettings() {
    loading = true;
    error = "";
    try {
      const result = await getRetentionSettings();
      settings = result.settings;
    } catch (err) {
      error =
        err instanceof Error
          ? err.message
          : "Impossibile caricare le impostazioni";
    } finally {
      loading = false;
    }
  }

  async function loadJobs(all = false) {
    try {
      const result = await listCleanupJobs(all ? undefined : 10, all);
      jobs = result.jobs;
      totalJobs = result.total ?? result.jobs.length;
    } catch (err) {
    }
  }

  async function loadServices() {
    try {
      const result = await fetchServices();
      services = result.services;
    } catch (err) {
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

    const maxRetentionDays = maxRetentionForSignal(editingSignal);
    if (editRetentionDays < 1 || editRetentionDays > maxRetentionDays) {
      error = `La conservazione per ${labelForSignal(editingSignal)} deve essere tra 1 e ${maxRetentionDays} giorni`;
      return;
    }

    loading = true;
    error = "";
    successMessage = "";

    try {
      await updateRetentionSetting(editingSignal, editRetentionDays);
      successMessage = `Conservazione per ${labelForSignal(editingSignal)} aggiornata a ${editRetentionDays} giorni`;
      editingSignal = null;
      await loadSettings();
    } catch (err) {
      error =
        err instanceof Error
          ? err.message
          : "Impossibile aggiornare la conservazione";
    } finally {
      loading = false;
    }
  }

  async function runCleanup() {
    if (selectedSignals.length === 0) {
      error = "Seleziona almeno un tipo di segnale";
      return;
    }

    confirmMessage = selectedService
      ? `Eliminare tutti i dati ${labelList(selectedSignals)} per il servizio "${selectedService}"?`
      : `Eliminare TUTTI i dati ${labelList(selectedSignals)}? Questa azione non può essere annullata!`;

    confirmOpen = true;
  }

  async function confirmCleanup() {
    confirmOpen = false;

    cleanupLoading = true;
    error = "";
    successMessage = "";

    try {
      const result = await executeCleanup({
        signalTypes: selectedSignals,
        serviceName: selectedService || undefined,
      });

      const totalDeleted = result.results.reduce(
        (sum, r) => sum + r.recordsDeleted,
        0,
      );
      successMessage = `Pulizia completata! ${totalDeleted} record eliminati (ID operazione: ${result.jobId})`;

      selectedSignals = [];
      selectedService = "";
      await loadJobs(showAllJobs);
    } catch (err) {
      error =
        err instanceof Error ? err.message : "Impossibile eseguire la pulizia";
    } finally {
      cleanupLoading = false;
    }
  }

  function cancelCleanup() {
    confirmOpen = false;
  }

  async function toggleJobsView() {
    showAllJobs = !showAllJobs;
    await loadJobs(showAllJobs);
  }

  function toggleSignal(signal: SignalType) {
    if (selectedSignals.includes(signal)) {
      selectedSignals = selectedSignals.filter((s) => s !== signal);
    } else {
      selectedSignals = [...selectedSignals, signal];
    }
  }

  onMount(() => {
    void loadSettings();
    void loadJobs(false);
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

  <div class="panels">
    <div class="panel">
      <h3>Impostazioni conservazione</h3>
      <p class="help-text">
        I dati piu vecchi del periodo di conservazione verranno eliminati
        automaticamente ogni 24 ore.
        Le tracce hanno un massimo di 15 giorni.
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
                <td class="signal-type">{labelForSignal(setting.signalType)}</td
                >
                <td>
                  {#if editingSignal === setting.signalType}
                    <input
                      type="number"
                      bind:value={editRetentionDays}
                      min="1"
                      max={maxRetentionForSignal(setting.signalType)}
                      class="edit-input"
                    />
                  {:else}
                    {setting.retentionDays} giorni
                  {/if}
                </td>
                <td>
                  {#if editingSignal === setting.signalType}
                    <div class="action-buttons">
                      <button class="btn-small btn-primary" on:click={saveEdit}
                        >Salva</button
                      >
                      <button class="btn-small" on:click={cancelEdit}
                        >Annulla</button
                      >
                    </div>
                  {:else}
                    <button
                      class="btn-small"
                      on:click={() => startEdit(setting)}>Modifica</button
                    >
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
                checked={selectedSignals.includes("logs")}
                on:change={() => toggleSignal("logs")}
              />
              {labelForSignal("logs")}
            </label>
            <label>
              <input
                type="checkbox"
                checked={selectedSignals.includes("traces")}
                on:change={() => toggleSignal("traces")}
              />
              {labelForSignal("traces")}
            </label>
            <label>
              <input
                type="checkbox"
                checked={selectedSignals.includes("metrics")}
                on:change={() => toggleSignal("metrics")}
              />
              {labelForSignal("metrics")}
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
          {cleanupLoading ? "Pulizia in corso..." : "Esegui pulizia"}
        </button>
      </div>
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
              <td>{job.serviceName || "tutti"}</td>
              <td>
                <span class="status-badge {job.status}">
                  {labelForStatus(job.status)}
                </span>
              </td>
              <td>{job.recordsDeleted.toLocaleString()}</td>
              <td
                >{job.completedAt
                  ? new Date(job.completedAt).toLocaleString("it-IT")
                  : "-"}</td
              >
            </tr>
          {/each}
        </tbody>
      </table>
      {#if totalJobs > 10 || showAllJobs}
        <button
          class="btn-small btn-outline toggle-jobs"
          on:click={toggleJobsView}
        >
          {showAllJobs ? "Mostra ultime 10" : "Mostra tutte"}
        </button>
      {/if}
    {/if}
  </div>

  <ConfirmModal
    open={confirmOpen}
    title="Conferma pulizia dati"
    message={confirmMessage}
    confirmLabel="Conferma"
    cancelLabel="Annulla"
    variant="danger"
    on:confirm={confirmCleanup}
    on:cancel={cancelCleanup}
  />
</section>

<style>
  .retention-management {
    display: grid;
    gap: 20px;
  }

  .panels {
    display: grid;
    gap: 20px;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
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

  .btn-small.btn-outline {
    background: #f8fafc;
    border-color: #cbd5e1;
    color: #475569;
  }

  .toggle-jobs {
    margin-top: 12px;
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
    padding: 12px 24px;
    border-radius: 10px;
    border: none;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: white;
    font-weight: 600;
    cursor: pointer;
    align-self: flex-start;
    transition: all 0.2s ease;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
  }

  .btn-danger:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(99, 102, 241, 0.4);
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
    font-family: "Courier New", monospace;
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
