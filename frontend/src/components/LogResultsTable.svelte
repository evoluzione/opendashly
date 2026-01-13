<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let logs: any[] = [];
  export let pagination: { page: number; totalPages: number } | null = null;

  const dispatch = createEventDispatcher();

  function changePage(nextPage: number) {
    dispatch('pageChange', { page: nextPage });
  }
</script>

<table>
  <thead>
    <tr>
      <th>Timestamp</th>
      <th>Severity</th>
      <th>Body</th>
    </tr>
  </thead>
  <tbody>
    {#each logs as log}
      <tr>
        <td>{log.timestamp}</td>
        <td>{log.severity}</td>
        <td>{log.body}</td>
      </tr>
    {/each}
  </tbody>
</table>

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
      disabled={pagination.totalPages > 0 ? pagination.page >= pagination.totalPages : logs.length === 0}
    >
      Next
    </button>
  </div>
{/if}

<style>
  table {
    width: 100%;
    border-collapse: collapse;
  }
  th, td {
    border-bottom: 1px solid #e5e5e5;
    padding: 8px;
    text-align: left;
  }
  .pager {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
  }
</style>
