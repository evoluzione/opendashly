<script lang="ts">
  import { onMount } from 'svelte';
  import LanguageSwitcher from '../../components/common/LanguageSwitcher.svelte';
  import { locale, t } from '$lib/i18n';
  import { authState } from '../../lib/stores/auth';
  import { workspaceState, saveWorkspaceTitle } from '../../lib/stores/workspace';

  let titleInput = '';
  let saveSuccess = false;
  let saveError = '';

  $: isAdmin = $authState.user?.role === 'admin';

  onMount(() => {
    titleInput = $workspaceState.title;
  });

  async function handleSaveTitle() {
    saveSuccess = false;
    saveError = '';
    const ok = await saveWorkspaceTitle(titleInput.trim() || $workspaceState.title);
    if (ok) {
      saveSuccess = true;
      setTimeout(() => { saveSuccess = false; }, 3000);
    } else {
      saveError = $workspaceState.error ?? t($locale, 'settings.titleSaveError');
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter' && isAdmin) handleSaveTitle();
  }
</script>

<div class="settings-page">
  <header class="page-header">
    <h1>{t($locale, 'settings.title')}</h1>
    <p class="subtitle">{t($locale, 'settings.subtitle')}</p>
  </header>

  <section class="settings-section">
    <div class="setting-row">
      <div class="setting-label">
        <h3>{t($locale, 'settings.projectTitle')}</h3>
        <p class="hint">
          {#if isAdmin}
            {t($locale, 'settings.projectTitleHint')}
          {:else}
            {t($locale, 'settings.adminOnlyDescription')}
          {/if}
        </p>
      </div>
      <div class="setting-control">
        <input
          id="workspace-title"
          type="text"
          bind:value={titleInput}
          disabled={!isAdmin}
          placeholder={t($locale, 'settings.projectTitlePlaceholder')}
          on:keydown={handleKeydown}
        />
        {#if isAdmin}
          <button class="save-btn" on:click={handleSaveTitle} disabled={$workspaceState.loading}>
            {$workspaceState.loading ? t($locale, 'settings.saving') : t($locale, 'settings.save')}
          </button>
        {/if}
      </div>
    </div>
    {#if saveSuccess}
      <p class="feedback success">{t($locale, 'settings.titleSaveSuccess')}</p>
    {/if}
    {#if saveError}
      <p class="feedback error">{saveError}</p>
    {/if}
  </section>

  <section class="settings-section">
    <div class="setting-row">
      <div class="setting-label">
        <h3>{t($locale, 'settings.languageTitle')}</h3>
        <p class="hint">{t($locale, 'settings.languageDescription')}</p>
      </div>
      <div class="setting-control">
        <LanguageSwitcher />
      </div>
    </div>
  </section>
</div>

<style>
  .settings-page {
    width: 100%;
  }

  .page-header {
    margin-bottom: 20px;
  }

  h1 {
    margin: 0;
    font-size: 22px;
    font-weight: 700;
    letter-spacing: -0.02em;
    color: var(--color-slate-950);
  }

  .subtitle {
    margin: 6px 0 0;
    color: var(--color-slate-500);
    font-size: 14px;
  }

  .settings-section {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 24px;
  }

  .setting-row {
    display: flex;
    align-items: center;
    gap: 16px;
    justify-content: space-between;
  }

  .setting-label {
    flex: 1;
    min-width: 0;
  }

  .setting-control {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  h3 {
    margin: 0 0 4px;
    font-size: 15px;
    font-weight: 600;
    color: var(--color-slate-950);
  }

  .hint {
    margin: 0;
    color: var(--color-slate-500);
    font-size: 14px;
    line-height: 1.45;
  }

  input[type="text"] {
    width: 220px;
    padding: 8px 10px;
    border-radius: 8px;
    border: 1px solid var(--color-slate-300);
    background: var(--color-white);
    color: var(--color-slate-950);
    font-size: 14px;
    font-weight: 500;
    transition: border-color 0.2s ease, box-shadow 0.2s ease;
  }

  input[type="text"]:hover {
    border-color: var(--color-slate-400);
  }

  input[type="text"]:focus {
    outline: none;
    border-color: var(--color-info-600);
    box-shadow: none;
  }

  input[type="text"]:disabled {
    opacity: 0.55;
    cursor: not-allowed;
    background: var(--color-slate-50);
  }

  .save-btn {
    padding: 8px 16px;
    border-radius: 8px;
    border: none;
    background: var(--color-primary-600);
    color: var(--color-white);
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    white-space: nowrap;
    transition: opacity 0.2s;
  }

  .save-btn:hover:not(:disabled) {
    opacity: 0.88;
  }

  .save-btn:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .feedback {
    margin: 0;
    font-size: 13px;
    padding-left: 2px;
  }

  .feedback.success {
    color: var(--color-success-600, #16a34a);
  }

  .feedback.error {
    color: var(--color-danger-600);
  }
</style>
