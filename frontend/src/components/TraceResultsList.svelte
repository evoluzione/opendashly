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
      Precedente
    </button>
    <span>Pagina {pagination.page} di {pagination.totalPages || 1}</span>
    <button
      type="button"
      on:click={() => changePage(pagination.page + 1)}
      disabled={pagination.totalPages > 0 ? pagination.page >= pagination.totalPages : traces.length === 0}
    >
      Successiva
    </button>
  </div>
{/if}

<style>
  .trace-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  
  li {
    padding: 16px 20px;
    background: #f8fafc;
    border-radius: 12px;
    border: 1px solid #e2e8f0;
    transition: all 0.2s ease;
  }
  
  li:hover {
    border-color: #6366f1;
    background: rgba(99, 102, 241, 0.03);
    transform: translateX(4px);
  }
  
  li a {
    color: #334155;
    text-decoration: none;
    font-weight: 500;
    font-size: 14px;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  li a::before {
    content: '';
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  }
  
  li:hover a {
    color: #6366f1;
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
  }
</style>
