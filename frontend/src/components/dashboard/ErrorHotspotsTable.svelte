<script lang="ts">
  import type { ErrorHotspot } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';

  export let data: ErrorHotspot[] = [];

  function getErrorRateColor(rate: number): string {
    if (rate <= 1) return '#22c55e';
    if (rate <= 5) return '#eab308';
    if (rate <= 10) return '#f97316';
    return '#ef4444';
  }
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      Hotspot Errori
      <InfoTooltip text="Endpoint con il più alto tasso di errore. Mostra quali API generano più problemi e richiedono attenzione prioritaria." />
    </span>
    <span class="table-subtitle">Top 20 per error rate</span>
  </div>

  {#if data.length === 0}
    <div class="empty">Nessun errore rilevato</div>
  {:else}
    <div class="table-wrapper">
      <table>
        <thead>
          <tr>
            <th class="col-endpoint">Endpoint</th>
            <th class="col-service">Servizio</th>
            <th class="col-count">Errori</th>
            <th class="col-count">Totale</th>
            <th class="col-rate">Error Rate</th>
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
              <td class="col-count error-count">{row.errorCount.toLocaleString()}</td>
              <td class="col-count">{row.totalCount.toLocaleString()}</td>
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

  .table-wrapper {
    overflow-x: auto;
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
    color: #64748b;
    background: #f8fafc;
    border-bottom: 1px solid #e2e8f0;
  }

  td {
    padding: 12px;
    border-bottom: 1px solid #f1f5f9;
    color: #0f172a;
  }

  tr:last-child td {
    border-bottom: none;
  }

  tr:hover td {
    background: #f8fafc;
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
    padding: 3px 8px;
    background: #f1f5f9;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 500;
    color: #475569;
  }

  .col-count {
    text-align: right;
    width: 12%;
    font-variant-numeric: tabular-nums;
    color: #64748b;
    font-size: 11px;
  }

  .col-count.error-count {
    color: #ef4444;
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
    background: #f1f5f9;
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
