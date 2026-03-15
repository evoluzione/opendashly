<script lang="ts">
  import type { ThroughputSummary } from '../../services/dashboard';
  import InfoTooltip from '../common/InfoTooltip.svelte';
  import { locale, t } from '../../lib/i18n';

  export let data: ThroughputSummary | null = null;

  function formatNumber(num: number): string {
    if (num === 0) return '0.0';
    if (num < 0.1) return num.toFixed(3);
    if (num < 1) return num.toFixed(2);
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'k';
    return num.toFixed(1);
  }

  $: reqPerMin = data?.requestsPerMin ?? 0;
  $: errPerMin = data?.errorsPerMin ?? 0;
  $: totalReq = data?.totalRequests ?? 0;
</script>

<div class="gauge-card">
  <div class="gauge-header">
    <span class="gauge-title">
      Throughput
      <InfoTooltip text="Numero di richieste elaborate al minuto. Indica il carico di lavoro del sistema e la sua capacità di gestire le richieste." position="bottom" />
    </span>
  </div>

  <div class="main-stat">
    <span class="value">{formatNumber(reqPerMin)}</span>
    <span class="unit">{t($locale, 'dashboard.throughput.unitReqPerMin')}</span>
  </div>

  <div class="secondary-stats">
    <div class="stat">
      <span class="stat-value">{formatNumber(errPerMin)}</span>
      <span class="stat-label">{t($locale, 'dashboard.throughput.errPerMin')}</span>
    </div>
    <div class="divider"></div>
    <div class="stat">
      <span class="stat-value">{totalReq.toLocaleString()}</span>
      <span class="stat-label">{t($locale, 'dashboard.throughput.total')}</span>
    </div>
  </div>
</div>

<style>
  .gauge-card {
    background: white;
    border-radius: 16px;
    padding: 20px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 4px 12px rgba(0, 0, 0, 0.03);
    border: 1px solid rgba(15, 23, 42, 0.06);
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-height: 0;
    container-type: inline-size;
  }

  .gauge-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .gauge-title {
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #64748b;
  }

  .main-stat {
    text-align: center;
    padding: clamp(8px, 3cqw, 24px) 0;
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .main-stat .value {
    display: block;
    font-size: clamp(26px, 9cqw, 36px);
    font-weight: 700;
    color: #2563eb;
    line-height: 1.1;
  }

  .main-stat .unit {
    display: block;
    font-size: clamp(10px, 3cqw, 12px);
    font-weight: 600;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-top: 4px;
  }

  .secondary-stats {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: clamp(10px, 3cqw, 16px);
    padding-top: 16px;
    border-top: 1px solid #f1f5f9;
  }

  .stat {
    text-align: center;
  }

  .stat-value {
    display: block;
    font-size: clamp(14px, 4.4cqw, 16px);
    font-weight: 700;
    color: #0f172a;
    font-variant-numeric: tabular-nums;
  }

  .stat-label {
    display: block;
    font-size: clamp(9px, 2.8cqw, 10px);
    color: #94a3b8;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-top: 2px;
  }

  .divider {
    width: 1px;
    height: 32px;
    background: #e2e8f0;
  }
</style>
