<script lang="ts">
  import { goto } from '$app/navigation';
  import { authState, changePasswordForUser } from '../../lib/stores/auth';

  export let params: Record<string, string> = {};

  let currentPassword = '';
  let newPassword = '';

  async function submit() {
    const session = await changePasswordForUser(currentPassword, newPassword);
    if (!session) {
      return;
    }
    await goto('/');
  }
</script>

<section class="login">
  <div class="card">
    <h1>Cambia password</h1>
    <p>Per sicurezza devi cambiare la password al primo accesso.</p>

    <div class="field">
      <label for="current">Password attuale</label>
      <input id="current" type="password" bind:value={currentPassword} />
    </div>
    <div class="field">
      <label for="new">Nuova password</label>
      <input id="new" type="password" bind:value={newPassword} />
    </div>

    {#if $authState.error}
      <div class="error">{$authState.error}</div>
    {/if}

    <button on:click={submit} disabled={$authState.loading}>Aggiorna password</button>
  </div>
</section>

<style>
  .login {
    min-height: 100vh;
    display: grid;
    place-items: center;
    background: radial-gradient(circle at top, #f8fafc 0%, #e2e8f0 60%);
    padding: 24px;
  }
  .card {
    background: #fff;
    border-radius: 20px;
    padding: 32px;
    width: min(420px, 100%);
    box-shadow: 0 20px 60px rgba(15, 23, 42, 0.12);
    border: 1px solid rgba(15, 23, 42, 0.08);
  }
  h1 {
    margin: 0 0 8px 0;
  }
  .field {
    display: grid;
    gap: 6px;
    margin-top: 16px;
  }
  label {
    font-size: 12px;
    text-transform: uppercase;
  }
  input {
    padding: 12px 14px;
    border-radius: 12px;
    border: 1px solid rgba(15, 23, 42, 0.12);
  }
  button {
    margin-top: 20px;
    width: 100%;
    padding: 12px 16px;
    border-radius: 12px;
    border: none;
    background: #0f172a;
    color: #fff;
    font-weight: 600;
    cursor: pointer;
  }
  .error {
    color: #b42318;
    margin-top: 12px;
  }
</style>
