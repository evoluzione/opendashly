<script lang="ts">
  import SavedQueryList from '../../components/SavedQueryList.svelte';

  export let params: Record<string, string> = {};
</script>

<h2>Query salvate</h2>
<SavedQueryList />
