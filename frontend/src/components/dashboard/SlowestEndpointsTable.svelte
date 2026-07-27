<script lang="ts">
  import type { EndpointLatency } from "../../services/dashboard";
  import InfoTooltip from "../common/InfoTooltip.svelte";
  import MarqueeText from "../common/MarqueeText.svelte";
  import { locale, t, getLocaleTag } from '../../lib/i18n';

  export let data: EndpointLatency[] = [];

  function formatMs(ms: number): string {
    if (ms >= 1000) return (ms / 1000).toFixed(2) + "s";
    return ms.toFixed(0) + "ms";
  }

  function getLatencyColor(ms: number): string {
    if (ms <= 500) return "var(--color-success-500)";
    if (ms <= 2000) return "#eab308";
    if (ms <= 5000) return "#f97316";
    return "var(--color-danger-500)";
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
                <MarqueeText text={row.endpoint} class="endpoint-name" />
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
    width: 30%;
    max-width: 0;
  }

  :global(.endpoint-name) {
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
    background: var(--color-slate-100);
    border-radius: 4px;
    font-size: 11px;
    font-weight: 500;
    color: var(--color-slate-600);
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
    color: var(--color-slate-500);
    font-size: 11px;
  }

  th.col-latency,
  th.col-count {
    text-align: right;
  }
</style>
