<script lang="ts">
  import CorrelationPanel from '../../../components/CorrelationPanel.svelte';
  export let params;
</script>

<h2>Trace Detail</h2>
<p>Trace ID: {params.traceId}</p>
<CorrelationPanel traceId={params.traceId} />
