<script lang="ts">
  export let logs: any[] = [];
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
</style>
