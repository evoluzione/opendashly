<script lang="ts">
  import type { EndpointThroughput } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';
  import { locale, t, getLocaleTag } from '../../lib/i18n';

  export let data: EndpointThroughput[] = [];

  type Row = { service: string; requests: number; errors: number };

  $: ranked = (() => {
    const map = new Map<string, { requests: number; errors: number }>();
    for (const row of data) {
      const cur = map.get(row.service) ?? { requests: 0, errors: 0 };
      cur.requests += row.requestCount;
      cur.errors += row.errorCount;
      map.set(row.service, cur);
    }

    const out: Row[] = [];
    for (const [service, agg] of map.entries()) {
      out.push({ service, requests: agg.requests, errors: agg.errors });
    }

    return out.sort((a, b) => b.requests - a.requests).slice(0, 8);
  })();

  $: maxReq = Math.max(...ranked.map((r) => r.requests), 1);
</script>

<div class="table-card">
  <div class="table-header">
    <span class="table-title">
      {t($locale, 'dashboard.serviceThroughput.title')}
      <InfoTooltip text={t($locale, 'dashboard.serviceThroughput.tooltip')} />
    </span>
    <span class="table-subtitle">{t($locale, 'dashboard.serviceThroughput.subtitle')}</span>
  </div>

  {#if ranked.length === 0}
    <div class="empty">{t($locale, 'dashboard.noData')}</div>
  {:else}
    <div class="rows">
      {#each ranked as row}
        <div class="row">
          <div class="service">{row.service}</div>
          <div class="bar-bg">
            <span class="bar" style={`width:${(row.requests / maxReq) * 100}%`}></span>
          </div>
          <div class="value">
            <span>{row.requests.toLocaleString(getLocaleTag($locale))}</span>
            <small>{t($locale, 'dashboard.serviceThroughput.errorsShort', { count: row.errors.toLocaleString(getLocaleTag($locale)) })}</small>
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

  .table-header { margin-bottom: 12px; }
  .table-title { display: block; font-size: 14px; font-weight: 600; color: #0f172a; }
  .table-subtitle { display: block; font-size: 12px; color: #94a3b8; margin-top: 2px; }
  .empty { text-align: center; color: #94a3b8; padding: 40px 0; font-size: 14px; }

  .rows { display: flex; flex-direction: column; gap: 8px; }

  .row {
    display: grid;
    grid-template-columns: minmax(90px, 1fr) minmax(120px, 3fr) auto;
    gap: 10px;
    align-items: center;
  }

  .service {
    font-size: 12px;
    color: #334155;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .bar-bg {
    height: 8px;
    background: #e2e8f0;
    border-radius: 999px;
    overflow: hidden;
  }

  .bar {
    display: block;
    height: 100%;
    border-radius: 999px;
    background: linear-gradient(90deg, #2563eb 0%, #06b6d4 100%);
  }

  .value {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    line-height: 1.1;
  }

  .value span {
    font-size: 12px;
    font-weight: 700;
    color: #0f172a;
    font-variant-numeric: tabular-nums;
  }

  .value small {
    font-size: 10px;
    color: #ef4444;
    font-weight: 700;
  }
</style>
