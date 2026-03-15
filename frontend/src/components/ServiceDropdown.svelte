<script lang="ts">
  import { onMount } from 'svelte';
  import { servicesState, loadServices, selectService } from '../lib/stores/query';
  import { locale, t } from '../lib/i18n';

  onMount(() => {
    void loadServices();
  });

  function handleChange(event: Event) {
    const value = (event.target as HTMLSelectElement).value;
    selectService(value);
  }
</script>

<div class="service-dropdown">
  <label for="service-select">{t($locale, 'serviceDropdown.label')}</label>
  <select id="service-select" on:change={handleChange}>
    <option value="Tutti">{t($locale, 'serviceDropdown.all')}</option>
    {#each $servicesState.services as service}
      <option value={service} selected={service === $servicesState.selectedService}>{service}</option>
    {/each}
  </select>
  {#if $servicesState.loading}
    <span class="status">{t($locale, 'common.loading')}</span>
  {:else if $servicesState.error}
    <span class="status error">{$servicesState.error}</span>
  {:else if $servicesState.services.length === 0}
    <span class="status">{t($locale, 'serviceDropdown.none')}</span>
  {/if}
</div>

<style>
  .service-dropdown {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  
  label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-slate-500);
  }
  
  select {
    padding: 12px 16px;
    border-radius: 10px;
    border: 1px solid var(--color-slate-200);
    background: white;
    font-size: 14px;
    font-weight: 500;
    color: var(--color-slate-950);
    cursor: pointer;
    transition: all 0.2s ease;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%2364748b' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 12px center;
    padding-right: 40px;
  }
  
  select:hover {
    border-color: var(--color-slate-300);
  }
  
  select:focus {
    outline: none;
    border-color: var(--color-primary-600);
    box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.1);
  }
  
  .status {
    font-size: 12px;
    color: var(--color-slate-400);
    display: flex;
    align-items: center;
    gap: 6px;
  }
  
  .status.error {
    color: var(--color-danger-500);
  }
</style>
