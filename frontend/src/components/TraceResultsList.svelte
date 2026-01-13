<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let traces: any[] = [];
  export let pagination: { page: number; totalPages: number } | null = null;

  const dispatch = createEventDispatcher();

  function changePage(nextPage: number) {
    dispatch('pageChange', { page: nextPage });
  }
</script>

<ul class="trace-list">
  {#each traces as trace}
    <li>
      <a href={`/traces/${trace.traceId}`}>{trace.name} ({trace.traceId})</a>
    </li>
  {/each}
</ul>

{#if pagination}
  <div class="pager">
    <button
      type="button"
      on:click={() => changePage(pagination.page - 1)}
      disabled={pagination.page <= 1}
    >
      Previous
    </button>
    <span>Page {pagination.page} of {pagination.totalPages || 1}</span>
    <button
      type="button"
      on:click={() => changePage(pagination.page + 1)}
      disabled={pagination.totalPages > 0 ? pagination.page >= pagination.totalPages : traces.length === 0}
    >
      Next
    </button>
  </div>
{/if}

<style>
  .trace-list {
    list-style: none;
    padding: 0;
  }
  li {
    margin-bottom: 8px;
  }
  .pager {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
  }
</style>
