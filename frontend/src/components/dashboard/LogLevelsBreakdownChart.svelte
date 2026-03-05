<script lang="ts">
  import type { LogLevelCount } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';

  export let data: LogLevelCount[] = [];

  const colors: Record<string, string> = {
    error: '#ef4444',
    warn: '#f59e0b',
    warning: '#f59e0b',
    info: '#3b82f6',
    debug: '#8b5cf6',
    trace: '#06b6d4',
    fatal: '#b91c1c'
  };

  function colorFor(level: string): string {
    return colors[level.toLowerCase()] ?? '#94a3b8';
  }
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      Livelli Log
      <InfoTooltip text="Distribuzione dei log per livello (ERROR/WARN/INFO/DEBUG/TRACE)." />
    </span>
    <span class="table-subtitle">Qualita e severita del rumore applicativo</span>
  </div>

  {#if data.length === 0}
    <div class="empty">Nessun dato disponibile</div>
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
            <span>{row.count.toLocaleString()}</span>
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
    border: 1px solid rgba(15, 23, 42, 0.06);
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
    color: #0f172a;
  }

  .table-subtitle {
    display: block;
    font-size: 12px;
    color: #94a3b8;
    margin-top: 2px;
  }

  .empty {
    text-align: center;
    color: #94a3b8;
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
    color: #0f172a;
  }

  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
  }

  .breakdown-bar {
    height: 8px;
    background: #f1f5f9;
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
    color: #475569;
  }

  .pct {
    font-weight: 600;
    color: #0f172a;
  }
</style>
