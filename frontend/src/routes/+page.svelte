<script lang="ts">
  export let data;
</script>

<section class="hero">
  <h2>Welcome</h2>
  <p>Start with the query page or browse saved queries.</p>
  <div class="actions">
    <a class="btn" href="/query">Go to Query</a>
    <a class="btn secondary" href="/queries">Saved Queries</a>
  </div>
</section>

<style>
  .hero {
    padding: 32px;
    background: #f4f4f0;
    border-radius: 12px;
  }
  .actions {
    display: flex;
    gap: 12px;
    margin-top: 16px;
  }
  .btn {
    display: inline-block;
    padding: 10px 16px;
    background: #111;
    color: #fff;
    text-decoration: none;
    border-radius: 8px;
  }
  .btn.secondary {
    background: #e5e5e5;
    color: #111;
  }
</style>
