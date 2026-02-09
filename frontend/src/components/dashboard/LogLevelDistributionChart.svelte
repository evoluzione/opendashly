<script lang="ts">
  import type { LogLevelCount } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';

  export let data: LogLevelCount[] = [];

  function colorFor(level: string) {
    switch (level) {
      case 'ERROR':
        return '#ef4444';
      case 'WARN':
      case 'WARNING':
        return '#f59e0b';
      case 'INFO':
        return '#0ea5e9';
      case 'DEBUG':
        return '#94a3b8';
      default:
        return '#64748b';
    }
  }
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      Distribuzione Log
      <InfoTooltip text="Ripartizione dei log per livello di severita." />
    </span>
    <span class="table-subtitle">Per livello di log</span>
  </div>

  {#if data.length === 0}
    <div class="empty">Nessun dato disponibile</div>
  {:else}
    <div class="levels">
      {#each data as row}
        <div class="level-row">
          <div class="level-label">{row.level || 'UNKNOWN'}</div>
          <div class="level-bar">
            <span class="bar" style="width: {Math.min(row.percentage, 100)}%; background: {colorFor(row.level)}"></span>
          </div>
          <div class="level-value">
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

  .levels {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .level-row {
    display: grid;
    grid-template-columns: minmax(80px, 1fr) minmax(120px, 2fr) auto;
    align-items: center;
    gap: 12px;
  }

  .level-label {
    font-size: 12px;
    font-weight: 600;
    color: #0f172a;
  }

  .level-bar {
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

  .level-value {
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
