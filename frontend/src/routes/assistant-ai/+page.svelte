<script lang="ts">
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";

  onMount(() => {
    void goto("/?tab=metriche", { replaceState: true });
  });
</script>
