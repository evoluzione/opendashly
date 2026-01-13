<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchRelated } from '../services/traces';

  export let traceId: string;
  let related: { logs: any[]; metrics: any[] } | null = null;
  let error: string | null = null;

  onMount(async () => {
    try {
      related = await fetchRelated(traceId);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load related telemetry';
    }
  });
</script>

<section>
  <h3>Related Telemetry</h3>
  {#if error}
    <p class="error">{error}</p>
  {:else if !related}
    <p>Loading...</p>
  {:else}
    <div>
      <h4>Logs</h4>
      <ul>
        {#each related.logs as log}
          <li>{log.body}</li>
        {/each}
      </ul>
    </div>
    <div>
      <h4>Metrics</h4>
      <ul>
        {#each related.metrics as metric}
          <li>{metric.name}</li>
        {/each}
      </ul>
    </div>
  {/if}
</section>

<style>
  .error {
    color: #b91c1c;
  }
</style>
