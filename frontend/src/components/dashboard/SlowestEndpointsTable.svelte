<script lang="ts">
  import type { EndpointLatency } from "../../services/dashboard";
  import InfoTooltip from "../common/InfoTooltip.svelte";
  import { locale, t, getLocaleTag } from '../../lib/i18n';

  export let data: EndpointLatency[] = [];

  function formatMs(ms: number): string {
    if (ms >= 1000) return (ms / 1000).toFixed(2) + "s";
    return ms.toFixed(0) + "ms";
  }

  function getLatencyColor(ms: number): string {
    if (ms <= 500) return "#22c55e";
    if (ms <= 2000) return "#eab308";
    if (ms <= 5000) return "#f97316";
    return "#ef4444";
  }
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      {t($locale, 'dashboard.slowestEndpoints.title')}
      <InfoTooltip
        text={t($locale, 'dashboard.slowestEndpoints.tooltip')}
      />
    </span>
    <span class="table-subtitle">{t($locale, 'dashboard.slowestEndpoints.subtitle')}</span>
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
            <th class="col-latency">P50</th>
            <th class="col-latency">P95</th>
            <th class="col-latency">P99</th>
            <th class="col-count">{t($locale, 'dashboard.table.count')}</th>
          </tr>
        </thead>
        <tbody>
          {#each data as row}
            <tr>
              <td class="col-endpoint">
                <span class="endpoint-name" title={row.endpoint}
                  >{row.endpoint}</span
                >
              </td>
              <td class="col-service">
                <span class="service-badge">{row.service}</span>
              </td>
              <td class="col-latency">
                <span
                  class="latency-value"
                  style="color: {getLatencyColor(row.p50)}"
                >
                  {formatMs(row.p50)}
                </span>
              </td>
              <td class="col-latency">
                <span
                  class="latency-value"
                  style="color: {getLatencyColor(row.p95)}"
                >
                  {formatMs(row.p95)}
                </span>
              </td>
              <td class="col-latency">
                <span
                  class="latency-value"
                  style="color: {getLatencyColor(row.p99)}"
                >
                  {formatMs(row.p99)}
                </span>
              </td>
              <td class="col-count">{row.count.toLocaleString(getLocaleTag($locale))}</td>
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
    box-shadow:
      0 1px 3px rgba(0, 0, 0, 0.05),
      0 4px 12px rgba(0, 0, 0, 0.03);
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
    width: 30%;
    max-width: 0;
  }

  .endpoint-name {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: "SF Mono", Monaco, "Cascadia Code", monospace;
    font-size: 12px;
  }

  .col-service {
    min-width: 100px;
  }

  .service-badge {
    display: inline-block;
    max-width: 100%;
    padding: 3px 8px;
    background: #f1f5f9;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 500;
    color: #475569;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .col-latency {
    text-align: right;
    width: 12%;
    font-size: 11px;
  }

  .latency-value {
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    font-size: 11px;
  }

  .col-count {
    text-align: right;
    width: 10%;
    font-variant-numeric: tabular-nums;
    color: #64748b;
    font-size: 11px;
  }

  th.col-latency,
  th.col-count {
    text-align: right;
  }
</style>
