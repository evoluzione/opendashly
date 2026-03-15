<script lang="ts">
  import type { ErrorHotspot } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';
  import { locale, t, getLocaleTag } from '../../lib/i18n';

  export let data: ErrorHotspot[] = [];

  function getErrorRateColor(rate: number): string {
    if (rate <= 1) return 'var(--color-success-500)';
    if (rate <= 5) return '#eab308';
    if (rate <= 10) return '#f97316';
    return 'var(--color-danger-500)';
  }
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      {t($locale, 'dashboard.errorHotspots.title')}
      <InfoTooltip text={t($locale, 'dashboard.errorHotspots.tooltip')} />
    </span>
    <span class="table-subtitle">{t($locale, 'dashboard.errorHotspots.subtitle')}</span>
  </div>

  {#if data.length === 0}
    <div class="empty">{t($locale, 'dashboard.errorHotspots.none')}</div>
  {:else}
    <div class="table-wrapper">
      <table>
        <thead>
          <tr>
            <th class="col-endpoint">{t($locale, 'dashboard.table.endpoint')}</th>
            <th class="col-service">{t($locale, 'dashboard.table.service')}</th>
            <th class="col-count">{t($locale, 'dashboard.table.errors')}</th>
            <th class="col-count">{t($locale, 'dashboard.table.total')}</th>
            <th class="col-rate">{t($locale, 'dashboard.table.errorRate')}</th>
          </tr>
        </thead>
        <tbody>
          {#each data as row}
            <tr>
              <td class="col-endpoint">
                <span class="endpoint-name" title={row.endpoint}>{row.endpoint}</span>
              </td>
              <td class="col-service">
                <span class="service-badge">{row.service}</span>
              </td>
              <td class="col-count error-count">{row.errorCount.toLocaleString(getLocaleTag($locale))}</td>
              <td class="col-count">{row.totalCount.toLocaleString(getLocaleTag($locale))}</td>
              <td class="col-rate">
                <div class="rate-cell">
                  <div class="rate-bar-bg">
                    <div
                      class="rate-bar"
                      style="width: {Math.min(row.errorRate, 100)}%; background: {getErrorRateColor(row.errorRate)}"
                    ></div>
                  </div>
                  <span class="rate-value" style="color: {getErrorRateColor(row.errorRate)}">
                    {row.errorRate.toFixed(1)}%
                  </span>
                </div>
              </td>
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

  .endpoint-name {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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
    width: 20%;
  }

  .rate-cell {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .rate-bar-bg {
    flex: 1;
    height: 6px;
    background: var(--color-slate-100);
    border-radius: 3px;
    overflow: hidden;
  }

  .rate-bar {
    height: 100%;
    border-radius: 3px;
    transition: width 0.3s ease-out;
  }

  .rate-value {
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    min-width: 40px;
    text-align: right;
    font-size: 11px;
  }

  th.col-count,
  th.col-rate {
    text-align: right;
  }
</style>
