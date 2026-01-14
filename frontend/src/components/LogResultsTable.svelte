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
      <th>Data/Ora</th>
      <th>Severità</th>
      <th>Contenuto</th>
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
      Precedente
    </button>
    <span>Pagina {pagination.page} di {pagination.totalPages || 1}</span>
    <button
      type="button"
      on:click={() => changePage(pagination.page + 1)}
      disabled={pagination.totalPages > 0 ? pagination.page >= pagination.totalPages : logs.length === 0}
    >
      Successiva
    </button>
  </div>
{/if}

<style>
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }
  
  th {
    padding: 12px 16px;
    text-align: left;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
    background: #f8fafc;
    border-bottom: 2px solid #e2e8f0;
  }
  
  td {
    padding: 14px 16px;
    border-bottom: 1px solid #f1f5f9;
    color: #334155;
    transition: background 0.15s ease;
  }
  
  tr:hover td {
    background: rgba(99, 102, 241, 0.03);
  }
  
  .pager {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    margin-top: 20px;
    padding-top: 20px;
    border-top: 1px solid #f1f5f9;
  }
  
  .pager button {
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 500;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    background: white;
    color: #475569;
    cursor: pointer;
    transition: all 0.2s ease;
  }
  
  .pager button:hover:not(:disabled) {
    border-color: #6366f1;
    color: #6366f1;
  }
  
  .pager button:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  
  .pager span {
    font-size: 13px;
    color: #64748b;
    padding: 0 8px;
  }
</style>
