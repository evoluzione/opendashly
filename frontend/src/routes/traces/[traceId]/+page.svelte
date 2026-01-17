<script lang="ts">
  import { page } from '$app/stores';
  import CorrelationPanel from '../../../components/CorrelationPanel.svelte';

  export let params: Record<string, string> = {};
</script>

<section class="trace-detail">
  <header>
    <div>
      <p class="kicker">Dettaglio traccia</p>
      <h2>{$page.params.traceId}</h2>
    </div>
    <a class="back" href="/">Torna alla dashboard</a>
  </header>
  <CorrelationPanel traceId={$page.params.traceId} />
</section>

<style>
  .trace-detail {
    padding: 28px;
    display: flex;
    flex-direction: column;
    gap: 24px;
    background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
    min-height: 100vh;
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
  }

  .kicker {
    margin: 0 0 6px 0;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #94a3b8;
  }

  h2 {
    margin: 0;
    font-size: 20px;
    color: #0f172a;
    word-break: break-all;
  }

  .back {
    padding: 10px 16px;
    border-radius: 999px;
    background: #1d4ed8;
    color: white;
    text-decoration: none;
    font-weight: 600;
    font-size: 12px;
  }
</style>
