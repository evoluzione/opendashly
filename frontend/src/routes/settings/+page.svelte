<script lang="ts">
  import LanguageSwitcher from '../../components/common/LanguageSwitcher.svelte';
  import { locale, t } from '$lib/i18n';
</script>

<div class="settings-page">
  <header class="page-header">
    <h1>{t($locale, 'settings.title')}</h1>
    <p class="subtitle">{t($locale, 'settings.subtitle')}</p>
  </header>

  <section class="language-section">
    <div class="language-head">
      <h3>{t($locale, 'settings.languageTitle')}</h3>
      <LanguageSwitcher />
    </div>
    <p class="hint">{t($locale, 'settings.languageDescription')}</p>
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

  .language-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .language-head {
    display: flex;
    align-items: center;
    gap: 12px;
    justify-content: flex-end;
  }

  .language-head h3 {
    margin-right: auto;
  }

  h3 {
    margin: 0;
    font-size: 15px;
    color: var(--color-slate-950);
  }

  .hint {
    margin: 0;
    color: var(--color-slate-500);
    font-size: 14px;
    line-height: 1.45;
    max-width: 620px;
  }

</style>
