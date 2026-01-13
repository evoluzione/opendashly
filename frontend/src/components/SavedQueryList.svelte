<script lang="ts">
  import { onMount } from 'svelte';
  import { listSavedQueries, deleteQuery, runSavedQuery } from '../services/saved_queries';

  let items: any[] = [];
  let error: string | null = null;

  async function load() {
    try {
      const result = await listSavedQueries();
      items = result.items;
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load saved queries';
    }
  }

  async function remove(id: string) {
    await deleteQuery(id);
    await load();
  }

  async function run(id: string) {
    await runSavedQuery(id);
  }

  onMount(load);
</script>

{#if error}
  <p class="error">{error}</p>
{:else}
  <ul>
    {#each items as item}
      <li>
        <strong>{item.name}</strong>
        <button on:click={() => run(item.id)}>Run</button>
        <button on:click={() => remove(item.id)}>Delete</button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  .error {
    color: #b91c1c;
  }
  ul {
    list-style: none;
    padding: 0;
  }
  li {
    margin-bottom: 8px;
  }
</style>
