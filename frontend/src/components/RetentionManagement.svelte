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
  import { getLocaleTag, locale, t } from "../lib/i18n";

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
    traces: "Trace",
  };
  const statusLabels: Record<string, string> = {
    completed: "retention.statusCompleted",
    running: "retention.statusRunning",
    failed: "retention.statusFailed",
  };
  const MAX_RETENTION_DAYS = 365;

  function labelForSignal(signal: SignalType) {
    const base = signalLabels[signal] ?? signal;
    if (base === "Log") return t($locale, "sidebar.logs");
    if (base === "Trace") return t($locale, "sidebar.traces");
    return base;
  }

  function labelList(signals: SignalType[]) {
    return signals.map(labelForSignal).join(", ");
  }

  function labelForStatus(status: string) {
    const key = statusLabels[status];
    return key ? t($locale, key) : status;
  }

  function maxRetentionForSignal(signal: SignalType) {
    const found = settings.find((s) => s.signalType === signal);
    if (found?.maxRetentionDays && found.maxRetentionDays > 0) {
      return found.maxRetentionDays;
    }
    return MAX_RETENTION_DAYS;
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
          : t($locale, "retention.loadError");
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
      error = t($locale, "retention.rangeError", {
        signal: labelForSignal(editingSignal),
        max: maxRetentionDays,
      });
      return;
    }

    loading = true;
    error = "";
    successMessage = "";

    try {
      await updateRetentionSetting(editingSignal, editRetentionDays);
      successMessage = t($locale, "retention.updated", {
        signal: labelForSignal(editingSignal),
        days: editRetentionDays,
      });
      editingSignal = null;
      await loadSettings();
    } catch (err) {
      error =
        err instanceof Error
          ? err.message
          : t($locale, "retention.updateError");
    } finally {
      loading = false;
    }
  }

  async function runCleanup() {
    if (selectedSignals.length === 0) {
      error = t($locale, "retention.signalRequired");
      return;
    }

    confirmMessage = selectedService
      ? t($locale, "retention.confirmService", {
          signals: labelList(selectedSignals),
          service: selectedService,
        })
      : t($locale, "retention.confirmAll", {
          signals: labelList(selectedSignals),
        });

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
      successMessage = t($locale, "retention.cleanupDone", {
        total: totalDeleted,
        jobId: result.jobId,
      });

      selectedSignals = [];
      selectedService = "";
      await loadJobs(showAllJobs);
    } catch (err) {
      error =
        err instanceof Error ? err.message : t($locale, "retention.cleanupError");
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
    <h2>{t($locale, "retention.title")}</h2>
    <p>{t($locale, "retention.subtitle")}</p>
  </header>

  {#if error}
    <div class="alert error">{error}</div>
  {/if}
  {#if successMessage}
    <div class="alert success">{successMessage}</div>
  {/if}

  <div class="panels">
    <div class="panel">
      <h3>{t($locale, "retention.settingsTitle")}</h3>
      <p class="help-text">
        {t($locale, "retention.settingsHelp", { traceMax: maxRetentionForSignal("traces") })}
      </p>

      {#if loading && settings.length === 0}
        <div class="status">{t($locale, "common.loading")}</div>
      {:else}
        <table>
          <thead>
            <tr>
              <th>{t($locale, "retention.signalType")}</th>
              <th>{t($locale, "retention.retentionDays")}</th>
              <th>{t($locale, "retention.actions")}</th>
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
                    {t($locale, "retention.daysSuffix", { days: setting.retentionDays })}
                  {/if}
                </td>
                <td>
                  {#if editingSignal === setting.signalType}
                    <div class="action-buttons">
                      <button class="btn-small btn-primary" on:click={saveEdit}
                        >{t($locale, "retention.save")}</button
                      >
                      <button class="btn-small" on:click={cancelEdit}
                        >{t($locale, "common.cancel")}</button
                      >
                    </div>
                  {:else}
                    <button
                      class="btn-small"
                      on:click={() => startEdit(setting)}>{t($locale, "retention.edit")}</button
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
      <h3>{t($locale, "retention.manualTitle")}</h3>
      <p class="help-text">
        {t($locale, "retention.manualHelp")}
      </p>

      <div class="cleanup-form">
        <fieldset class="form-group">
          <legend>{t($locale, "retention.signalTypes")}</legend>
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
          </div>
        </fieldset>

        <div class="form-group">
          <label for="service-select">{t($locale, "retention.serviceOptional")}</label>
          <select id="service-select" bind:value={selectedService}>
            <option value="">{t($locale, "retention.allServices")}</option>
            {#each services as service}
              <option value={service}>{service}</option>
            {/each}
          </select>
          <p class="hint">{t($locale, "retention.serviceHint")}</p>
        </div>

        <button
          class="btn-danger"
          on:click={runCleanup}
          disabled={cleanupLoading || selectedSignals.length === 0}
        >
          {cleanupLoading ? t($locale, "retention.cleanupRunning") : t($locale, "retention.runCleanup")}
        </button>
      </div>
    </div>
  </div>

  <div class="panel">
    <h3>{t($locale, "retention.historyTitle")}</h3>
    {#if jobs.length === 0}
      <div class="status">{t($locale, "retention.noHistory")}</div>
    {:else}
      <table>
        <thead>
          <tr>
            <th>{t($locale, "retention.jobId")}</th>
            <th>{t($locale, "retention.type")}</th>
            <th>{t($locale, "retention.signal")}</th>
            <th>{t($locale, "retention.service")}</th>
            <th>{t($locale, "retention.state")}</th>
            <th>{t($locale, "retention.deletedRecords")}</th>
            <th>{t($locale, "retention.completedAt")}</th>
          </tr>
        </thead>
        <tbody>
          {#each jobs as job}
            <tr>
              <td class="monospace">{job.jobId.slice(0, 8)}</td>
              <td>{job.jobType}</td>
              <td>{job.signalType}</td>
              <td>{job.serviceName || t($locale, "retention.all")}</td>
              <td>
                <span class="status-badge {job.status}">
                  {labelForStatus(job.status)}
                </span>
              </td>
              <td>{job.recordsDeleted.toLocaleString(getLocaleTag($locale))}</td>
              <td
                >{job.completedAt
                  ? new Date(job.completedAt).toLocaleString(getLocaleTag($locale))
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
          {showAllJobs ? t($locale, "retention.showLatest10") : t($locale, "retention.showAll")}
        </button>
      {/if}
    {/if}
  </div>

  <ConfirmModal
    open={confirmOpen}
    title={t($locale, "retention.confirmTitle")}
    message={confirmMessage}
    confirmLabel={t($locale, "retention.confirm")}
    cancelLabel={t($locale, "common.cancel")}
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
    color: var(--color-slate-950);
  }
  header p {
    margin: 0;
    color: var(--color-slate-500);
    font-size: 14px;
  }

  .panel {
    background: var(--color-white);
    border-radius: 16px;
    padding: 20px;
    border: 1px solid rgba(var(--rgb-slate-950), 0.06);
    box-shadow: 0 1px 3px rgba(var(--rgb-slate-950), 0.08);
  }

  .panel h3 {
    margin: 0 0 12px 0;
    font-size: 16px;
    font-weight: 600;
  }

  .help-text {
    color: var(--color-slate-500);
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
    border-bottom: 1px solid rgba(var(--rgb-slate-950), 0.06);
  }

  th {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-slate-500);
    font-weight: 600;
  }

  .signal-type {
    text-transform: capitalize;
    font-weight: 500;
  }

  .edit-input {
    width: 80px;
    padding: 8px 10px;
    border: 1px solid var(--color-slate-200);
    border-radius: 8px;
    background: var(--color-slate-50);
  }

  .edit-input:focus {
    outline: none;
    border-color: var(--color-primary-600);
    box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.12);
    background: var(--color-white);
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
    background: var(--color-white);
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .btn-small.btn-primary {
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
    color: white;
    border: none;
  }

  .btn-small:hover:not(:disabled) {
    border-color: var(--color-slate-300);
  }

  .btn-small.btn-primary:hover:not(:disabled) {
    box-shadow: 0 4px 10px rgba(var(--rgb-primary-600), 0.3);
  }

  .btn-small.btn-outline {
    background: var(--color-slate-50);
    border-color: var(--color-slate-300);
    color: var(--color-slate-600);
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
    color: var(--color-slate-500);
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
    border: 1px solid var(--color-slate-200);
    background: var(--color-slate-50);
    font-size: 14px;
    color: var(--color-slate-950);
    transition: all 0.2s ease;
  }

  select:focus {
    outline: none;
    border-color: var(--color-primary-600);
    box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.12);
    background: var(--color-white);
  }

  .hint {
    font-size: 12px;
    color: var(--color-slate-500);
    margin: 4px 0 0 0;
  }

  .btn-danger {
    padding: 12px 24px;
    border-radius: 10px;
    border: none;
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
    color: white;
    font-weight: 600;
    cursor: pointer;
    align-self: flex-start;
    transition: all 0.2s ease;
    box-shadow: 0 4px 12px rgba(var(--rgb-primary-600), 0.3);
  }

  .btn-danger:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(var(--rgb-primary-600), 0.4);
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
    background: var(--color-info-100);
    color: #1e40af;
  }

  .status-badge.failed {
    background: var(--color-danger-100);
    color: #991b1b;
  }
</style>
