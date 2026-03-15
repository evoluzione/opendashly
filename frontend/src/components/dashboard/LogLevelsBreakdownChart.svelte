<script lang="ts">
  import type { LogLevelCount } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';
  import { locale, t, getLocaleTag } from '../../lib/i18n';

  export let data: LogLevelCount[] = [];

  const colors: Record<string, string> = {
    error: 'var(--color-danger-500)',
    warn: 'var(--color-warning-500)',
    warning: 'var(--color-warning-500)',
    info: 'var(--color-info-500)',
    debug: 'var(--color-primary-500)',
    trace: 'var(--color-cyan-500)',
    fatal: 'var(--color-danger-700)'
  };

  function colorFor(level: string): string {
    return colors[level.toLowerCase()] ?? 'var(--color-slate-400)';
  }
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      Livelli Log
      <InfoTooltip text={t($locale, 'dashboard.logLevels.tooltip')} />
    </span>
    <span class="table-subtitle">{t($locale, 'dashboard.logLevels.subtitle')}</span>
  </div>

  {#if data.length === 0}
    <div class="empty">{t($locale, 'dashboard.noData')}</div>
  {:else}
    <div class="breakdown">
      {#each data as row}
        <div class="breakdown-row">
          <div class="breakdown-label">
            <span class="dot" style="background: {colorFor(row.level)}"></span>
            <span>{row.level.toUpperCase()}</span>
          </div>
          <div class="breakdown-bar">
            <span class="bar" style="width: {Math.min(row.percentage, 100)}%; background: {colorFor(row.level)}"></span>
          </div>
          <div class="breakdown-value">
            <span>{row.count.toLocaleString(getLocaleTag($locale))}</span>
            <span class="pct">{row.percentage.toFixed(1)}%</span>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .table-card {
    background: white;
    border-radius: 16px;
    padding: 20px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(var(--rgb-slate-950), 0.06);
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .table-header {
    margin-bottom: 16px;
  }

  .table-title {
    display: block;
    font-size: 14px;
    font-weight: 600;
    color: var(--color-slate-950);
  }

  .table-subtitle {
    display: block;
    font-size: 12px;
    color: var(--color-slate-400);
    margin-top: 2px;
  }

  .empty {
    text-align: center;
    color: var(--color-slate-400);
    padding: 40px 0;
    font-size: 14px;
  }

  .breakdown {
    display: flex;
    flex-direction: column;
    gap: 12px;
    flex: 1;
    min-height: 0;
  }

  .breakdown-row {
    display: grid;
    grid-template-columns: minmax(100px, 1fr) minmax(120px, 2fr) auto;
    align-items: center;
    gap: 12px;
  }

  .breakdown-label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: var(--color-slate-950);
  }

  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
  }

  .breakdown-bar {
    height: 8px;
    background: var(--color-slate-100);
    border-radius: 999px;
    overflow: hidden;
  }

  .bar {
    display: block;
    height: 100%;
    border-radius: 999px;
  }

  .breakdown-value {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    font-size: 12px;
    color: var(--color-slate-600);
  }

  .pct {
    font-weight: 600;
    color: var(--color-slate-950);
  }
</style>
