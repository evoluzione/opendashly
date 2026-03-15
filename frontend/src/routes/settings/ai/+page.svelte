<script lang="ts">
  import { onMount } from 'svelte';
  import { getAISettings, updateAISettings } from '../../../services/settings';
  import type { AISettings } from '../../../services/settings';
  import { locale, t } from '$lib/i18n';

  let settings: AISettings | null = null;
  let loading = true;
  let error = '';
  let saving = false;
  let successMessage = '';

  // Form fields
  let enabled = false;
  let apiKey = '';
  let model = 'gpt-4o-mini';
  let provider = 'openai';

  const models = [
    { value: 'gpt-4.1', label: 'GPT-4.1 (Latest, Most Capable)', recommended: true },
    { value: 'gpt-4.1-mini', label: 'GPT-4.1 Mini (Fast & Cheap)' },
    { value: 'gpt-4.1-nano', label: 'GPT-4.1 Nano (Fastest)' },
    { value: 'gpt-4o', label: 'GPT-4o (Multimodal)' },
    { value: 'gpt-4o-mini', label: 'GPT-4o Mini (Good Balance)' },
    { value: 'o3-mini', label: 'o3-mini (Reasoning)' },
    { value: 'o1', label: 'o1 (Advanced Reasoning)' },
    { value: 'o1-mini', label: 'o1-mini (Fast Reasoning)' }
  ];

  onMount(async () => {
    try {
      settings = await getAISettings();
      if (settings) {
        enabled = settings.enabled;
        model = settings.model || 'gpt-4o-mini';
        provider = settings.provider || 'openai';
        apiKey = '';
      }
    } catch (err) {
      error = t($locale, 'aiSettings.loadError');
    } finally {
      loading = false;
    }
  });

  async function save() {
    saving = true;
    error = '';
    successMessage = '';
    try {
      const updated = await updateAISettings({
        enabled,
        provider,
        model,
        apiKey: apiKey.trim()
      });
      settings = updated;
      apiKey = '';
      successMessage = t($locale, 'aiSettings.saveSuccess');
    } catch (err) {
      error = t($locale, 'aiSettings.saveError');
    } finally {
      saving = false;
    }
  }
</script>

<div class="settings-page">
  <header>
    <h1>{t($locale, 'aiSettings.title')}</h1>
    <p class="subtitle">{t($locale, 'aiSettings.subtitle')}</p>
  </header>

  {#if loading}
    <div class="loading">{t($locale, 'common.loading')}</div>
  {:else}
    <form on:submit|preventDefault={save} class="settings-form">
      
      <div class="field-group">
        <label class="toggle-label">
          <input type="checkbox" bind:checked={enabled} />
          <span class="toggle-text">{t($locale, 'aiSettings.enableSmartSearch')}</span>
        </label>
        <p class="helper">{t($locale, 'aiSettings.enableSmartSearchHelp')}</p>
      </div>

      <div class="field-group">
        <label for="provider">{t($locale, 'aiSettings.provider')}</label>
        <select id="provider" bind:value={provider} disabled>
          <option value="openai">OpenAI</option>
        </select>
      </div>

      <div class="field-group">
        <label for="model">{t($locale, 'aiSettings.model')}</label>
        <select id="model" bind:value={model}>
          {#each models as m}
            <option value={m.value}>{m.label}</option>
          {/each}
        </select>
        <p class="helper">{t($locale, 'aiSettings.modelHelp')}</p>
      </div>

      <div class="field-group">
        <label for="apikey">{t($locale, 'aiSettings.apiKey')}</label>
        <input 
            type="password" 
            id="apikey" 
            bind:value={apiKey} 
            placeholder={settings?.apiKey ? '******' : 'sk-...'} 
        />
        <p class="helper">
            {#if settings?.apiKey && apiKey.trim() === ''}
            {t($locale, 'aiSettings.apiKeySavedHint')}
            {:else}
            {t($locale, 'aiSettings.apiKeyHelp')}
            {/if}
        </p>
      </div>

      {#if error}
        <div class="alert error">{error}</div>
      {/if}

      {#if successMessage}
        <div class="alert success">{successMessage}</div>
      {/if}

      <div class="actions">
        <button type="submit" disabled={saving}>
          {saving ? t($locale, 'aiSettings.saving') : t($locale, 'aiSettings.save')}
        </button>
      </div>
    </form>
  {/if}
</div>

<style>
  .settings-page {
    width: 100%;
  }

  header {
    margin-bottom: 20px;
  }

  h1 {
    font-size: 22px;
    font-weight: 700;
    color: var(--color-slate-950);
    margin: 0 0 8px 0;
  }

  .subtitle {
    color: var(--color-slate-500);
    margin: 6px 0 0;
    font-size: 14px;
  }

  .settings-form {
    background: white;
    padding: 32px;
    border-radius: 16px;
    border: 1px solid var(--color-slate-200);
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .field-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  label {
    font-size: 14px;
    font-weight: 600;
    color: var(--color-slate-700);
  }

  .toggle-label {
    display: flex;
    align-items: center;
    gap: 12px;
    cursor: pointer;
  }

  .toggle-label input {
    width: 20px;
    height: 20px;
  }
  
  .toggle-text {
      font-size: 16px;
      font-weight: 600;
      color: var(--color-slate-950);
  }

  input[type="password"],
  select {
    padding: 10px 12px;
    border: 1px solid var(--color-slate-300);
    border-radius: 8px;
    font-size: 14px;
    color: var(--color-slate-950);
  }
  
  input:focus, select:focus {
      outline: none;
      border-color: var(--color-primary-600);
      box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.1);
  }

  .helper {
    margin: 0;
    font-size: 13px;
    color: var(--color-slate-400);
  }

  .actions {
    margin-top: 16px;
    display: flex;
    justify-content: flex-end;
  }

  button {
    width: fit-content;
    height: fit-content;
    padding: 12px 24px;
    align-self: flex-end;
    justify-self: end;
    border-radius: 10px;
    border: none;
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
    color: var(--color-white);
    cursor: pointer;
    font-weight: 600;
    transition: all 0.2s ease;
    box-shadow: 0 4px 12px rgba(var(--rgb-primary-600), 0.3);
  }
  button:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(var(--rgb-primary-600), 0.4);
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-450) 100%);
  }
  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .alert {
    padding: 12px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
  }

  .alert.error {
    background: var(--color-danger-50);
    color: var(--color-danger-700);
    border: 1px solid var(--color-danger-75);
  }

  .alert.success {
    background: var(--color-success-50);
    color: var(--color-success-700);
    border: 1px solid var(--color-success-100);
  }
</style>
