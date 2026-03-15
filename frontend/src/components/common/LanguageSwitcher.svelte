<script lang="ts">
  import { locale, setLocale, supportedLocales, t, type Locale } from '$lib/i18n';

  export let compact = false;

  const localeLabels: Partial<Record<Locale, string>> = {
    en: 'English',
    it: 'Italiano'
  };

  function isLocale(value: string): value is Locale {
    return supportedLocales.includes(value as Locale);
  }

  function select(value: string) {
    if (!isLocale(value)) return;
    setLocale(value);
  }

  function handleChange(event: Event) {
    const value = (event.currentTarget as HTMLSelectElement).value;
    select(value);
  }

  function optionLabel(option: Locale): string {
    const label = localeLabels[option];
    return label ?? option.toUpperCase();
  }
</script>

<select
  id="language-select"
  class="language-switcher"
  class:compact={compact}
  value={$locale}
  on:change={handleChange}
  aria-label={t($locale, 'language.label')}
  title={t($locale, 'language.label')}
>
  {#each supportedLocales as option}
    <option value={option}>{optionLabel(option)}</option>
  {/each}
</select>

<style>
  .language-switcher {
    min-width: 140px;
    border: 1px solid var(--color-slate-300);
    border-radius: 8px;
    background: var(--color-white);
    color: var(--color-slate-950);
    font-size: 14px;
    font-weight: 500;
    line-height: 1.2;
    padding: 8px 10px;
    appearance: auto;
    cursor: pointer;
    transition: border-color 0.2s ease, box-shadow 0.2s ease;
  }

  .language-switcher:hover {
    border-color: var(--color-slate-400);
  }

  .language-switcher:focus {
    outline: none;
    border-color: var(--color-info-600);
    box-shadow: none;
  }

  .language-switcher.compact {
    min-width: 120px;
    background: rgba(var(--rgb-slate-950), 0.35);
    border-color: rgba(148, 163, 184, 0.35);
    color: var(--color-slate-50);
  }

  .language-switcher.compact:hover {
    background: rgba(var(--rgb-slate-950), 0.55);
    border-color: rgba(148, 163, 184, 0.55);
  }

  .language-switcher.compact:focus {
    box-shadow: none;
    border-color: rgba(148, 163, 184, 0.7);
  }
</style>
