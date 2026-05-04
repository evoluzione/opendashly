<script lang="ts">
  import { onMount } from "svelte";
  import { fetchRelated } from "../services/traces";
  import LogResultsTable from "./LogResultsTable.svelte";
  import { locale, t } from "../lib/i18n";

  export let traceId: string;
  let related: { logs: any[] } | null = null;
  let error: string | null = null;

  onMount(async () => {
    try {
      related = await fetchRelated(traceId);
    } catch (err) {
      error =
        err instanceof Error
          ? err.message
          : t($locale, "correlation.loadError");
    }
  });
</script>

<section class="panel">
  <header>
    <div>
      <h3>{t($locale, "correlation.title")}</h3>
      <p>{t($locale, "correlation.subtitle")}</p>
    </div>
  </header>
  {#if error}
    <p class="error">{error}</p>
  {:else if !related}
    <p class="loading">{t($locale, "correlation.loading")}</p>
  {:else}
    <div class="grid">
      <div class="block">
        <h4>{t($locale, "correlation.logs")}</h4>
        <LogResultsTable logs={related.logs} inlineDetails={true} />
      </div>
    </div>
  {/if}
</section>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
    background: var(--color-slate-50);
    border-radius: 12px;
    padding: 12px;
    border: 1px solid rgba(148, 163, 184, 0.2);
    min-height: 0;
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  h3 {
    margin: 0 0 4px 0;
    font-size: 14px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--color-slate-500);
  }

  p {
    margin: 0;
    color: var(--color-slate-400);
    font-size: 12px;
  }

  .grid {
    display: grid;
    gap: 20px;
    grid-template-rows: minmax(0, 1fr);
    flex: 1;
    min-height: 0;
  }

  .block {
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-height: 0;
    min-width: 0;
  }

  .block h4 {
    margin: 0;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--color-slate-400);
    font-weight: 600;
  }

  .error {
    color: var(--color-danger-700);
    background: var(--color-danger-100);
    padding: 12px 16px;
    border-radius: 10px;
  }

  .loading {
    color: var(--color-slate-400);
    font-size: 14px;
  }
</style>
