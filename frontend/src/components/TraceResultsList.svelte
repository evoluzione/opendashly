<script lang="ts">
  export let traces: any[] = [];
</script>

<ul class="trace-list">
  {#each traces as trace}
    <li>
      <a href={`/traces/${trace.traceId}`}>{trace.name} ({trace.traceId})</a>
    </li>
  {/each}
</ul>

<style>
  .trace-list {
    list-style: none;
    padding: 0;
  }
  li {
    margin-bottom: 8px;
  }
</style>
