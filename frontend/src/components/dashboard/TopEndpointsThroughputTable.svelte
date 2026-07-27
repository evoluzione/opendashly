<script lang="ts">
  import type { EndpointThroughput } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';
  import MarqueeText from '../common/MarqueeText.svelte';
  import { locale, t, getLocaleTag } from '../../lib/i18n';

  export let data: EndpointThroughput[] = [];
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      Top Endpoint per Throughput
      <InfoTooltip text="Endpoint ordinati per richieste totali. Utile per capire dove si concentra il traffico." />
    </span>
    <span class="table-subtitle">{t($locale, 'dashboard.topEndpoints.subtitle')}</span>
  </div>

  {#if data.length === 0}
    <div class="empty">{t($locale, 'dashboard.noData')}</div>
  {:else}
    <div class="table-wrapper">
      <table>
        <thead>
          <tr>
            <th class="col-endpoint">{t($locale, 'dashboard.table.endpoint')}</th>
            <th class="col-service">{t($locale, 'dashboard.table.service')}</th>
            <th class="col-count">{t($locale, 'dashboard.table.requests')}</th>
            <th class="col-count">{t($locale, 'dashboard.table.errors')}</th>
            <th class="col-rate">{t($locale, 'dashboard.table.errorRate')}</th>
          </tr>
        </thead>
        <tbody>
          {#each data as row}
            <tr>
              <td class="col-endpoint">
                <MarqueeText text={row.endpoint} class="endpoint-name" />
              </td>
              <td class="col-service">
                <span class="service-badge">{row.service}</span>
              </td>
              <td class="col-count">{row.requestCount.toLocaleString(getLocaleTag($locale))}</td>
              <td class="col-count error-count">{row.errorCount.toLocaleString(getLocaleTag($locale))}</td>
              <td class="col-rate">{row.errorRate.toFixed(1)}%</td>
            </tr>
          {/each}
        </tbody>
      </table>
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

  .table-wrapper {
    overflow: auto;
    flex: 1;
    min-height: 0;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
    table-layout: fixed;
  }

  th {
    text-align: left;
    padding: 10px 12px;
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-slate-500);
    background: var(--color-slate-50);
    border-bottom: 1px solid var(--color-slate-200);
  }

  td {
    padding: 12px;
    border-bottom: 1px solid var(--color-slate-100);
    color: var(--color-slate-950);
  }

  tr:last-child td {
    border-bottom: none;
  }

  tr:hover td {
    background: var(--color-slate-50);
  }

  .col-endpoint {
    width: 40%;
    max-width: 0;
  }

  :global(.endpoint-name) {
    font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
    font-size: 12px;
  }

  .col-service {
    min-width: 100px;
  }

  .service-badge {
    display: inline-block;
    max-width: 100%;
    padding: 3px 8px;
    background: var(--color-slate-100);
    border-radius: 4px;
    font-size: 11px;
    font-weight: 500;
    color: var(--color-slate-600);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .col-count {
    text-align: right;
    width: 12%;
    font-variant-numeric: tabular-nums;
    color: var(--color-slate-500);
    font-size: 11px;
  }

  .col-count.error-count {
    color: var(--color-danger-500);
    font-weight: 600;
  }

  .col-rate {
    width: 14%;
    text-align: right;
    font-size: 11px;
    font-weight: 600;
  }
</style>
