<script lang="ts">
  import { onMount } from 'svelte';
  import { getAISettings, updateAISettings } from '../../../services/settings';
  import type { AISettings } from '../../../services/settings';

  let settings: AISettings | null = null;
  let loading = true;
  let error = '';
  let saving = false;
  let successMessage = '';

  // Form fields
  let enabled = false;
  let apiKey = '';
  let model = 'gpt-3.5-turbo';
  let provider = 'openai';

  const models = [
    { value: 'gpt-3.5-turbo', label: 'GPT-3.5 Turbo' },
    { value: 'gpt-4', label: 'GPT-4' },
    { value: 'gpt-4o', label: 'GPT-4o' }
  ];

  onMount(async () => {
    try {
      settings = await getAISettings();
      if (settings) {
        enabled = settings.enabled;
        model = settings.model || 'gpt-3.5-turbo';
        provider = settings.provider || 'openai';
        apiKey = settings.apiKey || ''; // usually masked
      }
    } catch (err) {
      error = 'Impossibile caricare le impostazioni AI.';
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
        apiKey
      });
      settings = updated;
      apiKey = updated.apiKey || '';
      successMessage = 'Impostazioni salvate con successo.';
    } catch (err) {
      error = 'Errore durante il salvataggio.';
    } finally {
      saving = false;
    }
  }
</script>

<div class="settings-page">
  <header>
    <h1>Impostazioni Intelligenza Artificiale</h1>
    <p class="subtitle">Configura l'agente AI per la ricerca smart e altre funzionalità.</p>
  </header>

  {#if loading}
    <div class="loading">Caricamento...</div>
  {:else}
    <form on:submit|preventDefault={save} class="settings-form">
      
      <div class="field-group">
        <label class="toggle-label">
          <input type="checkbox" bind:checked={enabled} />
          <span class="toggle-text">Abilita Smart Search</span>
        </label>
        <p class="helper">Attiva le funzionalità AI in tutta la piattaforma.</p>
      </div>

      <div class="field-group">
        <label for="provider">Provider</label>
        <select id="provider" bind:value={provider} disabled>
          <option value="openai">OpenAI</option>
        </select>
      </div>

      <div class="field-group">
        <label for="model">Modello</label>
        <select id="model" bind:value={model}>
          {#each models as m}
            <option value={m.value}>{m.label}</option>
          {/each}
        </select>
        <p class="helper">Seleziona il modello da utilizzare per la generazione delle query.</p>
      </div>

      <div class="field-group">
        <label for="apikey">API Key OpenAI</label>
        <input 
            type="password" 
            id="apikey" 
            bind:value={apiKey} 
            placeholder={settings?.apiKey ? '******' : 'sk-...'} 
        />
        <p class="helper">
            {#if settings?.apiKey && apiKey === settings.apiKey}
                Chiave salvata (mascherata). Modifica per aggiornare.
            {:else}
                Inserisci la tua chiave API di OpenAI.
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
          {saving ? 'Salvataggio...' : 'Salva Impostazioni'}
        </button>
      </div>
    </form>
  {/if}
</div>

<style>
  .settings-page {
    max-width: 800px;
    margin: 0 auto;
    padding: 24px;
  }

  header {
    margin-bottom: 32px;
  }

  h1 {
    font-size: 24px;
    font-weight: 700;
    color: #0f172a;
    margin: 0 0 8px 0;
  }

  .subtitle {
    color: #64748b;
    margin: 0;
  }

  .settings-form {
    background: white;
    padding: 32px;
    border-radius: 16px;
    border: 1px solid #e2e8f0;
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
    color: #334155;
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
      color: #0f172a;
  }

  input[type="text"],
  input[type="password"],
  select {
    padding: 10px 12px;
    border: 1px solid #cbd5e1;
    border-radius: 8px;
    font-size: 14px;
    color: #0f172a;
  }
  
  input:focus, select:focus {
      outline: none;
      border-color: #6366f1;
      box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
  }

  .helper {
    margin: 0;
    font-size: 13px;
    color: #94a3b8;
  }

  .actions {
    margin-top: 16px;
    display: flex;
    justify-content: flex-end;
  }

  button {
    background: #0f172a;
    color: white;
    padding: 12px 24px;
    border-radius: 8px;
    font-weight: 600;
    border: none;
    cursor: pointer;
    transition: all 0.2s;
  }

  button:hover:not(:disabled) {
    background: #334155;
  }

  button:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .alert {
    padding: 12px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
  }

  .alert.error {
    background: #fef2f2;
    color: #b91c1c;
    border: 1px solid #fecaca;
  }

  .alert.success {
    background: #f0fdf4;
    color: #15803d;
    border: 1px solid #bbf7d0;
  }
</style>
